-- +migrate Up

CREATE TABLE ai_credentials (
    id TEXT PRIMARY KEY,
    owner_user_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    label TEXT NOT NULL,
    secret_ciphertext TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    revoked_at INTEGER
);

CREATE INDEX ai_credentials_owner_id_idx ON ai_credentials(owner_user_id, id);

CREATE TABLE ai_agents (
    id TEXT PRIMARY KEY,
    owner_user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    credential_id TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX ai_agents_owner_id_idx ON ai_agents(owner_user_id, id);

-- +migrate Down
DROP TABLE IF EXISTS ai_agents;
DROP TABLE IF EXISTS ai_credentials;
