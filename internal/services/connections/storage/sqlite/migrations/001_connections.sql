-- +migrate Up

CREATE TABLE contacts (
    owner_user_id TEXT NOT NULL,
    contact_user_id TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (owner_user_id, contact_user_id),
    CHECK (owner_user_id <> contact_user_id)
);

CREATE INDEX IF NOT EXISTS contacts_contact_user_idx
ON contacts(contact_user_id);

CREATE TABLE user_profiles (
    user_id TEXT NOT NULL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    avatar_set_id TEXT NOT NULL DEFAULT '',
    avatar_asset_id TEXT NOT NULL DEFAULT '',
    bio TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- +migrate Down

DROP TABLE IF EXISTS user_profiles;
DROP INDEX IF EXISTS contacts_contact_user_idx;
DROP TABLE IF EXISTS contacts;
