-- sipplane control-plane schema (RFC 0003)
-- Applied by store.Migrate

CREATE TABLE IF NOT EXISTS config_meta (
    id INT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    revision BIGINT NOT NULL DEFAULT 0
);

INSERT INTO config_meta (id, revision) VALUES (1, 0)
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS config_revisions (
    revision BIGSERIAL PRIMARY KEY,
    actor TEXT NOT NULL,
    snapshot JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS audit_log (
    id BIGSERIAL PRIMARY KEY,
    revision BIGINT NOT NULL,
    action TEXT NOT NULL,
    actor TEXT NOT NULL,
    payload JSONB,
    at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS audit_log_revision_idx ON audit_log (revision DESC);
