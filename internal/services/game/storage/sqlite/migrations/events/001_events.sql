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
    system_id TEXT NOT NULL DEFAULT '',
    system_version TEXT NOT NULL DEFAULT '',
    session_id TEXT NOT NULL DEFAULT '',
    request_id TEXT NOT NULL DEFAULT '',
    invocation_id TEXT NOT NULL DEFAULT '',
    actor_type TEXT NOT NULL,
    actor_id TEXT NOT NULL DEFAULT '',
    entity_type TEXT NOT NULL DEFAULT '',
    entity_id TEXT NOT NULL DEFAULT '',
    correlation_id TEXT NOT NULL DEFAULT '',
    causation_id TEXT NOT NULL DEFAULT '',
    payload_json BLOB NOT NULL,
    PRIMARY KEY (campaign_id, seq)
);

CREATE UNIQUE INDEX idx_events_hash ON events(event_hash);
CREATE INDEX idx_events_session ON events(campaign_id, session_id)
    WHERE session_id != '';
CREATE INDEX idx_events_type ON events(campaign_id, event_type);
CREATE INDEX idx_events_system ON events(campaign_id, system_id, system_version)
    WHERE system_id != '';

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

CREATE TABLE projection_apply_outbox (
    campaign_id TEXT NOT NULL,
    seq INTEGER NOT NULL,
    event_type TEXT NOT NULL,
    status TEXT NOT NULL,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    next_attempt_at INTEGER NOT NULL,
    last_error TEXT NOT NULL DEFAULT '',
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (campaign_id, seq),
    FOREIGN KEY (campaign_id, seq) REFERENCES events(campaign_id, seq) ON DELETE CASCADE
);

CREATE INDEX idx_projection_apply_outbox_status_next_attempt
    ON projection_apply_outbox (status, next_attempt_at, seq);

CREATE INDEX idx_projection_apply_outbox_campaign
    ON projection_apply_outbox (campaign_id, seq);

-- +migrate Down
DROP TRIGGER IF EXISTS events_no_delete;
DROP TRIGGER IF EXISTS events_no_update;
DROP INDEX IF EXISTS idx_projection_apply_outbox_campaign;
DROP INDEX IF EXISTS idx_projection_apply_outbox_status_next_attempt;
DROP TABLE IF EXISTS projection_apply_outbox;
DROP INDEX IF EXISTS idx_telemetry_events_timestamp;
DROP INDEX IF EXISTS idx_telemetry_events_campaign_id;
DROP TABLE IF EXISTS telemetry_events;
DROP TABLE IF EXISTS outcome_applied;
DROP TABLE IF EXISTS event_seq;
DROP INDEX IF EXISTS idx_events_system;
DROP INDEX IF EXISTS idx_events_type;
DROP INDEX IF EXISTS idx_events_session;
DROP INDEX IF EXISTS idx_events_hash;
DROP TABLE IF EXISTS events;
