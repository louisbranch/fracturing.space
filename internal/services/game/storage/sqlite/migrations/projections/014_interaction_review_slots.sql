-- +migrate Up

ALTER TABLE scene_interactions ADD COLUMN phase_status TEXT NOT NULL DEFAULT '';
ALTER TABLE scene_interactions ADD COLUMN slots_json BLOB;

-- +migrate Down

-- SQLite does not support dropping columns in-place. This migration is
-- intentionally irreversible because projection schemas are rebuilt from
-- event history in non-production environments for this project.
