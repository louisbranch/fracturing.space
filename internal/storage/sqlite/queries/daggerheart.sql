-- Daggerheart-specific queries for extension tables.

-- Character Profile Extensions

-- name: GetDaggerheartCharacterProfile :one
SELECT * FROM daggerheart_character_profiles
WHERE campaign_id = ? AND character_id = ?;

-- name: PutDaggerheartCharacterProfile :exec
INSERT INTO daggerheart_character_profiles (
    campaign_id, character_id, hp_max, stress_max, evasion, major_threshold, severe_threshold,
    agility, strength, finesse, instinct, presence, knowledge
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(campaign_id, character_id) DO UPDATE SET
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
    knowledge = excluded.knowledge;

-- name: DeleteDaggerheartCharacterProfile :exec
DELETE FROM daggerheart_character_profiles
WHERE campaign_id = ? AND character_id = ?;

-- Character State Extensions

-- name: GetDaggerheartCharacterState :one
SELECT * FROM daggerheart_character_states
WHERE campaign_id = ? AND character_id = ?;

-- name: PutDaggerheartCharacterState :exec
INSERT INTO daggerheart_character_states (
    campaign_id, character_id, hp, hope, stress
) VALUES (?, ?, ?, ?, ?)
ON CONFLICT(campaign_id, character_id) DO UPDATE SET
    hp = excluded.hp,
    hope = excluded.hope,
    stress = excluded.stress;

-- name: UpdateDaggerheartCharacterState :exec
UPDATE daggerheart_character_states
SET hp = ?, hope = ?, stress = ?
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
INSERT INTO daggerheart_snapshots (campaign_id, gm_fear)
VALUES (?, ?)
ON CONFLICT(campaign_id) DO UPDATE SET
    gm_fear = excluded.gm_fear;

-- name: UpdateDaggerheartGmFear :exec
UPDATE daggerheart_snapshots SET gm_fear = ? WHERE campaign_id = ?;

-- name: DeleteDaggerheartSnapshot :exec
DELETE FROM daggerheart_snapshots WHERE campaign_id = ?;

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
    dcs.hp,
    dcs.hope,
    dcs.stress
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
    stress
FROM daggerheart_character_states
WHERE campaign_id = ?;
