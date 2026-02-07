-- +migrate Up

-- Drop old session event tables (per AGENTS.md schema rules: prefer DROP and CREATE over ALTER)
DROP TABLE IF EXISTS session_event_seq;
DROP TABLE IF EXISTS session_events;
DROP TABLE IF EXISTS outcome_applied;

-- Drop old campaign event tables
DROP TABLE IF EXISTS campaign_event_seq;
DROP INDEX IF EXISTS idx_campaign_events_type;
DROP INDEX IF EXISTS idx_campaign_events_session;
DROP TABLE IF EXISTS campaign_events;

-- Create unified events table
CREATE TABLE events (
    campaign_id TEXT NOT NULL,
    seq INTEGER NOT NULL,
    event_hash TEXT NOT NULL,
    timestamp TEXT NOT NULL,
    event_type TEXT NOT NULL,
    -- Correlation fields
    session_id TEXT NOT NULL DEFAULT '',
    request_id TEXT NOT NULL DEFAULT '',
    invocation_id TEXT NOT NULL DEFAULT '',
    -- Actor
    actor_type TEXT NOT NULL,
    actor_id TEXT NOT NULL DEFAULT '',
    -- Entity
    entity_type TEXT NOT NULL DEFAULT '',
    entity_id TEXT NOT NULL DEFAULT '',
    -- Payload (efficient binary)
    payload_json BLOB NOT NULL,
    PRIMARY KEY (campaign_id, seq),
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

-- Content-addressed lookup
CREATE UNIQUE INDEX idx_events_hash ON events(event_hash);

-- Session filtering
CREATE INDEX idx_events_session ON events(campaign_id, session_id)
    WHERE session_id != '';

-- Event type filtering
CREATE INDEX idx_events_type ON events(campaign_id, event_type);

-- Sequence counter table (one row per campaign)
CREATE TABLE event_seq (
    campaign_id TEXT PRIMARY KEY,
    next_seq INTEGER NOT NULL DEFAULT 1,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

-- Track applied outcomes for idempotency (keyed by campaign to allow fork isolation)
CREATE TABLE outcome_applied (
    campaign_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    request_id TEXT NOT NULL,
    PRIMARY KEY (campaign_id, session_id, request_id),
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

-- +migrate Down
DROP TABLE IF EXISTS outcome_applied;
DROP TABLE IF EXISTS event_seq;
DROP INDEX IF EXISTS idx_events_type;
DROP INDEX IF EXISTS idx_events_session;
DROP INDEX IF EXISTS idx_events_hash;
DROP TABLE IF EXISTS events;

-- Restore old tables (simplified - actual data would be lost)
CREATE TABLE campaign_events (
    campaign_id TEXT NOT NULL,
    seq INTEGER NOT NULL,
    timestamp TEXT NOT NULL,
    event_type TEXT NOT NULL,
    session_id TEXT NOT NULL DEFAULT '',
    actor_type TEXT NOT NULL,
    actor_id TEXT NOT NULL DEFAULT '',
    entity_type TEXT NOT NULL DEFAULT '',
    entity_id TEXT NOT NULL DEFAULT '',
    payload_json BLOB NOT NULL,
    PRIMARY KEY (campaign_id, seq),
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

CREATE INDEX idx_campaign_events_session ON campaign_events(campaign_id, session_id)
    WHERE session_id != '';
CREATE INDEX idx_campaign_events_type ON campaign_events(campaign_id, event_type);

CREATE TABLE campaign_event_seq (
    campaign_id TEXT PRIMARY KEY,
    next_seq INTEGER NOT NULL DEFAULT 1,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

CREATE TABLE session_events (
    session_id TEXT NOT NULL,
    seq INTEGER NOT NULL,
    timestamp TEXT NOT NULL,
    event_type TEXT NOT NULL,
    request_id TEXT NOT NULL DEFAULT '',
    invocation_id TEXT NOT NULL DEFAULT '',
    participant_id TEXT NOT NULL DEFAULT '',
    character_id TEXT NOT NULL DEFAULT '',
    payload_json BLOB NOT NULL,
    PRIMARY KEY (session_id, seq)
);

CREATE TABLE session_event_seq (
    session_id TEXT PRIMARY KEY,
    next_seq INTEGER NOT NULL DEFAULT 1
);
