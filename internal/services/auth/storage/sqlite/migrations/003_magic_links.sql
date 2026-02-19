-- +migrate Up

DROP TABLE IF EXISTS account_profiles;
DROP TABLE IF EXISTS user_emails;
DROP TABLE IF EXISTS magic_links;

CREATE TABLE account_profiles (
    user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    locale TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE user_emails (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email TEXT NOT NULL UNIQUE,
    is_primary INTEGER NOT NULL DEFAULT 0,
    verified_at INTEGER,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS user_emails_primary_idx
ON user_emails(user_id)
WHERE is_primary = 1;

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
DROP TABLE IF EXISTS account_profiles;
