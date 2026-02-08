-- Unified Events Table Queries

-- name: AppendEvent :exec
INSERT INTO events (
    campaign_id, seq, event_hash, timestamp, event_type,
    session_id, request_id, invocation_id,
    actor_type, actor_id, entity_type, entity_id, payload_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetEventByHash :one
SELECT * FROM events WHERE event_hash = ?;

-- name: GetEventBySeq :one
SELECT * FROM events WHERE campaign_id = ? AND seq = ?;

-- name: ListEvents :many
SELECT * FROM events
WHERE campaign_id = ? AND seq > ?
ORDER BY seq
LIMIT ?;

-- name: ListEventsBySession :many
SELECT * FROM events
WHERE campaign_id = ? AND session_id = ? AND seq > ?
ORDER BY seq
LIMIT ?;

-- name: GetEventSeq :one
SELECT next_seq FROM event_seq WHERE campaign_id = ?;

-- name: IncrementEventSeq :exec
INSERT INTO event_seq (campaign_id, next_seq)
VALUES (?, 2)
ON CONFLICT(campaign_id) DO UPDATE SET
    next_seq = event_seq.next_seq + 1;

-- name: InitEventSeq :exec
INSERT OR IGNORE INTO event_seq (campaign_id, next_seq) VALUES (?, 1);

-- name: GetLatestEventSeq :one
SELECT CAST(COALESCE(MAX(seq), 0) AS INTEGER) as latest_seq FROM events WHERE campaign_id = ?;

-- Outcome Applied Tracking

-- name: CheckOutcomeApplied :one
SELECT EXISTS(SELECT 1 FROM outcome_applied WHERE campaign_id = ? AND session_id = ? AND request_id = ?) as applied;

-- name: MarkOutcomeApplied :exec
INSERT INTO outcome_applied (campaign_id, session_id, request_id) VALUES (?, ?, ?);

-- Snapshot Queries (unchanged from campaign_events.sql)

-- name: PutSnapshot :exec
INSERT INTO snapshots (
    campaign_id, session_id, event_seq, character_states_json, gm_state_json, system_state_json, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(campaign_id, session_id) DO UPDATE SET
    event_seq = excluded.event_seq,
    character_states_json = excluded.character_states_json,
    gm_state_json = excluded.gm_state_json,
    system_state_json = excluded.system_state_json,
    created_at = excluded.created_at;

-- name: GetSnapshot :one
SELECT * FROM snapshots
WHERE campaign_id = ? AND session_id = ?;

-- name: GetLatestSnapshot :one
SELECT * FROM snapshots
WHERE campaign_id = ?
ORDER BY event_seq DESC
LIMIT 1;

-- name: ListSnapshots :many
SELECT * FROM snapshots
WHERE campaign_id = ?
ORDER BY event_seq DESC
LIMIT ?;

-- Fork Metadata Queries

-- name: GetCampaignForkMetadata :one
SELECT parent_campaign_id, fork_event_seq, origin_campaign_id
FROM campaigns
WHERE id = ?;

-- name: SetCampaignForkMetadata :exec
UPDATE campaigns
SET parent_campaign_id = ?, fork_event_seq = ?, origin_campaign_id = ?
WHERE id = ?;
