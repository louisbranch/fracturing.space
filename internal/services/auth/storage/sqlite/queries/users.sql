-- name: GetUser :one
SELECT * FROM users WHERE id = ?;

-- name: PutUser :exec
INSERT INTO users (
    id, email, locale, created_at, updated_at
) VALUES (?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    email = excluded.email,
    locale = excluded.locale,
    updated_at = excluded.updated_at;

-- name: ListUsersPaged :many
SELECT * FROM users
WHERE id > ?
ORDER BY id
LIMIT ?;

-- name: ListUsersPagedFirst :many
SELECT * FROM users
ORDER BY id
LIMIT ?;
