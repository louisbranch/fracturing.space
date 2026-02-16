-- +migrate Up

CREATE TABLE ai_provider_grants (
    id TEXT PRIMARY KEY,
    owner_user_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    granted_scopes TEXT NOT NULL,
    token_ciphertext TEXT NOT NULL,
    refresh_supported INTEGER NOT NULL,
    status TEXT NOT NULL,
    last_refresh_error TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    revoked_at INTEGER,
    expires_at INTEGER,
    last_refreshed_at INTEGER
);

CREATE INDEX ai_provider_grants_owner_id_idx ON ai_provider_grants(owner_user_id, id);

-- +migrate Down
DROP TABLE IF EXISTS ai_provider_grants;
