-- +migrate Up
ALTER TABLE daggerheart_character_states
    ADD COLUMN subclass_state_json TEXT NOT NULL DEFAULT '{}';

-- +migrate Down

DROP TABLE IF EXISTS daggerheart_character_states;

CREATE TABLE IF NOT EXISTS daggerheart_character_states (
    campaign_id TEXT NOT NULL,
    character_id TEXT NOT NULL,
    hp INTEGER NOT NULL,
    hope INTEGER NOT NULL,
    hope_max INTEGER NOT NULL DEFAULT 6,
    stress INTEGER NOT NULL,
    armor INTEGER NOT NULL,
    conditions_json TEXT NOT NULL,
    temporary_armor_json TEXT NOT NULL DEFAULT '[]',
    life_state TEXT NOT NULL DEFAULT 'alive',
    class_state_json TEXT NOT NULL DEFAULT '{}',
    companion_state_json TEXT NOT NULL DEFAULT '{}',
    impenetrable_used_this_short_rest INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (campaign_id, character_id)
);
