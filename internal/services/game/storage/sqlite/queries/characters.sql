-- name: GetCharacter :one
SELECT * FROM characters WHERE campaign_id = ? AND id = ?;

-- name: PutCharacter :exec
INSERT INTO characters (
    campaign_id, id, controller_participant_id, name, kind, notes, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(campaign_id, id) DO UPDATE SET
    controller_participant_id = excluded.controller_participant_id,
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
