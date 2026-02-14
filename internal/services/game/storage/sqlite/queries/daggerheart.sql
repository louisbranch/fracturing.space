-- Daggerheart-specific queries for extension tables.

-- Character Profile Extensions

-- name: GetDaggerheartCharacterProfile :one
SELECT * FROM daggerheart_character_profiles
WHERE campaign_id = ? AND character_id = ?;

-- name: PutDaggerheartCharacterProfile :exec
INSERT INTO daggerheart_character_profiles (
    campaign_id, character_id, level, hp_max, stress_max, evasion, major_threshold, severe_threshold,
    agility, strength, finesse, instinct, presence, knowledge, proficiency, armor_score, armor_max,
    experiences_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(campaign_id, character_id) DO UPDATE SET
    level = excluded.level,
    hp_max = excluded.hp_max,
    stress_max = excluded.stress_max,
    evasion = excluded.evasion,
    major_threshold = excluded.major_threshold,
    severe_threshold = excluded.severe_threshold,
    agility = excluded.agility,
    strength = excluded.strength,
    finesse = excluded.finesse,
    instinct = excluded.instinct,
    presence = excluded.presence,
    knowledge = excluded.knowledge,
    proficiency = excluded.proficiency,
    armor_score = excluded.armor_score,
    armor_max = excluded.armor_max,
    experiences_json = excluded.experiences_json;

-- name: DeleteDaggerheartCharacterProfile :exec
DELETE FROM daggerheart_character_profiles
WHERE campaign_id = ? AND character_id = ?;

-- Character State Extensions

-- name: GetDaggerheartCharacterState :one
SELECT * FROM daggerheart_character_states
WHERE campaign_id = ? AND character_id = ?;

-- name: PutDaggerheartCharacterState :exec
INSERT INTO daggerheart_character_states (
    campaign_id, character_id, hp, hope, hope_max, stress, armor, conditions_json, life_state
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(campaign_id, character_id) DO UPDATE SET
    hp = excluded.hp,
    hope = excluded.hope,
    hope_max = excluded.hope_max,
    stress = excluded.stress,
    armor = excluded.armor,
    conditions_json = excluded.conditions_json,
    life_state = excluded.life_state;

-- name: UpdateDaggerheartCharacterState :exec
UPDATE daggerheart_character_states
SET hp = ?, hope = ?, hope_max = ?, stress = ?, armor = ?, conditions_json = ?, life_state = ?
WHERE campaign_id = ? AND character_id = ?;

-- name: UpdateDaggerheartCharacterStateHopeStress :exec
UPDATE daggerheart_character_states
SET hope = ?, stress = ?
WHERE campaign_id = ? AND character_id = ?;

-- name: DeleteDaggerheartCharacterState :exec
DELETE FROM daggerheart_character_states
WHERE campaign_id = ? AND character_id = ?;

-- Snapshot Extensions

-- name: GetDaggerheartSnapshot :one
SELECT * FROM daggerheart_snapshots WHERE campaign_id = ?;

-- name: PutDaggerheartSnapshot :exec
INSERT INTO daggerheart_snapshots (campaign_id, gm_fear, consecutive_short_rests)
VALUES (?, ?, ?)
ON CONFLICT(campaign_id) DO UPDATE SET
    gm_fear = excluded.gm_fear,
    consecutive_short_rests = excluded.consecutive_short_rests;

-- name: UpdateDaggerheartGmFear :exec
UPDATE daggerheart_snapshots SET gm_fear = ? WHERE campaign_id = ?;

-- name: DeleteDaggerheartSnapshot :exec
DELETE FROM daggerheart_snapshots WHERE campaign_id = ?;

-- Countdown Extensions

-- name: GetDaggerheartCountdown :one
SELECT * FROM daggerheart_countdowns
WHERE campaign_id = ? AND countdown_id = ?;

-- name: ListDaggerheartCountdowns :many
SELECT * FROM daggerheart_countdowns
WHERE campaign_id = ?;

-- name: PutDaggerheartCountdown :exec
INSERT INTO daggerheart_countdowns (
    campaign_id, countdown_id, name, kind, current, max, direction, looping
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(campaign_id, countdown_id) DO UPDATE SET
    name = excluded.name,
    kind = excluded.kind,
    current = excluded.current,
    max = excluded.max,
    direction = excluded.direction,
    looping = excluded.looping;

-- name: DeleteDaggerheartCountdown :exec
DELETE FROM daggerheart_countdowns
WHERE campaign_id = ? AND countdown_id = ?;

-- Adversary Extensions

-- name: GetDaggerheartAdversary :one
SELECT * FROM daggerheart_adversaries
WHERE campaign_id = ? AND adversary_id = ?;

-- name: ListDaggerheartAdversariesByCampaign :many
SELECT * FROM daggerheart_adversaries
WHERE campaign_id = ?
ORDER BY name ASC, adversary_id ASC;

-- name: ListDaggerheartAdversariesBySession :many
SELECT * FROM daggerheart_adversaries
WHERE campaign_id = ? AND session_id = ?
ORDER BY name ASC, adversary_id ASC;

-- name: PutDaggerheartAdversary :exec
INSERT INTO daggerheart_adversaries (
    campaign_id, adversary_id, name, kind, session_id, notes, hp, hp_max, stress, stress_max,
    evasion, major_threshold, severe_threshold, armor, conditions_json, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(campaign_id, adversary_id) DO UPDATE SET
    name = excluded.name,
    kind = excluded.kind,
    session_id = excluded.session_id,
    notes = excluded.notes,
    hp = excluded.hp,
    hp_max = excluded.hp_max,
    stress = excluded.stress,
    stress_max = excluded.stress_max,
    evasion = excluded.evasion,
    major_threshold = excluded.major_threshold,
    severe_threshold = excluded.severe_threshold,
    armor = excluded.armor,
    conditions_json = excluded.conditions_json,
    created_at = excluded.created_at,
    updated_at = excluded.updated_at;

-- name: DeleteDaggerheartAdversary :exec
DELETE FROM daggerheart_adversaries
WHERE campaign_id = ? AND adversary_id = ?;

-- Content Catalog

-- name: GetDaggerheartClass :one
SELECT * FROM daggerheart_classes WHERE id = ?;

-- name: ListDaggerheartClasses :many
SELECT * FROM daggerheart_classes ORDER BY name ASC, id ASC;

-- name: PutDaggerheartClass :exec
INSERT INTO daggerheart_classes (
    id, name, starting_evasion, starting_hp, starting_items_json, features_json, hope_feature_json, domain_ids_json,
    created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    name = excluded.name,
    starting_evasion = excluded.starting_evasion,
    starting_hp = excluded.starting_hp,
    starting_items_json = excluded.starting_items_json,
    features_json = excluded.features_json,
    hope_feature_json = excluded.hope_feature_json,
    domain_ids_json = excluded.domain_ids_json,
    created_at = excluded.created_at,
    updated_at = excluded.updated_at;

-- name: DeleteDaggerheartClass :exec
DELETE FROM daggerheart_classes WHERE id = ?;

-- name: GetDaggerheartSubclass :one
SELECT * FROM daggerheart_subclasses WHERE id = ?;

-- name: ListDaggerheartSubclasses :many
SELECT * FROM daggerheart_subclasses ORDER BY name ASC, id ASC;

-- name: PutDaggerheartSubclass :exec
INSERT INTO daggerheart_subclasses (
    id, name, spellcast_trait, foundation_features_json, specialization_features_json, mastery_features_json,
    created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    name = excluded.name,
    spellcast_trait = excluded.spellcast_trait,
    foundation_features_json = excluded.foundation_features_json,
    specialization_features_json = excluded.specialization_features_json,
    mastery_features_json = excluded.mastery_features_json,
    created_at = excluded.created_at,
    updated_at = excluded.updated_at;

-- name: DeleteDaggerheartSubclass :exec
DELETE FROM daggerheart_subclasses WHERE id = ?;

-- name: GetDaggerheartHeritage :one
SELECT * FROM daggerheart_heritages WHERE id = ?;

-- name: ListDaggerheartHeritages :many
SELECT * FROM daggerheart_heritages ORDER BY name ASC, id ASC;

-- name: PutDaggerheartHeritage :exec
INSERT INTO daggerheart_heritages (
    id, name, kind, features_json, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    name = excluded.name,
    kind = excluded.kind,
    features_json = excluded.features_json,
    created_at = excluded.created_at,
    updated_at = excluded.updated_at;

-- name: DeleteDaggerheartHeritage :exec
DELETE FROM daggerheart_heritages WHERE id = ?;

-- name: GetDaggerheartExperience :one
SELECT * FROM daggerheart_experiences WHERE id = ?;

-- name: ListDaggerheartExperiences :many
SELECT * FROM daggerheart_experiences ORDER BY name ASC, id ASC;

-- name: PutDaggerheartExperience :exec
INSERT INTO daggerheart_experiences (
    id, name, description, created_at, updated_at
) VALUES (?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    name = excluded.name,
    description = excluded.description,
    created_at = excluded.created_at,
    updated_at = excluded.updated_at;

-- name: DeleteDaggerheartExperience :exec
DELETE FROM daggerheart_experiences WHERE id = ?;

-- name: GetDaggerheartAdversaryEntry :one
SELECT * FROM daggerheart_adversary_entries WHERE id = ?;

-- name: ListDaggerheartAdversaryEntries :many
SELECT * FROM daggerheart_adversary_entries ORDER BY name ASC, id ASC;

-- name: PutDaggerheartAdversaryEntry :exec
INSERT INTO daggerheart_adversary_entries (
    id, name, tier, role, description, motives, difficulty, major_threshold, severe_threshold, hp, stress, armor,
    attack_modifier, standard_attack_json, experiences_json, features_json, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    name = excluded.name,
    tier = excluded.tier,
    role = excluded.role,
    description = excluded.description,
    motives = excluded.motives,
    difficulty = excluded.difficulty,
    major_threshold = excluded.major_threshold,
    severe_threshold = excluded.severe_threshold,
    hp = excluded.hp,
    stress = excluded.stress,
    armor = excluded.armor,
    attack_modifier = excluded.attack_modifier,
    standard_attack_json = excluded.standard_attack_json,
    experiences_json = excluded.experiences_json,
    features_json = excluded.features_json,
    created_at = excluded.created_at,
    updated_at = excluded.updated_at;

-- name: DeleteDaggerheartAdversaryEntry :exec
DELETE FROM daggerheart_adversary_entries WHERE id = ?;

-- name: GetDaggerheartBeastform :one
SELECT * FROM daggerheart_beastforms WHERE id = ?;

-- name: ListDaggerheartBeastforms :many
SELECT * FROM daggerheart_beastforms ORDER BY name ASC, id ASC;

-- name: PutDaggerheartBeastform :exec
INSERT INTO daggerheart_beastforms (
    id, name, tier, examples, trait, trait_bonus, evasion_bonus, attack_json, advantages_json, features_json,
    created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    name = excluded.name,
    tier = excluded.tier,
    examples = excluded.examples,
    trait = excluded.trait,
    trait_bonus = excluded.trait_bonus,
    evasion_bonus = excluded.evasion_bonus,
    attack_json = excluded.attack_json,
    advantages_json = excluded.advantages_json,
    features_json = excluded.features_json,
    created_at = excluded.created_at,
    updated_at = excluded.updated_at;

-- name: DeleteDaggerheartBeastform :exec
DELETE FROM daggerheart_beastforms WHERE id = ?;

-- name: GetDaggerheartCompanionExperience :one
SELECT * FROM daggerheart_companion_experiences WHERE id = ?;

-- name: ListDaggerheartCompanionExperiences :many
SELECT * FROM daggerheart_companion_experiences ORDER BY name ASC, id ASC;

-- name: PutDaggerheartCompanionExperience :exec
INSERT INTO daggerheart_companion_experiences (
    id, name, description, created_at, updated_at
) VALUES (?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    name = excluded.name,
    description = excluded.description,
    created_at = excluded.created_at,
    updated_at = excluded.updated_at;

-- name: DeleteDaggerheartCompanionExperience :exec
DELETE FROM daggerheart_companion_experiences WHERE id = ?;

-- name: GetDaggerheartLootEntry :one
SELECT * FROM daggerheart_loot_entries WHERE id = ?;

-- name: ListDaggerheartLootEntries :many
SELECT * FROM daggerheart_loot_entries ORDER BY roll ASC, id ASC;

-- name: PutDaggerheartLootEntry :exec
INSERT INTO daggerheart_loot_entries (
    id, name, roll, description, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    name = excluded.name,
    roll = excluded.roll,
    description = excluded.description,
    created_at = excluded.created_at,
    updated_at = excluded.updated_at;

-- name: DeleteDaggerheartLootEntry :exec
DELETE FROM daggerheart_loot_entries WHERE id = ?;

-- name: GetDaggerheartDamageType :one
SELECT * FROM daggerheart_damage_types WHERE id = ?;

-- name: ListDaggerheartDamageTypes :many
SELECT * FROM daggerheart_damage_types ORDER BY name ASC, id ASC;

-- name: PutDaggerheartDamageType :exec
INSERT INTO daggerheart_damage_types (
    id, name, description, created_at, updated_at
) VALUES (?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    name = excluded.name,
    description = excluded.description,
    created_at = excluded.created_at,
    updated_at = excluded.updated_at;

-- name: DeleteDaggerheartDamageType :exec
DELETE FROM daggerheart_damage_types WHERE id = ?;

-- name: GetDaggerheartDomain :one
SELECT * FROM daggerheart_domains WHERE id = ?;

-- name: ListDaggerheartDomains :many
SELECT * FROM daggerheart_domains ORDER BY name ASC, id ASC;

-- name: PutDaggerheartDomain :exec
INSERT INTO daggerheart_domains (
    id, name, description, created_at, updated_at
) VALUES (?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    name = excluded.name,
    description = excluded.description,
    created_at = excluded.created_at,
    updated_at = excluded.updated_at;

-- name: DeleteDaggerheartDomain :exec
DELETE FROM daggerheart_domains WHERE id = ?;

-- name: GetDaggerheartDomainCard :one
SELECT * FROM daggerheart_domain_cards WHERE id = ?;

-- name: ListDaggerheartDomainCards :many
SELECT * FROM daggerheart_domain_cards ORDER BY name ASC, id ASC;

-- name: ListDaggerheartDomainCardsByDomain :many
SELECT * FROM daggerheart_domain_cards WHERE domain_id = ? ORDER BY level ASC, name ASC, id ASC;

-- name: PutDaggerheartDomainCard :exec
INSERT INTO daggerheart_domain_cards (
    id, name, domain_id, level, type, recall_cost, usage_limit, feature_text, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    name = excluded.name,
    domain_id = excluded.domain_id,
    level = excluded.level,
    type = excluded.type,
    recall_cost = excluded.recall_cost,
    usage_limit = excluded.usage_limit,
    feature_text = excluded.feature_text,
    created_at = excluded.created_at,
    updated_at = excluded.updated_at;

-- name: DeleteDaggerheartDomainCard :exec
DELETE FROM daggerheart_domain_cards WHERE id = ?;

-- name: GetDaggerheartWeapon :one
SELECT * FROM daggerheart_weapons WHERE id = ?;

-- name: ListDaggerheartWeapons :many
SELECT * FROM daggerheart_weapons ORDER BY name ASC, id ASC;

-- name: PutDaggerheartWeapon :exec
INSERT INTO daggerheart_weapons (
    id, name, category, tier, trait, range, damage_dice_json, damage_type, burden, feature, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    name = excluded.name,
    category = excluded.category,
    tier = excluded.tier,
    trait = excluded.trait,
    range = excluded.range,
    damage_dice_json = excluded.damage_dice_json,
    damage_type = excluded.damage_type,
    burden = excluded.burden,
    feature = excluded.feature,
    created_at = excluded.created_at,
    updated_at = excluded.updated_at;

-- name: DeleteDaggerheartWeapon :exec
DELETE FROM daggerheart_weapons WHERE id = ?;

-- name: GetDaggerheartArmor :one
SELECT * FROM daggerheart_armor WHERE id = ?;

-- name: ListDaggerheartArmor :many
SELECT * FROM daggerheart_armor ORDER BY name ASC, id ASC;

-- name: PutDaggerheartArmor :exec
INSERT INTO daggerheart_armor (
    id, name, tier, base_major_threshold, base_severe_threshold, armor_score, feature, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    name = excluded.name,
    tier = excluded.tier,
    base_major_threshold = excluded.base_major_threshold,
    base_severe_threshold = excluded.base_severe_threshold,
    armor_score = excluded.armor_score,
    feature = excluded.feature,
    created_at = excluded.created_at,
    updated_at = excluded.updated_at;

-- name: DeleteDaggerheartArmor :exec
DELETE FROM daggerheart_armor WHERE id = ?;

-- name: GetDaggerheartItem :one
SELECT * FROM daggerheart_items WHERE id = ?;

-- name: ListDaggerheartItems :many
SELECT * FROM daggerheart_items ORDER BY name ASC, id ASC;

-- name: PutDaggerheartItem :exec
INSERT INTO daggerheart_items (
    id, name, rarity, kind, stack_max, description, effect_text, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    name = excluded.name,
    rarity = excluded.rarity,
    kind = excluded.kind,
    stack_max = excluded.stack_max,
    description = excluded.description,
    effect_text = excluded.effect_text,
    created_at = excluded.created_at,
    updated_at = excluded.updated_at;

-- name: DeleteDaggerheartItem :exec
DELETE FROM daggerheart_items WHERE id = ?;

-- name: GetDaggerheartEnvironment :one
SELECT * FROM daggerheart_environments WHERE id = ?;

-- name: ListDaggerheartEnvironments :many
SELECT * FROM daggerheart_environments ORDER BY name ASC, id ASC;

-- name: PutDaggerheartEnvironment :exec
INSERT INTO daggerheart_environments (
    id, name, tier, type, difficulty, impulses_json, potential_adversary_ids_json, features_json, prompts_json,
    created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    name = excluded.name,
    tier = excluded.tier,
    type = excluded.type,
    difficulty = excluded.difficulty,
    impulses_json = excluded.impulses_json,
    potential_adversary_ids_json = excluded.potential_adversary_ids_json,
    features_json = excluded.features_json,
    prompts_json = excluded.prompts_json,
    created_at = excluded.created_at,
    updated_at = excluded.updated_at;

-- name: DeleteDaggerheartEnvironment :exec
DELETE FROM daggerheart_environments WHERE id = ?;

-- Content Strings

-- name: PutDaggerheartContentString :exec
INSERT INTO daggerheart_content_strings (
    content_id, content_type, field, locale, text, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(content_id, field, locale) DO UPDATE SET
    content_type = excluded.content_type,
    text = excluded.text,
    created_at = excluded.created_at,
    updated_at = excluded.updated_at;

-- Joined queries for convenience

-- name: GetDaggerheartCharacterSheet :one
SELECT
    c.campaign_id,
    c.id as character_id,
    c.name,
    c.kind,
    c.notes,
    c.created_at,
    c.updated_at,
    dcp.level,
    dcp.hp_max,
    dcp.stress_max,
    dcp.evasion,
    dcp.major_threshold,
    dcp.severe_threshold,
    dcp.agility,
    dcp.strength,
    dcp.finesse,
    dcp.instinct,
    dcp.presence,
    dcp.knowledge,
    dcp.proficiency,
    dcp.armor_score,
    dcp.armor_max,
    dcp.experiences_json,
    dcs.hp,
    dcs.hope,
    dcs.hope_max,
    dcs.stress,
    dcs.armor,
    dcs.conditions_json,
    dcs.life_state
FROM characters c
LEFT JOIN daggerheart_character_profiles dcp ON c.campaign_id = dcp.campaign_id AND c.id = dcp.character_id
LEFT JOIN daggerheart_character_states dcs ON c.campaign_id = dcs.campaign_id AND c.id = dcs.character_id
WHERE c.campaign_id = ? AND c.id = ?;

-- name: ListDaggerheartCharacterStates :many
SELECT
    campaign_id,
    character_id,
    hp,
    hope,
    hope_max,
    stress,
    armor,
    conditions_json,
    life_state
FROM daggerheart_character_states
WHERE campaign_id = ?;
