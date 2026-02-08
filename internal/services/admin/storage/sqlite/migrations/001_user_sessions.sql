-- +migrate Up

CREATE TABLE user_sessions (
    session_id TEXT PRIMARY KEY,
    created_at TEXT NOT NULL
);

-- +migrate Down
DROP TABLE IF EXISTS user_sessions;
