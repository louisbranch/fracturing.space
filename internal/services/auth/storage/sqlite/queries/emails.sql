-- name: PutUserEmail :exec
INSERT INTO user_emails (
    id, user_id, email, verified_at, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(email) DO UPDATE SET
    user_id = excluded.user_id,
    verified_at = excluded.verified_at,
    updated_at = excluded.updated_at;

-- name: GetUserEmailByEmail :one
SELECT * FROM user_emails WHERE email = ?;

-- name: ListUserEmailsByUser :many
SELECT * FROM user_emails WHERE user_id = ? ORDER BY email;

-- name: UpdateUserEmailVerified :exec
UPDATE user_emails SET verified_at = ?, updated_at = ? WHERE email = ? AND user_id = ?;
