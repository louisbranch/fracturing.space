-- +migrate Up

ALTER TABLE projection_watermarks
ADD COLUMN expected_next_seq INTEGER NOT NULL DEFAULT 0;

-- +migrate Down

-- SQLite does not support dropping columns in place.
