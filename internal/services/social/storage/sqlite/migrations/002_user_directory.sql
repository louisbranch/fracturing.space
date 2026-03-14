-- +migrate Up

CREATE TABLE user_directory (
    user_id TEXT NOT NULL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS user_directory_username_idx
ON user_directory(username);

-- +migrate Down

DROP INDEX IF EXISTS user_directory_username_idx;
DROP TABLE IF EXISTS user_directory;
