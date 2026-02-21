-- +migrate Up

CREATE TABLE IF NOT EXISTS web_sessions (
    session_id TEXT PRIMARY KEY,
    access_token_hash TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    expires_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_web_sessions_expires_at
    ON web_sessions (expires_at);

-- +migrate Down

DROP INDEX IF EXISTS idx_web_sessions_expires_at;
DROP TABLE IF EXISTS web_sessions;
