-- name: GetCharacter :one
SELECT * FROM characters WHERE campaign_id = ? AND id = ?;

-- name: PutCharacter :exec
INSERT INTO characters (
    campaign_id, id, name, kind, notes, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(campaign_id, id) DO UPDATE SET
    name = excluded.name,
    kind = excluded.kind,
    notes = excluded.notes,
    updated_at = excluded.updated_at;

-- name: DeleteCharacter :exec
DELETE FROM characters
WHERE campaign_id = ? AND id = ?;

-- name: ListCharactersByCampaign :many
SELECT * FROM characters
WHERE campaign_id = ?
ORDER BY id;

-- name: ListCharactersByCampaignPaged :many
SELECT * FROM characters
WHERE campaign_id = ? AND id > ?
ORDER BY id
LIMIT ?;

-- name: ListCharactersByCampaignPagedFirst :many
SELECT * FROM characters
WHERE campaign_id = ?
ORDER BY id
LIMIT ?;

-- name: GetControlDefault :one
SELECT * FROM control_defaults WHERE campaign_id = ? AND character_id = ?;

-- name: PutControlDefault :exec
INSERT INTO control_defaults (
    campaign_id, character_id, is_gm, participant_id
) VALUES (?, ?, ?, ?)
ON CONFLICT(campaign_id, character_id) DO UPDATE SET
    is_gm = excluded.is_gm,
    participant_id = excluded.participant_id;
