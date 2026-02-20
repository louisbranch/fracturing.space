-- name: GetParticipant :one
SELECT campaign_id, id, user_id, display_name, role, controller, campaign_access, created_at, updated_at FROM participants WHERE campaign_id = ? AND id = ?;

-- name: PutParticipant :exec
INSERT INTO participants (
	campaign_id, id, user_id, display_name, role, controller, campaign_access, created_at, updated_at
 ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(campaign_id, id) DO UPDATE SET
	user_id = excluded.user_id,
	display_name = excluded.display_name,
	role = excluded.role,
	controller = excluded.controller,
	campaign_access = excluded.campaign_access,
	updated_at = excluded.updated_at;

-- name: DeleteParticipant :exec
DELETE FROM participants
WHERE campaign_id = ? AND id = ?;

-- name: ListParticipantsByCampaign :many
SELECT campaign_id, id, user_id, display_name, role, controller, campaign_access, created_at, updated_at FROM participants
WHERE campaign_id = ?
ORDER BY id;

-- name: ListParticipantsByCampaignPaged :many
SELECT campaign_id, id, user_id, display_name, role, controller, campaign_access, created_at, updated_at FROM participants
WHERE campaign_id = ? AND id > ?
ORDER BY id
LIMIT ?;

-- name: ListParticipantsByCampaignPagedFirst :many
SELECT campaign_id, id, user_id, display_name, role, controller, campaign_access, created_at, updated_at FROM participants
WHERE campaign_id = ?
ORDER BY id
LIMIT ?;

-- name: ListCampaignIDsByUser :many
SELECT DISTINCT campaign_id FROM participants WHERE user_id = ? ORDER BY campaign_id;

-- name: ListCampaignIDsByParticipant :many
SELECT DISTINCT campaign_id FROM participants WHERE id = ? ORDER BY campaign_id;
