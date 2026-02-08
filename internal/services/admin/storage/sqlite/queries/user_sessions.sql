-- name: PutUserSession :exec
INSERT OR IGNORE INTO user_sessions (
    session_id, created_at
) VALUES (?, ?);
