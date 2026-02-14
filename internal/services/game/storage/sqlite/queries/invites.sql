-- name: GetInvite :one
SELECT * FROM invites WHERE id = ?;

-- name: PutInvite :exec
INSERT INTO invites (
    id, campaign_id, participant_id, recipient_user_id, status, created_by_participant_id, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    status = excluded.status,
    updated_at = excluded.updated_at;

-- name: UpdateInviteStatus :exec
UPDATE invites
SET status = ?, updated_at = ?
WHERE id = ?;

-- name: ListInvitesByCampaignPaged :many
SELECT * FROM invites
WHERE campaign_id = ?
  AND id > ?
  AND (? = '' OR recipient_user_id = ?)
  AND (? = '' OR status = ?)
ORDER BY id
LIMIT ?;

-- name: ListInvitesByCampaignPagedFirst :many
SELECT * FROM invites
WHERE campaign_id = ?
  AND (? = '' OR recipient_user_id = ?)
  AND (? = '' OR status = ?)
ORDER BY id
LIMIT ?;

-- name: ListPendingInvitesByCampaignPaged :many
SELECT * FROM invites
WHERE campaign_id = ? AND status = ? AND id > ?
ORDER BY id
LIMIT ?;

-- name: ListPendingInvitesByCampaignPagedFirst :many
SELECT * FROM invites
WHERE campaign_id = ? AND status = ?
ORDER BY id
LIMIT ?;

-- name: ListPendingInvitesByRecipientPaged :many
SELECT * FROM invites
WHERE recipient_user_id = ? AND status = ? AND id > ?
ORDER BY id
LIMIT ?;

-- name: ListPendingInvitesByRecipientPagedFirst :many
SELECT * FROM invites
WHERE recipient_user_id = ? AND status = ?
ORDER BY id
LIMIT ?;
