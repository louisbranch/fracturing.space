-- +migrate Up

DROP TABLE IF EXISTS user_emails;
DROP TABLE IF EXISTS magic_links;

CREATE TABLE user_emails (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email TEXT NOT NULL UNIQUE,
    verified_at INTEGER,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE magic_links (
    token TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    pending_id TEXT,
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    used_at INTEGER
);

-- +migrate Down
DROP TABLE IF EXISTS magic_links;
DROP TABLE IF EXISTS user_emails;
