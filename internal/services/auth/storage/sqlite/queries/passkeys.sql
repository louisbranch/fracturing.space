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

-- name: DeletePasskeysByUser :exec
DELETE FROM passkeys WHERE user_id = ?;

-- name: DeletePasskeysByUserExcept :exec
DELETE FROM passkeys WHERE user_id = ? AND credential_id <> ?;

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

-- name: PutRegistrationSession :exec
INSERT INTO registration_sessions (
    id, user_id, username, locale, recovery_code_hash, credential_id, credential_json, expires_at, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    user_id = excluded.user_id,
    username = excluded.username,
    locale = excluded.locale,
    recovery_code_hash = excluded.recovery_code_hash,
    credential_id = excluded.credential_id,
    credential_json = excluded.credential_json,
    expires_at = excluded.expires_at,
    updated_at = excluded.updated_at;

-- name: GetRegistrationSession :one
SELECT * FROM registration_sessions WHERE id = ?;

-- name: GetRegistrationSessionByUsername :one
SELECT * FROM registration_sessions WHERE username = ?;

-- name: DeleteRegistrationSession :exec
DELETE FROM registration_sessions WHERE id = ?;

-- name: DeleteExpiredRegistrationSessions :exec
DELETE FROM registration_sessions WHERE expires_at <= ?;

-- name: PutRecoverySession :exec
INSERT INTO recovery_sessions (
    id, user_id, expires_at, created_at
) VALUES (?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    user_id = excluded.user_id,
    expires_at = excluded.expires_at,
    created_at = excluded.created_at;

-- name: GetRecoverySession :one
SELECT * FROM recovery_sessions WHERE id = ?;

-- name: DeleteRecoverySession :exec
DELETE FROM recovery_sessions WHERE id = ?;

-- name: DeleteExpiredRecoverySessions :exec
DELETE FROM recovery_sessions WHERE expires_at <= ?;
