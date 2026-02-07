-- +migrate Up

-- Campaign Layer
CREATE TABLE campaigns (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    game_system TEXT NOT NULL,
    status TEXT NOT NULL,
    gm_mode TEXT NOT NULL,
    participant_count INTEGER NOT NULL DEFAULT 0,
    character_count INTEGER NOT NULL DEFAULT 0,
    theme_prompt TEXT NOT NULL DEFAULT '',
    parent_campaign_id TEXT,
    fork_event_seq INTEGER,
    origin_campaign_id TEXT,
    created_at TEXT NOT NULL,
    last_activity_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    completed_at TEXT,
    archived_at TEXT
);

CREATE TABLE participants (
    campaign_id TEXT NOT NULL,
    id TEXT NOT NULL,
    display_name TEXT NOT NULL,
    role TEXT NOT NULL,
    controller TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (campaign_id, id),
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

CREATE TABLE characters (
    campaign_id TEXT NOT NULL,
    id TEXT NOT NULL,
    name TEXT NOT NULL,
    kind TEXT NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (campaign_id, id),
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

CREATE TABLE control_defaults (
    campaign_id TEXT NOT NULL,
    character_id TEXT NOT NULL,
    is_gm INTEGER NOT NULL DEFAULT 0,
    participant_id TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (campaign_id, character_id)
);

-- Session Layer
CREATE TABLE sessions (
    campaign_id TEXT NOT NULL,
    id TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    started_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    ended_at TEXT,
    PRIMARY KEY (campaign_id, id),
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

CREATE INDEX idx_sessions_active ON sessions(campaign_id)
    WHERE status = 'ACTIVE';

CREATE TABLE campaign_active_session (
    campaign_id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
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

CREATE TABLE outcome_applied (
    session_id TEXT NOT NULL,
    request_id TEXT NOT NULL,
    PRIMARY KEY (session_id, request_id)
);

-- Campaign Events (event sourcing)
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

-- Snapshots (continuity checkpoints)
CREATE TABLE snapshots (
    campaign_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    event_seq INTEGER NOT NULL,
    character_states_json BLOB NOT NULL,
    gm_state_json BLOB NOT NULL,
    system_state_json BLOB NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    PRIMARY KEY (campaign_id, session_id),
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

CREATE INDEX idx_snapshots_seq ON snapshots(campaign_id, event_seq DESC);

-- Daggerheart Extension Tables
CREATE TABLE daggerheart_character_profiles (
    campaign_id TEXT NOT NULL,
    character_id TEXT NOT NULL,
    hp_max INTEGER NOT NULL DEFAULT 6,
    stress_max INTEGER NOT NULL DEFAULT 6,
    evasion INTEGER NOT NULL DEFAULT 10,
    major_threshold INTEGER NOT NULL DEFAULT 8,
    severe_threshold INTEGER NOT NULL DEFAULT 12,
    agility INTEGER NOT NULL DEFAULT 0,
    strength INTEGER NOT NULL DEFAULT 0,
    finesse INTEGER NOT NULL DEFAULT 0,
    instinct INTEGER NOT NULL DEFAULT 0,
    presence INTEGER NOT NULL DEFAULT 0,
    knowledge INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (campaign_id, character_id),
    FOREIGN KEY (campaign_id, character_id)
        REFERENCES characters(campaign_id, id) ON DELETE CASCADE
);

CREATE TABLE daggerheart_character_states (
    campaign_id TEXT NOT NULL,
    character_id TEXT NOT NULL,
    hp INTEGER NOT NULL DEFAULT 6,
    hope INTEGER NOT NULL DEFAULT 2,
    stress INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (campaign_id, character_id),
    FOREIGN KEY (campaign_id, character_id)
        REFERENCES characters(campaign_id, id) ON DELETE CASCADE
);

CREATE TABLE daggerheart_snapshots (
    campaign_id TEXT PRIMARY KEY,
    gm_fear INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

-- +migrate Down
DROP TABLE IF EXISTS daggerheart_snapshots;
DROP TABLE IF EXISTS daggerheart_character_states;
DROP TABLE IF EXISTS daggerheart_character_profiles;
DROP INDEX IF EXISTS idx_snapshots_seq;
DROP TABLE IF EXISTS snapshots;
DROP TABLE IF EXISTS campaign_event_seq;
DROP INDEX IF EXISTS idx_campaign_events_type;
DROP INDEX IF EXISTS idx_campaign_events_session;
DROP TABLE IF EXISTS campaign_events;
DROP TABLE IF EXISTS outcome_applied;
DROP TABLE IF EXISTS session_event_seq;
DROP TABLE IF EXISTS session_events;
DROP TABLE IF EXISTS campaign_active_session;
DROP INDEX IF EXISTS idx_sessions_active;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS control_defaults;
DROP TABLE IF EXISTS characters;
DROP TABLE IF EXISTS participants;
DROP TABLE IF EXISTS campaigns;
