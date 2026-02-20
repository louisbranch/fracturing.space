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
    updated_at INTEGER NOT NULL,
    provider_grant_id TEXT NOT NULL DEFAULT ''
);

CREATE INDEX ai_agents_owner_id_idx ON ai_agents(owner_user_id, id);
CREATE INDEX ai_agents_owner_provider_grant_id_idx ON ai_agents(owner_user_id, provider_grant_id);

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

CREATE TABLE ai_access_requests (
    id TEXT PRIMARY KEY,
    requester_user_id TEXT NOT NULL,
    owner_user_id TEXT NOT NULL,
    agent_id TEXT NOT NULL,
    scope TEXT NOT NULL,
    request_note TEXT NOT NULL,
    status TEXT NOT NULL,
    reviewer_user_id TEXT NOT NULL,
    review_note TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    reviewed_at INTEGER
);

CREATE INDEX ai_access_requests_requester_id_idx ON ai_access_requests(requester_user_id, id);
CREATE INDEX ai_access_requests_owner_id_idx ON ai_access_requests(owner_user_id, id);

CREATE TABLE ai_audit_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_name TEXT NOT NULL,
    actor_user_id TEXT NOT NULL,
    owner_user_id TEXT NOT NULL,
    requester_user_id TEXT NOT NULL,
    agent_id TEXT NOT NULL,
    access_request_id TEXT NOT NULL,
    outcome TEXT NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE INDEX ai_audit_events_actor_id_idx ON ai_audit_events(actor_user_id, id);
CREATE INDEX ai_audit_events_owner_id_idx ON ai_audit_events(owner_user_id, id);
CREATE INDEX ai_audit_events_requester_id_idx ON ai_audit_events(requester_user_id, id);

-- +migrate Down
DROP INDEX IF EXISTS ai_audit_events_requester_id_idx;
DROP INDEX IF EXISTS ai_audit_events_owner_id_idx;
DROP INDEX IF EXISTS ai_audit_events_actor_id_idx;
DROP TABLE IF EXISTS ai_audit_events;
DROP TABLE IF EXISTS ai_access_requests;
DROP INDEX IF EXISTS ai_provider_connect_sessions_owner_id_idx;
DROP TABLE IF EXISTS ai_provider_connect_sessions;
DROP INDEX IF EXISTS ai_provider_grants_owner_id_idx;
DROP TABLE IF EXISTS ai_provider_grants;
DROP INDEX IF EXISTS ai_agents_owner_provider_grant_id_idx;
DROP INDEX IF EXISTS ai_agents_owner_id_idx;
DROP TABLE IF EXISTS ai_agents;
DROP INDEX IF EXISTS ai_credentials_owner_id_idx;
DROP TABLE IF EXISTS ai_credentials;
