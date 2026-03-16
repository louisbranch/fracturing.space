-- +migrate Up

ALTER TABLE daggerheart_character_states
    ADD COLUMN class_state_json TEXT NOT NULL DEFAULT '{}';

-- +migrate Down

DROP TABLE IF EXISTS daggerheart_character_states;

CREATE TABLE IF NOT EXISTS daggerheart_character_states (
    campaign_id TEXT NOT NULL,
    character_id TEXT NOT NULL,
    hp INTEGER NOT NULL DEFAULT 6,
    hope INTEGER NOT NULL DEFAULT 2,
    hope_max INTEGER NOT NULL DEFAULT 6,
    stress INTEGER NOT NULL DEFAULT 0,
    armor INTEGER NOT NULL DEFAULT 0,
    conditions_json TEXT NOT NULL DEFAULT '[]',
    temporary_armor_json TEXT NOT NULL DEFAULT '[]',
    life_state TEXT NOT NULL DEFAULT 'alive',
    impenetrable_used_this_short_rest INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (campaign_id, character_id),
    FOREIGN KEY (campaign_id, character_id)
        REFERENCES characters(campaign_id, id) ON DELETE CASCADE
);
