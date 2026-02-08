-- +migrate Up

ALTER TABLE participants
    ADD COLUMN is_owner INTEGER NOT NULL DEFAULT 0;

-- +migrate Down
-- SQLite does not support dropping columns; noop.
