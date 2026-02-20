-- +migrate Up

PRAGMA foreign_keys = OFF;

DROP INDEX IF EXISTS idx_session_spotlight_session;
DROP TABLE IF EXISTS session_spotlight;
DROP INDEX IF EXISTS idx_session_gates_open;
DROP TABLE IF EXISTS session_gates;
DROP TABLE IF EXISTS daggerheart_adversaries;
DROP TABLE IF EXISTS daggerheart_countdowns;
DROP TABLE IF EXISTS daggerheart_snapshots;
DROP TABLE IF EXISTS daggerheart_character_states;
DROP TABLE IF EXISTS daggerheart_character_profiles;
DROP INDEX IF EXISTS idx_snapshots_seq;
DROP TABLE IF EXISTS snapshots;
DROP INDEX IF EXISTS idx_invites_campaign_recipient;
DROP INDEX IF EXISTS idx_invites_recipient_user;
DROP INDEX IF EXISTS idx_invites_participant;
DROP INDEX IF EXISTS idx_invites_campaign;
DROP TABLE IF EXISTS invites;
DROP TABLE IF EXISTS campaign_active_session;
DROP INDEX IF EXISTS idx_sessions_active;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS characters;
DROP INDEX IF EXISTS idx_participant_claims_participant;
DROP TABLE IF EXISTS participant_claims;
DROP INDEX IF EXISTS idx_participants_campaign_user;
DROP INDEX IF EXISTS idx_participants_user_id;
DROP TABLE IF EXISTS participants;
DROP TABLE IF EXISTS campaigns;

-- Campaign layer
CREATE TABLE campaigns (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    locale TEXT NOT NULL DEFAULT 'en-US',
    game_system TEXT NOT NULL,
    status TEXT NOT NULL,
    gm_mode TEXT NOT NULL,
    intent TEXT NOT NULL DEFAULT 'STANDARD',
    access_policy TEXT NOT NULL DEFAULT 'PRIVATE',
    participant_count INTEGER NOT NULL DEFAULT 0,
    character_count INTEGER NOT NULL DEFAULT 0,
    theme_prompt TEXT NOT NULL DEFAULT '',
    parent_campaign_id TEXT,
    fork_event_seq INTEGER,
    origin_campaign_id TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    completed_at INTEGER,
    archived_at INTEGER
);

