-- +migrate Up

CREATE TABLE ai_provider_connect_sessions (
    id TEXT PRIMARY KEY,
    owner_user_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    status TEXT NOT NULL,
    requested_scopes TEXT NOT NULL,
    state_hash TEXT NOT NULL,
    code_verifier_ciphertext TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    completed_at INTEGER
);

CREATE INDEX ai_provider_connect_sessions_owner_id_idx ON ai_provider_connect_sessions(owner_user_id, id);

-- +migrate Down
DROP TABLE IF EXISTS ai_provider_connect_sessions;
