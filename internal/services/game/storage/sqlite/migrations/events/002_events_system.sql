-- +migrate Up
ALTER TABLE events ADD COLUMN system_id TEXT NOT NULL DEFAULT '';
ALTER TABLE events ADD COLUMN system_version TEXT NOT NULL DEFAULT '';

CREATE INDEX idx_events_system ON events(campaign_id, system_id, system_version)
    WHERE system_id != '';

-- +migrate Down
DROP INDEX IF EXISTS idx_events_system;
