-- +migrate Up
ALTER TABLE daggerheart_adversaries ADD COLUMN feature_state_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE daggerheart_adversaries ADD COLUMN pending_experience_json TEXT NOT NULL DEFAULT '';

-- +migrate Down
