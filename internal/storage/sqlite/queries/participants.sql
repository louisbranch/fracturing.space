-- name: GetParticipant :one
SELECT * FROM participants WHERE campaign_id = ? AND id = ?;

-- name: PutParticipant :exec
INSERT INTO participants (
    campaign_id, id, display_name, role, controller, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(campaign_id, id) DO UPDATE SET
    display_name = excluded.display_name,
    role = excluded.role,
    controller = excluded.controller,
    updated_at = excluded.updated_at;

-- name: DeleteParticipant :exec
DELETE FROM participants
WHERE campaign_id = ? AND id = ?;

-- name: ListParticipantsByCampaign :many
SELECT * FROM participants
WHERE campaign_id = ?
ORDER BY id;

-- name: ListParticipantsByCampaignPaged :many
SELECT * FROM participants
WHERE campaign_id = ? AND id > ?
ORDER BY id
LIMIT ?;

-- name: ListParticipantsByCampaignPagedFirst :many
SELECT * FROM participants
WHERE campaign_id = ?
ORDER BY id
LIMIT ?;
