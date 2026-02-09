-- name: GetGameStatistics :one
SELECT
    (SELECT COUNT(*) FROM campaigns WHERE (?1 IS NULL OR created_at >= ?1)) AS campaign_count,
    (SELECT COUNT(*) FROM sessions WHERE (?1 IS NULL OR started_at >= ?1)) AS session_count,
    (SELECT COUNT(*) FROM characters WHERE (?1 IS NULL OR created_at >= ?1)) AS character_count,
    (SELECT COUNT(*) FROM participants WHERE (?1 IS NULL OR created_at >= ?1)) AS participant_count;
