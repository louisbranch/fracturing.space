-- +migrate Up
CREATE TABLE IF NOT EXISTS ai_audit_events (
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

CREATE INDEX IF NOT EXISTS ai_audit_events_actor_id_idx ON ai_audit_events(actor_user_id, id);
CREATE INDEX IF NOT EXISTS ai_audit_events_owner_id_idx ON ai_audit_events(owner_user_id, id);
CREATE INDEX IF NOT EXISTS ai_audit_events_requester_id_idx ON ai_audit_events(requester_user_id, id);

-- +migrate Down
DROP INDEX IF EXISTS ai_audit_events_requester_id_idx;
DROP INDEX IF EXISTS ai_audit_events_owner_id_idx;
DROP INDEX IF EXISTS ai_audit_events_actor_id_idx;
DROP TABLE IF EXISTS ai_audit_events;
