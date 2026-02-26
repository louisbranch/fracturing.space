-- +migrate Up

CREATE TABLE web_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    revoked_at INTEGER
);

CREATE INDEX web_sessions_user_id_idx ON web_sessions(user_id);
CREATE INDEX web_sessions_expires_at_idx ON web_sessions(expires_at);

-- +migrate Down
DROP INDEX IF EXISTS web_sessions_expires_at_idx;
DROP INDEX IF EXISTS web_sessions_user_id_idx;
DROP TABLE IF EXISTS web_sessions;
