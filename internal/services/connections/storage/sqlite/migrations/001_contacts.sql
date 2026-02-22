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

-- +migrate Down

DROP INDEX IF EXISTS contacts_contact_user_idx;
DROP TABLE IF EXISTS contacts;
