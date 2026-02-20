-- +migrate Up

DROP TABLE IF EXISTS daggerheart_damage_types;
DROP TABLE IF EXISTS daggerheart_loot_entries;
DROP TABLE IF EXISTS daggerheart_companion_experiences;
DROP TABLE IF EXISTS daggerheart_beastforms;
DROP TABLE IF EXISTS daggerheart_adversary_entries;
DROP TABLE IF EXISTS daggerheart_experiences;
DROP TABLE IF EXISTS daggerheart_content_strings;
DROP TABLE IF EXISTS daggerheart_environments;
DROP TABLE IF EXISTS daggerheart_items;
DROP TABLE IF EXISTS daggerheart_armor;
DROP TABLE IF EXISTS daggerheart_weapons;
DROP TABLE IF EXISTS daggerheart_domain_cards;
DROP TABLE IF EXISTS daggerheart_domains;
DROP TABLE IF EXISTS daggerheart_heritages;
DROP TABLE IF EXISTS daggerheart_subclasses;
DROP TABLE IF EXISTS daggerheart_classes;

CREATE TABLE daggerheart_classes (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    starting_evasion INTEGER NOT NULL DEFAULT 0,
    starting_hp INTEGER NOT NULL DEFAULT 0,
    starting_items_json TEXT NOT NULL DEFAULT '[]',
    features_json TEXT NOT NULL DEFAULT '[]',
    hope_feature_json TEXT NOT NULL DEFAULT '{}',
    domain_ids_json TEXT NOT NULL DEFAULT '[]',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE daggerheart_subclasses (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    spellcast_trait TEXT NOT NULL DEFAULT '',
    foundation_features_json TEXT NOT NULL DEFAULT '[]',
    specialization_features_json TEXT NOT NULL DEFAULT '[]',
    mastery_features_json TEXT NOT NULL DEFAULT '[]',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE daggerheart_heritages (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT '',
    features_json TEXT NOT NULL DEFAULT '[]',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE daggerheart_domains (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE daggerheart_domain_cards (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    domain_id TEXT NOT NULL,
    level INTEGER NOT NULL DEFAULT 0,
    type TEXT NOT NULL DEFAULT '',
    recall_cost INTEGER NOT NULL DEFAULT 0,
    usage_limit TEXT NOT NULL DEFAULT '',
    feature_text TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (domain_id) REFERENCES daggerheart_domains(id) ON DELETE CASCADE
);

CREATE TABLE daggerheart_weapons (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    category TEXT NOT NULL DEFAULT '',
    tier INTEGER NOT NULL DEFAULT 0,
    trait TEXT NOT NULL DEFAULT '',
    range TEXT NOT NULL DEFAULT '',
    damage_dice_json TEXT NOT NULL DEFAULT '[]',
    damage_type TEXT NOT NULL DEFAULT '',
    burden INTEGER NOT NULL DEFAULT 0,
    feature TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE daggerheart_armor (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    tier INTEGER NOT NULL DEFAULT 0,
    base_major_threshold INTEGER NOT NULL DEFAULT 0,
    base_severe_threshold INTEGER NOT NULL DEFAULT 0,
    armor_score INTEGER NOT NULL DEFAULT 0,
    feature TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE daggerheart_items (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    rarity TEXT NOT NULL DEFAULT '',
    kind TEXT NOT NULL DEFAULT '',
    stack_max INTEGER NOT NULL DEFAULT 0,
    description TEXT NOT NULL DEFAULT '',
    effect_text TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE daggerheart_environments (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    tier INTEGER NOT NULL DEFAULT 0,
    type TEXT NOT NULL DEFAULT '',
    difficulty INTEGER NOT NULL DEFAULT 0,
    impulses_json TEXT NOT NULL DEFAULT '[]',
    potential_adversary_ids_json TEXT NOT NULL DEFAULT '[]',
    features_json TEXT NOT NULL DEFAULT '[]',
    prompts_json TEXT NOT NULL DEFAULT '[]',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE daggerheart_content_strings (
    content_id TEXT NOT NULL,
    content_type TEXT NOT NULL,
    field TEXT NOT NULL,
    locale TEXT NOT NULL,
    text TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (content_id, field, locale)
);

CREATE TABLE daggerheart_experiences (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE daggerheart_companion_experiences (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE daggerheart_adversary_entries (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    tier INTEGER NOT NULL DEFAULT 0,
    role TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    motives TEXT NOT NULL DEFAULT '',
    difficulty INTEGER NOT NULL DEFAULT 0,
    major_threshold INTEGER NOT NULL DEFAULT 0,
    severe_threshold INTEGER NOT NULL DEFAULT 0,
    hp INTEGER NOT NULL DEFAULT 0,
    stress INTEGER NOT NULL DEFAULT 0,
    armor INTEGER NOT NULL DEFAULT 0,
    attack_modifier INTEGER NOT NULL DEFAULT 0,
    standard_attack_json TEXT NOT NULL DEFAULT '{}',
    experiences_json TEXT NOT NULL DEFAULT '[]',
    features_json TEXT NOT NULL DEFAULT '[]',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE daggerheart_beastforms (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    tier INTEGER NOT NULL DEFAULT 0,
    examples TEXT NOT NULL DEFAULT '',
    trait TEXT NOT NULL DEFAULT '',
    trait_bonus INTEGER NOT NULL DEFAULT 0,
    evasion_bonus INTEGER NOT NULL DEFAULT 0,
    attack_json TEXT NOT NULL DEFAULT '{}',
    advantages_json TEXT NOT NULL DEFAULT '[]',
    features_json TEXT NOT NULL DEFAULT '[]',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE daggerheart_loot_entries (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    roll INTEGER NOT NULL DEFAULT 0,
    description TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE daggerheart_damage_types (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- +migrate Down
DROP TABLE IF EXISTS daggerheart_damage_types;
DROP TABLE IF EXISTS daggerheart_loot_entries;
DROP TABLE IF EXISTS daggerheart_companion_experiences;
DROP TABLE IF EXISTS daggerheart_beastforms;
DROP TABLE IF EXISTS daggerheart_adversary_entries;
DROP TABLE IF EXISTS daggerheart_experiences;
DROP TABLE IF EXISTS daggerheart_content_strings;
DROP TABLE IF EXISTS daggerheart_environments;
DROP TABLE IF EXISTS daggerheart_items;
DROP TABLE IF EXISTS daggerheart_armor;
DROP TABLE IF EXISTS daggerheart_weapons;
DROP TABLE IF EXISTS daggerheart_domain_cards;
DROP TABLE IF EXISTS daggerheart_domains;
DROP TABLE IF EXISTS daggerheart_heritages;
DROP TABLE IF EXISTS daggerheart_subclasses;
DROP TABLE IF EXISTS daggerheart_classes;
