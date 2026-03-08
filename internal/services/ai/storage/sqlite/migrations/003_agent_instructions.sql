-- +migrate Up

ALTER TABLE ai_agents ADD COLUMN instructions TEXT NOT NULL DEFAULT '';

-- +migrate Down
