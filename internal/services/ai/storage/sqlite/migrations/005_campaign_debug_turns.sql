-- +migrate Up

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

-- +migrate Down
DROP INDEX IF EXISTS idx_ai_campaign_debug_turn_entries_turn_created;
DROP TABLE IF EXISTS ai_campaign_debug_turn_entries;
DROP INDEX IF EXISTS idx_ai_campaign_debug_turns_turn_token;
DROP INDEX IF EXISTS idx_ai_campaign_debug_turns_campaign_session_started;
DROP TABLE IF EXISTS ai_campaign_debug_turns;
