-- +migrate Up

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
    created_at INTEGER NOT NULL,
    last_activity_at INTEGER NOT NULL,
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

CREATE TABLE invites (
    id TEXT PRIMARY KEY,
    campaign_id TEXT NOT NULL,
    participant_id TEXT NOT NULL,
    status TEXT NOT NULL,
    created_by_participant_id TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE,
    FOREIGN KEY (campaign_id, participant_id) REFERENCES participants(campaign_id, id) ON DELETE CASCADE
);

CREATE INDEX idx_invites_campaign ON invites(campaign_id);
CREATE INDEX idx_invites_participant ON invites(participant_id);

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
DROP INDEX IF EXISTS idx_invites_participant;
DROP INDEX IF EXISTS idx_invites_campaign;
DROP TABLE IF EXISTS invites;
DROP TABLE IF EXISTS campaign_active_session;
DROP INDEX IF EXISTS idx_sessions_active;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS characters;
DROP INDEX IF EXISTS idx_participants_campaign_user;
DROP INDEX IF EXISTS idx_participants_user_id;
DROP TABLE IF EXISTS participants;
DROP TABLE IF EXISTS campaigns;
