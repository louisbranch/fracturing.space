-- name: GetCampaign :one
SELECT
    c.id, c.name, c.game_system, c.status, c.gm_mode,
    (SELECT COUNT(*) FROM participants p WHERE p.campaign_id = c.id) AS participant_count,
    (SELECT COUNT(*) FROM characters ch WHERE ch.campaign_id = c.id) AS character_count,
    c.theme_prompt, c.parent_campaign_id, c.fork_event_seq, c.origin_campaign_id,
    c.created_at, c.updated_at, c.completed_at, c.archived_at
FROM campaigns c WHERE c.id = ?;

-- name: PutCampaign :exec
INSERT INTO campaigns (
    id, name, game_system, status, gm_mode,
    participant_count, character_count, theme_prompt,
    created_at, updated_at, completed_at, archived_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    name = excluded.name,
    game_system = excluded.game_system,
    status = excluded.status,
    gm_mode = excluded.gm_mode,
    participant_count = excluded.participant_count,
    character_count = excluded.character_count,
    theme_prompt = excluded.theme_prompt,
    updated_at = excluded.updated_at,
    completed_at = excluded.completed_at,
    archived_at = excluded.archived_at;

-- name: ListCampaigns :many
SELECT
    c.id, c.name, c.game_system, c.status, c.gm_mode,
    (SELECT COUNT(*) FROM participants p WHERE p.campaign_id = c.id) AS participant_count,
    (SELECT COUNT(*) FROM characters ch WHERE ch.campaign_id = c.id) AS character_count,
    c.theme_prompt, c.parent_campaign_id, c.fork_event_seq, c.origin_campaign_id,
    c.created_at, c.updated_at, c.completed_at, c.archived_at
FROM campaigns c
WHERE c.id > ?
ORDER BY c.id
LIMIT ?;

-- name: ListAllCampaigns :many
SELECT
    c.id, c.name, c.game_system, c.status, c.gm_mode,
    (SELECT COUNT(*) FROM participants p WHERE p.campaign_id = c.id) AS participant_count,
    (SELECT COUNT(*) FROM characters ch WHERE ch.campaign_id = c.id) AS character_count,
    c.theme_prompt, c.parent_campaign_id, c.fork_event_seq, c.origin_campaign_id,
    c.created_at, c.updated_at, c.completed_at, c.archived_at
FROM campaigns c
ORDER BY c.id
LIMIT ?;
