-- name: GetUser :one
SELECT users.id, user_emails.email, users.locale, users.created_at, users.updated_at
FROM users
JOIN user_emails
    ON users.id = user_emails.user_id
    AND user_emails.is_primary = 1
WHERE users.id = ?;

-- name: PutUser :exec
INSERT INTO users (
    id, locale, created_at, updated_at
) VALUES (?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    locale = excluded.locale,
    updated_at = excluded.updated_at;

-- name: ListUsersPaged :many
SELECT users.id, user_emails.email, users.locale, users.created_at, users.updated_at
FROM users
JOIN user_emails
    ON users.id = user_emails.user_id
    AND user_emails.is_primary = 1
WHERE users.id > ?
ORDER BY users.id
LIMIT ?;

-- name: ListUsersPagedFirst :many
SELECT users.id, user_emails.email, users.locale, users.created_at, users.updated_at
FROM users
JOIN user_emails
    ON users.id = user_emails.user_id
    AND user_emails.is_primary = 1
ORDER BY users.id
LIMIT ?;
