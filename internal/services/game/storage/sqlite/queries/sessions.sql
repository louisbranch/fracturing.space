-- name: GetSession :one
SELECT * FROM sessions WHERE campaign_id = ? AND id = ?;

-- name: PutSession :exec
INSERT INTO sessions (
    campaign_id, id, name, status, started_at, updated_at, ended_at
) VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(campaign_id, id) DO UPDATE SET
    name = excluded.name,
    status = excluded.status,
    updated_at = excluded.updated_at,
    ended_at = excluded.ended_at;

-- name: UpdateSessionStatus :exec
UPDATE sessions
SET status = ?, updated_at = ?, ended_at = ?
WHERE campaign_id = ? AND id = ?;

-- name: ListSessionsByCampaign :many
SELECT * FROM sessions
WHERE campaign_id = ?
ORDER BY id;

-- name: ListSessionsByCampaignPaged :many
SELECT * FROM sessions
WHERE campaign_id = ? AND id > ?
ORDER BY id
LIMIT ?;

-- name: ListSessionsByCampaignPagedFirst :many
SELECT * FROM sessions
WHERE campaign_id = ?
ORDER BY id
LIMIT ?;

-- name: GetActiveSession :one
SELECT s.* FROM sessions s
JOIN campaign_active_session cas ON s.campaign_id = cas.campaign_id AND s.id = cas.session_id
WHERE s.campaign_id = ?;

-- name: SetActiveSession :exec
INSERT INTO campaign_active_session (campaign_id, session_id)
VALUES (?, ?)
ON CONFLICT(campaign_id) DO UPDATE SET
    session_id = excluded.session_id;

-- name: ClearActiveSession :exec
DELETE FROM campaign_active_session WHERE campaign_id = ?;

-- name: HasActiveSession :one
SELECT EXISTS(SELECT 1 FROM campaign_active_session WHERE campaign_id = ?) as has_active;

-- Session event queries have been moved to events.sql (unified event table)
