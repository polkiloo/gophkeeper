CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    login TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS records (
    id TEXT PRIMARY KEY,
    owner_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    payload JSONB NOT NULL,
    meta JSONB NOT NULL,
    version BIGINT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    deleted BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_records_owner_updated ON records(owner_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_records_owner_type ON records(owner_id, type);

CREATE TABLE IF NOT EXISTS change_log (
    id BIGSERIAL PRIMARY KEY,
    record_id TEXT NOT NULL,
    owner_id TEXT NOT NULL,
    type TEXT NOT NULL,
    change TEXT NOT NULL,
    payload JSONB,
    meta JSONB NOT NULL,
    version BIGINT NOT NULL,
    happened_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_change_log_owner_id ON change_log(owner_id, id);
