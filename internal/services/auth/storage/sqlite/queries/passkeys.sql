-- name: PutPasskey :exec
INSERT INTO passkeys (
    credential_id, user_id, credential_json, created_at, updated_at, last_used_at
) VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(credential_id) DO UPDATE SET
    user_id = excluded.user_id,
    credential_json = excluded.credential_json,
    updated_at = excluded.updated_at,
    last_used_at = excluded.last_used_at;

-- name: GetPasskey :one
SELECT * FROM passkeys WHERE credential_id = ?;

-- name: ListPasskeysByUser :many
SELECT * FROM passkeys WHERE user_id = ? ORDER BY credential_id;

-- name: DeletePasskey :exec
DELETE FROM passkeys WHERE credential_id = ?;

-- name: PutPasskeySession :exec
INSERT INTO passkey_sessions (
    id, kind, user_id, session_json, expires_at
) VALUES (?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    kind = excluded.kind,
    user_id = excluded.user_id,
    session_json = excluded.session_json,
    expires_at = excluded.expires_at;

-- name: GetPasskeySession :one
SELECT * FROM passkey_sessions WHERE id = ?;

-- name: DeletePasskeySession :exec
DELETE FROM passkey_sessions WHERE id = ?;

-- name: DeleteExpiredPasskeySessions :exec
DELETE FROM passkey_sessions WHERE expires_at <= ?;
