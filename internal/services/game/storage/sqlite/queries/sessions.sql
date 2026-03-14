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

-- name: PutSessionGate :exec
INSERT INTO session_gates (
    campaign_id, session_id, gate_id, gate_type, status, reason,
    created_at, created_by_actor_type, created_by_actor_id,
    resolved_at, resolved_by_actor_type, resolved_by_actor_id,
    metadata_json, progress_json, resolution_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(campaign_id, session_id, gate_id) DO UPDATE SET
    gate_type = excluded.gate_type,
    status = excluded.status,
    reason = excluded.reason,
    created_at = excluded.created_at,
    created_by_actor_type = excluded.created_by_actor_type,
    created_by_actor_id = excluded.created_by_actor_id,
    resolved_at = excluded.resolved_at,
    resolved_by_actor_type = excluded.resolved_by_actor_type,
    resolved_by_actor_id = excluded.resolved_by_actor_id,
    metadata_json = excluded.metadata_json,
    progress_json = excluded.progress_json,
    resolution_json = excluded.resolution_json;

-- name: GetSessionGate :one
SELECT * FROM session_gates
WHERE campaign_id = ? AND session_id = ? AND gate_id = ?;

-- name: GetOpenSessionGate :one
SELECT * FROM session_gates
WHERE campaign_id = ? AND session_id = ? AND status = 'open'
ORDER BY created_at DESC
LIMIT 1;

-- name: PutSessionSpotlight :exec
INSERT INTO session_spotlight (
    campaign_id, session_id, spotlight_type, character_id,
    updated_at, updated_by_actor_type, updated_by_actor_id
) VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(campaign_id, session_id) DO UPDATE SET
    spotlight_type = excluded.spotlight_type,
    character_id = excluded.character_id,
    updated_at = excluded.updated_at,
    updated_by_actor_type = excluded.updated_by_actor_type,
    updated_by_actor_id = excluded.updated_by_actor_id;

-- name: GetSessionSpotlight :one
SELECT * FROM session_spotlight
WHERE campaign_id = ? AND session_id = ?;

-- name: ClearSessionSpotlight :exec
DELETE FROM session_spotlight WHERE campaign_id = ? AND session_id = ?;

-- Session event queries have been moved to events.sql (unified event table)
