package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sipplane/sipplane/internal/resources"
)

const notifyChannel = "sipplane_config"

// Postgres is the production config store (RFC 0003).
type Postgres struct {
	pool *pgxpool.Pool
}

// OpenPostgres connects and runs migrations.
func OpenPostgres(ctx context.Context, databaseURL string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("postgres connect: %w", err)
	}
	p := &Postgres{pool: pool}
	if err := p.Migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return p, nil
}

// Close closes the pool.
func (p *Postgres) Close() {
	if p.pool != nil {
		p.pool.Close()
	}
}

// Migrate applies embedded SQL schema.
func (p *Postgres) Migrate(ctx context.Context) error {
	_, err := p.pool.Exec(ctx, schemaSQL)
	return err
}

func (p *Postgres) Snapshot(ctx context.Context) (*resources.Snapshot, error) {
	var rev int64
	err := p.pool.QueryRow(ctx, `SELECT revision FROM config_meta WHERE id = 1`).Scan(&rev)
	if err != nil {
		return nil, err
	}
	if rev == 0 {
		return &resources.Snapshot{
			Revision:  0,
			Tenants:   map[string]*resources.Tenant{},
			Trunks:    map[string]*resources.Trunk{},
			Secrets:   map[string]string{},
			Endpoints: nil,
			Routes:    nil,
		}, nil
	}
	var raw []byte
	err = p.pool.QueryRow(ctx,
		`SELECT snapshot FROM config_revisions WHERE revision = $1`, rev,
	).Scan(&raw)
	if err != nil {
		return nil, err
	}
	var snap resources.Snapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		return nil, err
	}
	snap.Revision = rev
	if snap.Tenants == nil {
		snap.Tenants = map[string]*resources.Tenant{}
	}
	if snap.Trunks == nil {
		snap.Trunks = map[string]*resources.Trunk{}
	}
	if snap.Secrets == nil {
		snap.Secrets = map[string]string{}
	}
	return &snap, nil
}

func (p *Postgres) Revision(ctx context.Context) (int64, error) {
	var rev int64
	err := p.pool.QueryRow(ctx, `SELECT revision FROM config_meta WHERE id = 1`).Scan(&rev)
	return rev, err
}

func (p *Postgres) Apply(ctx context.Context, actor string, docs []byte, dryRun bool) (int64, error) {
	next, err := resources.ParseYAML(docs)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if err := ValidateSnapshot(next); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrValidation, err)
	}

	cur, err := p.Snapshot(ctx)
	if err != nil {
		return 0, err
	}
	if dryRun {
		return cur.Revision, nil
	}

	// Merge secrets from previous if not re-specified.
	if cur != nil {
		for k, v := range cur.Secrets {
			if _, ok := next.Secrets[k]; !ok {
				next.Secrets[k] = v
			}
		}
	}

	payload, err := json.Marshal(next)
	if err != nil {
		return 0, err
	}
	h := fnv.New64a()
	_, _ = h.Write(payload)
	hash := strconv.FormatUint(h.Sum64(), 16)

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var rev int64
	err = tx.QueryRow(ctx,
		`INSERT INTO config_revisions (actor, snapshot) VALUES ($1, $2) RETURNING revision`,
		actor, payload,
	).Scan(&rev)
	if err != nil {
		return 0, err
	}
	next.Revision = rev
	// Re-marshal with revision set.
	payload, err = json.Marshal(next)
	if err != nil {
		return 0, err
	}
	_, err = tx.Exec(ctx,
		`UPDATE config_revisions SET snapshot = $1 WHERE revision = $2`,
		payload, rev,
	)
	if err != nil {
		return 0, err
	}
	_, err = tx.Exec(ctx, `UPDATE config_meta SET revision = $1 WHERE id = 1`, rev)
	if err != nil {
		return 0, err
	}
	_, err = tx.Exec(ctx,
		`INSERT INTO audit_log (revision, action, actor, payload) VALUES ($1, $2, $3, $4)`,
		rev, "apply", actor, json.RawMessage(fmt.Sprintf(`{"hash":%q}`, hash)),
	)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}

	// NOTIFY outside transaction is fine; listeners may poll as fallback.
	_, _ = p.pool.Exec(ctx, `SELECT pg_notify($1, $2)`, notifyChannel, strconv.FormatInt(rev, 10))
	return rev, nil
}

func (p *Postgres) Watch(ctx context.Context, since int64) (int64, error) {
	// Fast path: already newer.
	rev, err := p.Revision(ctx)
	if err != nil {
		return 0, err
	}
	if rev > since {
		return rev, nil
	}

	conn, err := p.pool.Acquire(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Release()

	_, err = conn.Exec(ctx, `LISTEN `+notifyChannel)
	if err != nil {
		return 0, err
	}
	defer func() {
		_, _ = conn.Exec(context.Background(), `UNLISTEN `+notifyChannel)
	}()

	for {
		// Re-check in case we missed a notify between Revision() and LISTEN.
		rev, err = p.Revision(ctx)
		if err != nil {
			return 0, err
		}
		if rev > since {
			return rev, nil
		}

		notification, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return 0, err
			}
			return 0, err
		}
		if notification == nil {
			continue
		}
		n, _ := strconv.ParseInt(notification.Payload, 10, 64)
		if n > since {
			return n, nil
		}
	}
}

func (p *Postgres) Audit(ctx context.Context, limit int) ([]AuditEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := p.pool.Query(ctx,
		`SELECT revision, action, actor, at FROM audit_log ORDER BY id DESC LIMIT $1`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.Revision, &e.Action, &e.Actor, &e.At); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	// Return chronological order (oldest first among the limited set).
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, rows.Err()
}

// Ensure pgx types referenced for compile when unused imports shift.
var (
	_ = pgx.ErrNoRows
	_ = time.Second
)
