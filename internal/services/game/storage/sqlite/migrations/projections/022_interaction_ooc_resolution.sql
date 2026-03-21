-- +migrate Up

ALTER TABLE session_interactions ADD COLUMN ooc_requested_by_participant_id TEXT NOT NULL DEFAULT '';
ALTER TABLE session_interactions ADD COLUMN ooc_reason TEXT NOT NULL DEFAULT '';
ALTER TABLE session_interactions ADD COLUMN ooc_interrupted_scene_id TEXT NOT NULL DEFAULT '';
ALTER TABLE session_interactions ADD COLUMN ooc_interrupted_phase_id TEXT NOT NULL DEFAULT '';
ALTER TABLE session_interactions ADD COLUMN ooc_interrupted_phase_status TEXT NOT NULL DEFAULT '';
ALTER TABLE session_interactions ADD COLUMN ooc_resolution_pending INTEGER NOT NULL DEFAULT 0;

-- +migrate Down

