-- +migrate Up

CREATE TABLE user_contacts (
    owner_user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    contact_user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (owner_user_id, contact_user_id),
    CHECK (owner_user_id <> contact_user_id)
);

CREATE INDEX IF NOT EXISTS user_contacts_contact_user_idx
ON user_contacts(contact_user_id);

-- +migrate Down

DROP INDEX IF EXISTS user_contacts_contact_user_idx;
DROP TABLE IF EXISTS user_contacts;
