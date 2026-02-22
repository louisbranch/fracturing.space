-- name: PutContact :exec
INSERT INTO user_contacts (
    owner_user_id, contact_user_id, created_at, updated_at
) VALUES (?, ?, ?, ?)
ON CONFLICT(owner_user_id, contact_user_id) DO UPDATE SET
    updated_at = excluded.updated_at;

-- name: DeleteContact :exec
DELETE FROM user_contacts
WHERE owner_user_id = ? AND contact_user_id = ?;

-- name: GetContact :one
SELECT owner_user_id, contact_user_id, created_at, updated_at
FROM user_contacts
WHERE owner_user_id = ? AND contact_user_id = ?;

-- name: ListContactsPagedFirst :many
SELECT owner_user_id, contact_user_id, created_at, updated_at
FROM user_contacts
WHERE owner_user_id = ?
ORDER BY contact_user_id
LIMIT ?;

-- name: ListContactsPaged :many
SELECT owner_user_id, contact_user_id, created_at, updated_at
FROM user_contacts
WHERE owner_user_id = ? AND contact_user_id > ?
ORDER BY contact_user_id
LIMIT ?;
