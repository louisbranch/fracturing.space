-- +migrate Up
ALTER TABLE daggerheart_snapshots ADD COLUMN consecutive_short_rests INTEGER NOT NULL DEFAULT 0;

-- +migrate Down
