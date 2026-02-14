-- +migrate Up
ALTER TABLE daggerheart_character_profiles ADD COLUMN proficiency INTEGER NOT NULL DEFAULT 0;
ALTER TABLE daggerheart_character_profiles ADD COLUMN armor_score INTEGER NOT NULL DEFAULT 0;
ALTER TABLE daggerheart_character_profiles ADD COLUMN armor_max INTEGER NOT NULL DEFAULT 0;
ALTER TABLE daggerheart_character_profiles ADD COLUMN experiences_json TEXT NOT NULL DEFAULT '[]';

ALTER TABLE daggerheart_character_states ADD COLUMN armor INTEGER NOT NULL DEFAULT 0;

-- +migrate Down
