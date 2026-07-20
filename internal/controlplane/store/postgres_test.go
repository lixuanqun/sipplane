package store

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestPostgresApplyWatchAudit(t *testing.T) {
	dsn := os.Getenv("SIPPLANE_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("set SIPPLANE_DATABASE_URL to run Postgres store tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	p, err := OpenPostgres(ctx, dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer p.Close()

	// Reset meta for isolated test run (best-effort).
	_, _ = p.pool.Exec(ctx, `UPDATE config_meta SET revision = 0 WHERE id = 1`)

	yaml := []byte(`
apiVersion: sipplane.io/v1alpha1
kind: Route
metadata:
  name: pg-route
  tenant: acme
spec:
  priority: 10
  match:
    methods: ["INVITE"]
  action:
    type: registerLookup
`)

	rev0, err := p.Revision(ctx)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Apply(ctx, "test", yaml, true)
	if err != nil {
		t.Fatal(err)
	}
	revAfterDry, _ := p.Revision(ctx)
	if revAfterDry != rev0 {
		t.Fatalf("dry-run bumped revision: %d -> %d", rev0, revAfterDry)
	}

	done := make(chan int64, 1)
	go func() {
		wctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		r, err := p.Watch(wctx, rev0)
		if err != nil {
			t.Errorf("watch: %v", err)
			return
		}
		done <- r
	}()
	time.Sleep(200 * time.Millisecond)

	rev, err := p.Apply(ctx, "tester", yaml, false)
	if err != nil {
		t.Fatal(err)
	}
	if rev <= rev0 {
		t.Fatalf("expected revision > %d, got %d", rev0, rev)
	}

	select {
	case got := <-done:
		if got < rev {
			t.Fatalf("watch got %d want >= %d", got, rev)
		}
	case <-time.After(8 * time.Second):
		t.Fatal("watch timeout")
	}

	snap, err := p.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Revision != rev || len(snap.Routes) != 1 {
		t.Fatalf("snapshot=%+v", snap)
	}

	audit, err := p.Audit(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(audit) == 0 {
		t.Fatal("expected audit entries")
	}

	bad := []byte(`kind: Route
metadata: {name: bad}
spec:
  action: {type: nope}
`)
	_, err = p.Apply(ctx, "tester", bad, false)
	if err == nil {
		t.Fatal("expected validation error")
	}
}
