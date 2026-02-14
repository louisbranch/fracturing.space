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
    evasion, major_threshold, severe_threshold, armor, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
    created_at = excluded.created_at,
    updated_at = excluded.updated_at;

-- name: DeleteDaggerheartAdversary :exec
DELETE FROM daggerheart_adversaries
WHERE campaign_id = ? AND adversary_id = ?;

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
