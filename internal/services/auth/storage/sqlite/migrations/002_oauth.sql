-- +migrate Up

CREATE TABLE oauth_user_credentials (
    user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE oauth_authorization_codes (
    code TEXT PRIMARY KEY,
    client_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    redirect_uri TEXT NOT NULL,
    code_challenge TEXT NOT NULL,
    code_challenge_method TEXT NOT NULL,
    scope TEXT NOT NULL,
    state TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    used INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE oauth_access_tokens (
    token TEXT PRIMARY KEY,
    client_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    scope TEXT NOT NULL,
    expires_at TEXT NOT NULL
);

CREATE TABLE oauth_pending_authorizations (
    id TEXT PRIMARY KEY,
    response_type TEXT NOT NULL,
    client_id TEXT NOT NULL,
    redirect_uri TEXT NOT NULL,
    scope TEXT NOT NULL,
    state TEXT NOT NULL,
    code_challenge TEXT NOT NULL,
    code_challenge_method TEXT NOT NULL,
    user_id TEXT NOT NULL,
    expires_at TEXT NOT NULL
);

CREATE TABLE oauth_provider_states (
    state TEXT PRIMARY KEY,
    provider TEXT NOT NULL,
    redirect_uri TEXT NOT NULL,
    code_verifier TEXT NOT NULL,
    expires_at TEXT NOT NULL
);

CREATE TABLE oauth_external_identities (
    id TEXT PRIMARY KEY,
    provider TEXT NOT NULL,
    provider_user_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    access_token TEXT NOT NULL,
    refresh_token TEXT NOT NULL,
    scope TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    id_token TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(provider, provider_user_id)
);

-- +migrate Down
DROP TABLE IF EXISTS oauth_external_identities;
DROP TABLE IF EXISTS oauth_provider_states;
DROP TABLE IF EXISTS oauth_pending_authorizations;
DROP TABLE IF EXISTS oauth_access_tokens;
DROP TABLE IF EXISTS oauth_authorization_codes;
DROP TABLE IF EXISTS oauth_user_credentials;
