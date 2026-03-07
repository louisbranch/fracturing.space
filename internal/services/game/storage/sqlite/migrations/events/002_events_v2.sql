-- +migrate Up

ALTER TABLE events ADD COLUMN scene_id TEXT NOT NULL DEFAULT '';
ALTER TABLE events ADD COLUMN correlation_id TEXT NOT NULL DEFAULT '';
ALTER TABLE events ADD COLUMN causation_id TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_events_scene ON events(campaign_id, scene_id)
    WHERE scene_id != '';

-- +migrate Down
DROP INDEX IF EXISTS idx_events_scene;
