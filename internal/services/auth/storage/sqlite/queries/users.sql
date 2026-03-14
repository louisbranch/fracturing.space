-- name: GetUser :one
SELECT *
FROM users
WHERE id = ?;

-- name: GetUserByUsername :one
SELECT *
FROM users
WHERE username = ?;

-- name: PutUser :exec
INSERT INTO users (
    id, username, locale, recovery_code_hash, recovery_reserved_session_id,
    recovery_reserved_until, recovery_code_updated_at, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    username = excluded.username,
    locale = excluded.locale,
    recovery_code_hash = excluded.recovery_code_hash,
    recovery_reserved_session_id = excluded.recovery_reserved_session_id,
    recovery_reserved_until = excluded.recovery_reserved_until,
    recovery_code_updated_at = excluded.recovery_code_updated_at,
    updated_at = excluded.updated_at;

-- name: ReserveUserRecoverySession :exec
UPDATE users
SET recovery_reserved_session_id = ?, recovery_reserved_until = ?, updated_at = ?
WHERE id = ?;

-- name: RotateUserRecoveryCode :exec
UPDATE users
SET recovery_code_hash = ?, recovery_reserved_session_id = '', recovery_reserved_until = NULL,
    recovery_code_updated_at = ?, updated_at = ?
WHERE id = ?;

-- name: ClearUserRecoveryReservation :exec
UPDATE users
SET recovery_reserved_session_id = '', recovery_reserved_until = NULL, updated_at = ?
WHERE id = ?;

-- name: ListUsersPaged :many
SELECT *
FROM users
WHERE id > ?
ORDER BY id
LIMIT ?;

-- name: ListUsersPagedFirst :many
SELECT *
FROM users
ORDER BY id
LIMIT ?;