CREATE TABLE participants (
    campaign_id TEXT NOT NULL,
    id TEXT NOT NULL,
    user_id TEXT NOT NULL DEFAULT '',
    display_name TEXT NOT NULL,
    role TEXT NOT NULL,
    controller TEXT NOT NULL,
    campaign_access TEXT NOT NULL DEFAULT 'MEMBER',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (campaign_id, id),
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

CREATE INDEX idx_participants_user_id ON participants(user_id);
CREATE UNIQUE INDEX idx_participants_campaign_user
    ON participants(campaign_id, user_id)
    WHERE user_id != '';

CREATE TABLE participant_claims (
    campaign_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    participant_id TEXT NOT NULL,
    claimed_at INTEGER NOT NULL,
    PRIMARY KEY (campaign_id, user_id),
    UNIQUE (campaign_id, participant_id),
    FOREIGN KEY (campaign_id, participant_id) REFERENCES participants(campaign_id, id) ON DELETE CASCADE
);

CREATE INDEX idx_participant_claims_participant ON participant_claims(participant_id);

CREATE TABLE characters (
    campaign_id TEXT NOT NULL,
    id TEXT NOT NULL,
    controller_participant_id TEXT,
    name TEXT NOT NULL,
    kind TEXT NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (campaign_id, id),
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

-- Session layer
CREATE TABLE sessions (
    campaign_id TEXT NOT NULL,
    id TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    started_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    ended_at INTEGER,
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

-- Invitations
CREATE TABLE invites (
    id TEXT PRIMARY KEY,
    campaign_id TEXT NOT NULL,
    participant_id TEXT NOT NULL,
    recipient_user_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    created_by_participant_id TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE,
    FOREIGN KEY (campaign_id, participant_id) REFERENCES participants(campaign_id, id) ON DELETE CASCADE
);

CREATE INDEX idx_invites_campaign ON invites(campaign_id);
CREATE INDEX idx_invites_participant ON invites(participant_id);
CREATE INDEX idx_invites_recipient_user ON invites(recipient_user_id);
CREATE INDEX idx_invites_campaign_recipient ON invites(campaign_id, recipient_user_id);

-- Snapshots
CREATE TABLE snapshots (
    campaign_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    event_seq INTEGER NOT NULL,
    character_states_json BLOB NOT NULL,
    gm_state_json BLOB NOT NULL,
    system_state_json BLOB NOT NULL DEFAULT '{}',
    created_at INTEGER NOT NULL,
    PRIMARY KEY (campaign_id, session_id),
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

CREATE INDEX idx_snapshots_seq ON snapshots(campaign_id, event_seq DESC);

-- Daggerheart projection tables
CREATE TABLE daggerheart_character_profiles (
    campaign_id TEXT NOT NULL,
    character_id TEXT NOT NULL,
    level INTEGER NOT NULL DEFAULT 1,
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
    proficiency INTEGER NOT NULL DEFAULT 0,
    armor_score INTEGER NOT NULL DEFAULT 0,
    armor_max INTEGER NOT NULL DEFAULT 0,
    experiences_json TEXT NOT NULL DEFAULT '[]',
    PRIMARY KEY (campaign_id, character_id),
    FOREIGN KEY (campaign_id, character_id)
        REFERENCES characters(campaign_id, id) ON DELETE CASCADE
);

CREATE TABLE daggerheart_character_states (
    campaign_id TEXT NOT NULL,
    character_id TEXT NOT NULL,
    hp INTEGER NOT NULL DEFAULT 6,
    hope INTEGER NOT NULL DEFAULT 2,
    hope_max INTEGER NOT NULL DEFAULT 6,
    stress INTEGER NOT NULL DEFAULT 0,
    armor INTEGER NOT NULL DEFAULT 0,
    conditions_json TEXT NOT NULL DEFAULT '[]',
    life_state TEXT NOT NULL DEFAULT 'alive',
    PRIMARY KEY (campaign_id, character_id),
    FOREIGN KEY (campaign_id, character_id)
        REFERENCES characters(campaign_id, id) ON DELETE CASCADE
);

CREATE TABLE daggerheart_snapshots (
    campaign_id TEXT PRIMARY KEY,
    gm_fear INTEGER NOT NULL DEFAULT 0,
    consecutive_short_rests INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

CREATE TABLE daggerheart_countdowns (
    campaign_id TEXT NOT NULL,
    countdown_id TEXT NOT NULL,
    name TEXT NOT NULL,
    kind TEXT NOT NULL,
    current INTEGER NOT NULL,
    max INTEGER NOT NULL,
    direction TEXT NOT NULL,
    looping INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (campaign_id, countdown_id)
);

CREATE TABLE daggerheart_adversaries (
    campaign_id TEXT NOT NULL,
    adversary_id TEXT NOT NULL,
    name TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT '',
    session_id TEXT,
    notes TEXT NOT NULL DEFAULT '',
    hp INTEGER NOT NULL DEFAULT 6,
    hp_max INTEGER NOT NULL DEFAULT 6,
    stress INTEGER NOT NULL DEFAULT 0,
    stress_max INTEGER NOT NULL DEFAULT 6,
    evasion INTEGER NOT NULL DEFAULT 10,
    major_threshold INTEGER NOT NULL DEFAULT 8,
    severe_threshold INTEGER NOT NULL DEFAULT 12,
    armor INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (campaign_id, adversary_id),
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

-- Session governance
CREATE TABLE session_gates (
    campaign_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    gate_id TEXT NOT NULL,
    gate_type TEXT NOT NULL,
    status TEXT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    created_by_actor_type TEXT NOT NULL,
    created_by_actor_id TEXT NOT NULL DEFAULT '',
    resolved_at INTEGER,
    resolved_by_actor_type TEXT,
    resolved_by_actor_id TEXT,
    metadata_json BLOB,
    resolution_json BLOB,
    PRIMARY KEY (campaign_id, session_id, gate_id)
);

CREATE INDEX idx_session_gates_open ON session_gates(campaign_id, session_id, status);

CREATE TABLE session_spotlight (
    campaign_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    spotlight_type TEXT NOT NULL,
    character_id TEXT NOT NULL DEFAULT '',
    updated_at INTEGER NOT NULL,
    updated_by_actor_type TEXT NOT NULL,
    updated_by_actor_id TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (campaign_id, session_id)
);

CREATE INDEX idx_session_spotlight_session ON session_spotlight(campaign_id, session_id);

PRAGMA foreign_keys = ON;

-- +migrate Down
DROP INDEX IF EXISTS idx_session_spotlight_session;
DROP TABLE IF EXISTS session_spotlight;
DROP INDEX IF EXISTS idx_session_gates_open;
DROP TABLE IF EXISTS session_gates;
DROP TABLE IF EXISTS daggerheart_adversaries;
DROP TABLE IF EXISTS daggerheart_countdowns;
DROP TABLE IF EXISTS daggerheart_snapshots;
DROP TABLE IF EXISTS daggerheart_character_states;
DROP TABLE IF EXISTS daggerheart_character_profiles;
DROP INDEX IF EXISTS idx_snapshots_seq;
DROP TABLE IF EXISTS snapshots;
DROP INDEX IF EXISTS idx_invites_campaign_recipient;
DROP INDEX IF EXISTS idx_invites_recipient_user;
DROP INDEX IF EXISTS idx_invites_participant;
DROP INDEX IF EXISTS idx_invites_campaign;
DROP TABLE IF EXISTS invites;
DROP TABLE IF EXISTS campaign_active_session;
DROP INDEX IF EXISTS idx_sessions_active;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS characters;
DROP INDEX IF EXISTS idx_participant_claims_participant;
DROP TABLE IF EXISTS participant_claims;
DROP INDEX IF EXISTS idx_participants_campaign_user;
DROP INDEX IF EXISTS idx_participants_user_id;
DROP TABLE IF EXISTS participants;
DROP TABLE IF EXISTS campaigns;
