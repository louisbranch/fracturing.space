-- +migrate Up

CREATE TABLE IF NOT EXISTS game_integration_outbox (
    id TEXT PRIMARY KEY,
    event_type TEXT NOT NULL,
    payload_json TEXT NOT NULL,
    dedupe_key TEXT NOT NULL,
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

CREATE UNIQUE INDEX IF NOT EXISTS idx_game_integration_outbox_dedupe
    ON game_integration_outbox (dedupe_key);

CREATE INDEX IF NOT EXISTS idx_game_integration_outbox_lease
    ON game_integration_outbox (status, next_attempt_at, lease_expires_at, id);

-- +migrate Down
DROP INDEX IF EXISTS idx_game_integration_outbox_lease;
DROP INDEX IF EXISTS idx_game_integration_outbox_dedupe;
DROP TABLE IF EXISTS game_integration_outbox;
