-- +migrate Up

CREATE TABLE events (
    campaign_id TEXT NOT NULL,
    seq INTEGER NOT NULL,
    event_hash TEXT NOT NULL,
    prev_event_hash TEXT NOT NULL DEFAULT '',
    chain_hash TEXT NOT NULL,
    signature_key_id TEXT NOT NULL,
    event_signature TEXT NOT NULL,
    timestamp INTEGER NOT NULL,
    event_type TEXT NOT NULL,
    session_id TEXT NOT NULL DEFAULT '',
    request_id TEXT NOT NULL DEFAULT '',
    invocation_id TEXT NOT NULL DEFAULT '',
    actor_type TEXT NOT NULL,
    actor_id TEXT NOT NULL DEFAULT '',
    entity_type TEXT NOT NULL DEFAULT '',
    entity_id TEXT NOT NULL DEFAULT '',
    payload_json BLOB NOT NULL,
    PRIMARY KEY (campaign_id, seq)
);

CREATE UNIQUE INDEX idx_events_hash ON events(event_hash);
CREATE INDEX idx_events_session ON events(campaign_id, session_id)
    WHERE session_id != '';
CREATE INDEX idx_events_type ON events(campaign_id, event_type);

CREATE TABLE event_seq (
    campaign_id TEXT PRIMARY KEY,
    next_seq INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE outcome_applied (
    campaign_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    request_id TEXT NOT NULL,
    PRIMARY KEY (campaign_id, session_id, request_id)
);

CREATE TABLE telemetry_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  timestamp INTEGER NOT NULL,
  event_name TEXT NOT NULL,
  severity TEXT NOT NULL,
  campaign_id TEXT,
  session_id TEXT,
  actor_type TEXT,
  actor_id TEXT,
  request_id TEXT,
  invocation_id TEXT,
  trace_id TEXT,
  span_id TEXT,
  attributes_json BLOB
);

CREATE INDEX idx_telemetry_events_campaign_id ON telemetry_events (campaign_id);
CREATE INDEX idx_telemetry_events_timestamp ON telemetry_events (timestamp);

CREATE TRIGGER events_no_update
BEFORE UPDATE ON events
BEGIN
    SELECT RAISE(FAIL, 'events are append-only');
END;

CREATE TRIGGER events_no_delete
BEFORE DELETE ON events
BEGIN
    SELECT RAISE(FAIL, 'events are append-only');
END;

-- +migrate Down

DROP TRIGGER IF EXISTS events_no_delete;
DROP TRIGGER IF EXISTS events_no_update;
DROP INDEX IF EXISTS idx_telemetry_events_timestamp;
DROP INDEX IF EXISTS idx_telemetry_events_campaign_id;
DROP TABLE IF EXISTS telemetry_events;
DROP TABLE IF EXISTS outcome_applied;
DROP TABLE IF EXISTS event_seq;
DROP INDEX IF EXISTS idx_events_type;
DROP INDEX IF EXISTS idx_events_session;
DROP INDEX IF EXISTS idx_events_hash;
DROP TABLE IF EXISTS events;
