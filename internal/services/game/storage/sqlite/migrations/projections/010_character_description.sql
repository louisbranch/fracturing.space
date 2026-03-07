-- +migrate Up
ALTER TABLE daggerheart_character_profiles ADD COLUMN description TEXT NOT NULL DEFAULT '';
