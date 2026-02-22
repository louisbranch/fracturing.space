-- +migrate Up

CREATE TABLE auth_integration_outbox (
    id TEXT PRIMARY KEY,
    event_type TEXT NOT NULL,
    payload_json TEXT NOT NULL DEFAULT '{}',
    dedupe_key TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    next_attempt_at INTEGER NOT NULL,
    lease_owner TEXT NOT NULL DEFAULT '',
    lease_expires_at INTEGER,
    last_error TEXT NOT NULL DEFAULT '',
    processed_at INTEGER,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE UNIQUE INDEX auth_integration_outbox_dedupe_unique_idx
ON auth_integration_outbox(dedupe_key)
WHERE dedupe_key <> '';

CREATE INDEX auth_integration_outbox_lease_idx
ON auth_integration_outbox(status, next_attempt_at, lease_expires_at, id);

-- +migrate Down
DROP INDEX IF EXISTS auth_integration_outbox_lease_idx;
DROP INDEX IF EXISTS auth_integration_outbox_dedupe_unique_idx;
DROP TABLE IF EXISTS auth_integration_outbox;
