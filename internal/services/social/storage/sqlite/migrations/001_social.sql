-- Baseline schema for fresh alpha databases.

CREATE TABLE contacts (
    owner_user_id TEXT NOT NULL,
    contact_user_id TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (owner_user_id, contact_user_id),
    CHECK (owner_user_id <> contact_user_id)
);
CREATE INDEX contacts_contact_user_idx
ON contacts(contact_user_id);
CREATE TABLE user_profiles (
    user_id TEXT NOT NULL PRIMARY KEY,
    name TEXT NOT NULL DEFAULT '',
    avatar_set_id TEXT NOT NULL DEFAULT '',
    avatar_asset_id TEXT NOT NULL DEFAULT '',
    bio TEXT NOT NULL DEFAULT '',
    pronouns TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
CREATE TABLE user_directory (
    user_id TEXT NOT NULL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
CREATE INDEX user_directory_username_idx
ON user_directory(username);
