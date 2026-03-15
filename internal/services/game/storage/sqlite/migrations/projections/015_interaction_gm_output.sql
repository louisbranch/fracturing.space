-- +migrate Up

ALTER TABLE scene_interactions ADD COLUMN gm_output_text TEXT NOT NULL DEFAULT '';
ALTER TABLE scene_interactions ADD COLUMN gm_output_participant_id TEXT NOT NULL DEFAULT '';
ALTER TABLE scene_interactions ADD COLUMN gm_output_updated_at INTEGER;

-- +migrate Down

-- SQLite does not support dropping columns in-place. This migration is
-- intentionally irreversible because projection schemas are rebuilt from
-- event history in non-production environments for this project.
