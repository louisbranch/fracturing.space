-- +migrate Up

DROP TRIGGER IF EXISTS events_no_delete;
DROP TRIGGER IF EXISTS events_no_update;
DROP INDEX IF EXISTS idx_events_system;
DROP INDEX IF EXISTS idx_events_type;
DROP INDEX IF EXISTS idx_events_session;
DROP INDEX IF EXISTS idx_events_hash;
DROP TABLE IF EXISTS events;

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
DROP INDEX IF EXISTS idx_events_system;
DROP INDEX IF EXISTS idx_events_type;
DROP INDEX IF EXISTS idx_events_session;
DROP INDEX IF EXISTS idx_events_hash;
DROP TABLE IF EXISTS events;
