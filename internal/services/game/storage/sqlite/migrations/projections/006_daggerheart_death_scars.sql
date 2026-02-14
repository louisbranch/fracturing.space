-- +migrate Up

DROP TABLE IF EXISTS daggerheart_character_states;
DROP TABLE IF EXISTS daggerheart_character_profiles;

CREATE TABLE daggerheart_character_profiles (
    campaign_id TEXT NOT NULL,
    character_id TEXT NOT NULL,
    level INTEGER NOT NULL DEFAULT 1,
    hp_max INTEGER NOT NULL DEFAULT 6,
    stress_max INTEGER NOT NULL DEFAULT 6,
    evasion INTEGER NOT NULL DEFAULT 10,
    major_threshold INTEGER NOT NULL DEFAULT 8,
    severe_threshold INTEGER NOT NULL DEFAULT 12,
    agility INTEGER NOT NULL DEFAULT 0,
    strength INTEGER NOT NULL DEFAULT 0,
    finesse INTEGER NOT NULL DEFAULT 0,
    instinct INTEGER NOT NULL DEFAULT 0,
    presence INTEGER NOT NULL DEFAULT 0,
    knowledge INTEGER NOT NULL DEFAULT 0,
    proficiency INTEGER NOT NULL DEFAULT 0,
    armor_score INTEGER NOT NULL DEFAULT 0,
    armor_max INTEGER NOT NULL DEFAULT 0,
    experiences_json TEXT NOT NULL DEFAULT '[]',
    PRIMARY KEY (campaign_id, character_id),
    FOREIGN KEY (campaign_id, character_id)
        REFERENCES characters(campaign_id, id) ON DELETE CASCADE
);

CREATE TABLE daggerheart_character_states (
    campaign_id TEXT NOT NULL,
    character_id TEXT NOT NULL,
    hp INTEGER NOT NULL DEFAULT 6,
    hope INTEGER NOT NULL DEFAULT 2,
    hope_max INTEGER NOT NULL DEFAULT 6,
    stress INTEGER NOT NULL DEFAULT 0,
    armor INTEGER NOT NULL DEFAULT 0,
    conditions_json TEXT NOT NULL DEFAULT '[]',
    life_state TEXT NOT NULL DEFAULT 'alive',
    PRIMARY KEY (campaign_id, character_id),
    FOREIGN KEY (campaign_id, character_id)
        REFERENCES characters(campaign_id, id) ON DELETE CASCADE
);

-- +migrate Down
