-- +migrate Up
ALTER TABLE daggerheart_character_states ADD COLUMN stat_modifiers_json TEXT NOT NULL DEFAULT '[]';

-- +migrate Down
