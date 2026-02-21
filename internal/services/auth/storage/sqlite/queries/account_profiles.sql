-- name: GetAccountProfile :one
SELECT user_id, name, locale, created_at, updated_at
FROM account_profiles
WHERE user_id = ?;

-- name: PutAccountProfile :exec
INSERT INTO account_profiles (
    user_id, name, locale, created_at, updated_at
) VALUES (?, ?, ?, ?, ?)
ON CONFLICT(user_id) DO UPDATE SET
    name = excluded.name,
    locale = excluded.locale,
    updated_at = excluded.updated_at;
