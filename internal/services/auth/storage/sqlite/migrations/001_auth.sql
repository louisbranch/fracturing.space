-- +migrate Up

CREATE TABLE users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    locale TEXT NOT NULL,
    recovery_code_hash TEXT NOT NULL DEFAULT '',
    recovery_reserved_session_id TEXT NOT NULL DEFAULT '',
    recovery_reserved_until INTEGER,
    recovery_code_updated_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
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

CREATE TABLE passkeys (
    credential_id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_json TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    last_used_at INTEGER
);

CREATE TABLE passkey_sessions (
    id TEXT PRIMARY KEY,
    kind TEXT NOT NULL,
    user_id TEXT,
    session_json TEXT NOT NULL,
    expires_at INTEGER NOT NULL
);

CREATE TABLE registration_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL UNIQUE,
    username TEXT NOT NULL UNIQUE,
    locale TEXT NOT NULL,
    recovery_code_hash TEXT NOT NULL,
    expires_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE recovery_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE TABLE web_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    revoked_at INTEGER
);

CREATE TABLE auth_integration_outbox (
    id TEXT PRIMARY KEY,
    event_type TEXT NOT NULL,
    payload_json TEXT NOT NULL DEFAULT '{}',
    dedupe_key TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    next_attempt_at INTEGER NOT NULL,
    lease_owner TEXT NOT NULL DEFAULT '',
    lease_expires_at INTEGER,
    last_error TEXT NOT NULL DEFAULT '',
    processed_at INTEGER,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE UNIQUE INDEX auth_integration_outbox_dedupe_unique_idx
ON auth_integration_outbox(dedupe_key)
WHERE dedupe_key <> '';

CREATE INDEX auth_integration_outbox_lease_idx
ON auth_integration_outbox(status, next_attempt_at, lease_expires_at, id);

-- +migrate Down
DROP INDEX IF EXISTS auth_integration_outbox_lease_idx;
DROP INDEX IF EXISTS auth_integration_outbox_dedupe_unique_idx;
DROP TABLE IF EXISTS auth_integration_outbox;
DROP TABLE IF EXISTS web_sessions;
DROP TABLE IF EXISTS recovery_sessions;
DROP TABLE IF EXISTS registration_sessions;
DROP TABLE IF EXISTS passkey_sessions;
DROP TABLE IF EXISTS passkeys;
DROP TABLE IF EXISTS oauth_pending_authorizations;
DROP TABLE IF EXISTS oauth_access_tokens;
DROP TABLE IF EXISTS oauth_authorization_codes;
DROP TABLE IF EXISTS users;
