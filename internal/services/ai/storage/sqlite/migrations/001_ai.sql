-- Baseline schema for fresh alpha databases.

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
CREATE UNIQUE INDEX ai_credentials_owner_active_label_idx ON ai_credentials(owner_user_id, lower(trim(label)))
WHERE revoked_at IS NULL;
CREATE TABLE ai_agents (
    id TEXT PRIMARY KEY,
    owner_user_id TEXT NOT NULL,
    label TEXT NOT NULL,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    credential_id TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    provider_grant_id TEXT NOT NULL DEFAULT ''
, instructions TEXT NOT NULL DEFAULT '');
CREATE INDEX ai_agents_owner_id_idx ON ai_agents(owner_user_id, id);
CREATE UNIQUE INDEX ai_agents_owner_label_idx ON ai_agents(owner_user_id, lower(trim(label)));
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
CREATE TABLE ai_campaign_artifacts (
	campaign_id TEXT NOT NULL,
	path TEXT NOT NULL,
	content TEXT NOT NULL,
	read_only INTEGER NOT NULL DEFAULT 0,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL,
	PRIMARY KEY (campaign_id, path)
);
CREATE INDEX idx_ai_campaign_artifacts_campaign
	ON ai_campaign_artifacts(campaign_id, path);
CREATE TABLE ai_campaign_debug_turns (
	id TEXT PRIMARY KEY,
	campaign_id TEXT NOT NULL,
	session_id TEXT NOT NULL,
	turn_token TEXT NOT NULL,
	participant_id TEXT NOT NULL,
	provider TEXT NOT NULL,
	model TEXT NOT NULL,
	status TEXT NOT NULL,
	last_error TEXT NOT NULL,
	input_tokens INTEGER NOT NULL DEFAULT 0,
	output_tokens INTEGER NOT NULL DEFAULT 0,
	reasoning_tokens INTEGER NOT NULL DEFAULT 0,
	total_tokens INTEGER NOT NULL DEFAULT 0,
	started_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL,
	completed_at INTEGER,
	entry_count INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_ai_campaign_debug_turns_campaign_session_started
	ON ai_campaign_debug_turns(campaign_id, session_id, started_at DESC, id DESC);
CREATE INDEX idx_ai_campaign_debug_turns_turn_token
	ON ai_campaign_debug_turns(campaign_id, session_id, turn_token);
CREATE TABLE ai_campaign_debug_turn_entries (
	turn_id TEXT NOT NULL,
	sequence INTEGER NOT NULL,
	kind TEXT NOT NULL,
	tool_name TEXT NOT NULL,
	payload TEXT NOT NULL,
	payload_truncated INTEGER NOT NULL DEFAULT 0,
	call_id TEXT NOT NULL,
	response_id TEXT NOT NULL,
	is_error INTEGER NOT NULL DEFAULT 0,
	input_tokens INTEGER NOT NULL DEFAULT 0,
	output_tokens INTEGER NOT NULL DEFAULT 0,
	reasoning_tokens INTEGER NOT NULL DEFAULT 0,
	total_tokens INTEGER NOT NULL DEFAULT 0,
	created_at INTEGER NOT NULL,
	PRIMARY KEY (turn_id, sequence),
	FOREIGN KEY (turn_id) REFERENCES ai_campaign_debug_turns(id) ON DELETE CASCADE
);
CREATE INDEX idx_ai_campaign_debug_turn_entries_turn_created
	ON ai_campaign_debug_turn_entries(turn_id, sequence);
