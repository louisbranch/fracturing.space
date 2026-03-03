-- +migrate Up

ALTER TABLE ai_agents ADD COLUMN handle TEXT NOT NULL DEFAULT '';

UPDATE ai_agents
SET handle = id
WHERE TRIM(handle) = '';

CREATE UNIQUE INDEX IF NOT EXISTS ai_agents_owner_handle_idx
ON ai_agents(owner_user_id, handle);

CREATE TABLE ai_campaign_turns (
    id TEXT PRIMARY KEY,
    campaign_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    agent_id TEXT NOT NULL,
    requester_user_id TEXT NOT NULL,
    participant_id TEXT NOT NULL,
    participant_name TEXT NOT NULL,
    correlation_message_id TEXT NOT NULL,
    input_text TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX ai_campaign_turns_campaign_id_idx ON ai_campaign_turns(campaign_id, id);
CREATE INDEX ai_campaign_turns_agent_id_idx ON ai_campaign_turns(agent_id, id);

CREATE TABLE ai_campaign_turn_events (
    sequence_id INTEGER PRIMARY KEY AUTOINCREMENT,
    campaign_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    turn_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    content TEXT NOT NULL,
    participant_visible INTEGER NOT NULL,
    correlation_message_id TEXT NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE INDEX ai_campaign_turn_events_campaign_seq_idx ON ai_campaign_turn_events(campaign_id, sequence_id);

-- +migrate Down
DROP INDEX IF EXISTS ai_campaign_turn_events_campaign_seq_idx;
DROP TABLE IF EXISTS ai_campaign_turn_events;
DROP INDEX IF EXISTS ai_campaign_turns_agent_id_idx;
DROP INDEX IF EXISTS ai_campaign_turns_campaign_id_idx;
DROP TABLE IF EXISTS ai_campaign_turns;
DROP INDEX IF EXISTS ai_agents_owner_handle_idx;
