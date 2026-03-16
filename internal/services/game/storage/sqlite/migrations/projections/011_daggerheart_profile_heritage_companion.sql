-- +migrate Up

DROP TABLE IF EXISTS daggerheart_character_profiles;

CREATE TABLE IF NOT EXISTS daggerheart_character_profiles (
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
    class_id TEXT NOT NULL DEFAULT '',
    subclass_id TEXT NOT NULL DEFAULT '',
    subclass_tracks_json TEXT NOT NULL DEFAULT '[]',
    subclass_creation_requirements_json TEXT NOT NULL DEFAULT '[]',
    heritage_json TEXT NOT NULL DEFAULT '{}',
    companion_sheet_json TEXT NOT NULL DEFAULT '',
    equipped_armor_id TEXT NOT NULL DEFAULT '',
    spellcast_roll_bonus INTEGER NOT NULL DEFAULT 0,
    traits_assigned INTEGER NOT NULL DEFAULT 0,
    details_recorded INTEGER NOT NULL DEFAULT 0,
    starting_weapon_ids_json TEXT NOT NULL DEFAULT '[]',
    starting_armor_id TEXT NOT NULL DEFAULT '',
    starting_potion_item_id TEXT NOT NULL DEFAULT '',
    background TEXT NOT NULL DEFAULT '',
    domain_card_ids_json TEXT NOT NULL DEFAULT '[]',
    connections TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (campaign_id, character_id),
    FOREIGN KEY (campaign_id, character_id)
        REFERENCES characters(campaign_id, id) ON DELETE CASCADE
);

-- +migrate Down

DROP TABLE IF EXISTS daggerheart_character_profiles;

CREATE TABLE IF NOT EXISTS daggerheart_character_profiles (
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
    class_id TEXT NOT NULL DEFAULT '',
    subclass_id TEXT NOT NULL DEFAULT '',
    subclass_tracks_json TEXT NOT NULL DEFAULT '[]',
    subclass_creation_requirements_json TEXT NOT NULL DEFAULT '[]',
    heritage_json TEXT NOT NULL DEFAULT '{}',
    companion_sheet_json TEXT NOT NULL DEFAULT '',
    equipped_armor_id TEXT NOT NULL DEFAULT '',
    spellcast_roll_bonus INTEGER NOT NULL DEFAULT 0,
    traits_assigned INTEGER NOT NULL DEFAULT 0,
    details_recorded INTEGER NOT NULL DEFAULT 0,
    starting_weapon_ids_json TEXT NOT NULL DEFAULT '[]',
    starting_armor_id TEXT NOT NULL DEFAULT '',
    starting_potion_item_id TEXT NOT NULL DEFAULT '',
    background TEXT NOT NULL DEFAULT '',
    domain_card_ids_json TEXT NOT NULL DEFAULT '[]',
    connections TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (campaign_id, character_id),
    FOREIGN KEY (campaign_id, character_id)
        REFERENCES characters(campaign_id, id) ON DELETE CASCADE
);
