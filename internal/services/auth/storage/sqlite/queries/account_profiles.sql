-- name: GetAccountProfile :one
SELECT user_id, name, locale, avatar_set_id, avatar_asset_id, created_at, updated_at
FROM account_profiles
WHERE user_id = ?;

-- name: PutAccountProfile :exec
INSERT INTO account_profiles (
    user_id, name, locale, avatar_set_id, avatar_asset_id, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(user_id) DO UPDATE SET
    name = excluded.name,
    locale = excluded.locale,
    avatar_set_id = excluded.avatar_set_id,
    avatar_asset_id = excluded.avatar_asset_id,
    updated_at = excluded.updated_at;
