-- name: PutMagicLink :exec
INSERT INTO magic_links (
    token, user_id, email, pending_id, created_at, expires_at, used_at
) VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetMagicLink :one
SELECT * FROM magic_links WHERE token = ?;

-- name: MarkMagicLinkUsed :exec
UPDATE magic_links SET used_at = ? WHERE token = ?;
