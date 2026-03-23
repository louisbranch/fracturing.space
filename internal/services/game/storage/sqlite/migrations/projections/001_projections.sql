-- Baseline schema for fresh alpha databases.

-- +migrate Up

-- Campaign layer
CREATE TABLE IF NOT EXISTS campaigns (
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

CREATE TABLE IF NOT EXISTS participants (
    campaign_id TEXT NOT NULL,
    id TEXT NOT NULL,
    user_id TEXT NOT NULL DEFAULT '',
    display_name TEXT NOT NULL,
    role TEXT NOT NULL,
    controller TEXT NOT NULL,
    campaign_access TEXT NOT NULL DEFAULT 'MEMBER',
    pronouns TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (campaign_id, id),
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_participants_user_id ON participants(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_participants_campaign_user
    ON participants(campaign_id, user_id)
    WHERE user_id != '';

CREATE TABLE IF NOT EXISTS participant_claims (
    campaign_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    participant_id TEXT NOT NULL,
    claimed_at INTEGER NOT NULL,
    PRIMARY KEY (campaign_id, user_id),
    UNIQUE (campaign_id, participant_id),
    FOREIGN KEY (campaign_id, participant_id) REFERENCES participants(campaign_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_participant_claims_participant ON participant_claims(participant_id);

CREATE TABLE IF NOT EXISTS characters (
    campaign_id TEXT NOT NULL,
    id TEXT NOT NULL,
    controller_participant_id TEXT,
    name TEXT NOT NULL,
    kind TEXT NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    pronouns TEXT NOT NULL DEFAULT '',
    aliases_json TEXT NOT NULL DEFAULT '[]',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (campaign_id, id),
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

-- Session layer
CREATE TABLE IF NOT EXISTS sessions (
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

CREATE INDEX IF NOT EXISTS idx_sessions_active ON sessions(campaign_id)
    WHERE status = 'ACTIVE';

CREATE TABLE IF NOT EXISTS campaign_active_session (
    campaign_id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

-- Invitations
CREATE TABLE IF NOT EXISTS invites (
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

CREATE INDEX IF NOT EXISTS idx_invites_campaign ON invites(campaign_id);
CREATE INDEX IF NOT EXISTS idx_invites_participant ON invites(participant_id);
CREATE INDEX IF NOT EXISTS idx_invites_recipient_user ON invites(recipient_user_id);
CREATE INDEX IF NOT EXISTS idx_invites_campaign_recipient ON invites(campaign_id, recipient_user_id);

-- Snapshots
CREATE TABLE IF NOT EXISTS snapshots (
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

CREATE INDEX IF NOT EXISTS idx_snapshots_seq ON snapshots(campaign_id, event_seq DESC);

-- Daggerheart projection tables
CREATE TABLE IF NOT EXISTS daggerheart_character_profiles (
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
    class_id TEXT NOT NULL DEFAULT '',
    subclass_id TEXT NOT NULL DEFAULT '',
    subclass_tracks_json TEXT NOT NULL DEFAULT '[]',
    subclass_creation_requirements_json TEXT NOT NULL DEFAULT '[]',
    heritage_json TEXT NOT NULL DEFAULT '',
    companion_sheet_json TEXT NOT NULL DEFAULT '',
    equipped_armor_id TEXT NOT NULL DEFAULT '',
    spellcast_roll_bonus INTEGER NOT NULL DEFAULT 0,
    traits_assigned INTEGER NOT NULL DEFAULT 0,
    background TEXT NOT NULL DEFAULT '',
    details_recorded INTEGER NOT NULL DEFAULT 0,
    starting_weapon_ids_json TEXT NOT NULL DEFAULT '[]',
    starting_armor_id TEXT NOT NULL DEFAULT '',
    starting_potion_item_id TEXT NOT NULL DEFAULT '',
    domain_card_ids_json TEXT NOT NULL DEFAULT '[]',
    connections TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (campaign_id, character_id),
    FOREIGN KEY (campaign_id, character_id)
        REFERENCES characters(campaign_id, id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS daggerheart_character_states (
    campaign_id TEXT NOT NULL,
    character_id TEXT NOT NULL,
    hp INTEGER NOT NULL DEFAULT 6,
    hope INTEGER NOT NULL DEFAULT 2,
    hope_max INTEGER NOT NULL DEFAULT 6,
    stress INTEGER NOT NULL DEFAULT 0,
    armor INTEGER NOT NULL DEFAULT 0,
    conditions_json TEXT NOT NULL DEFAULT '[]',
    temporary_armor_json TEXT NOT NULL DEFAULT '[]',
    life_state TEXT NOT NULL DEFAULT 'alive',
    class_state_json TEXT NOT NULL DEFAULT '{}',
    subclass_state_json TEXT NOT NULL DEFAULT '{}',
    companion_state_json TEXT NOT NULL DEFAULT '{}',
    impenetrable_used_this_short_rest INTEGER NOT NULL DEFAULT 0,
    stat_modifiers_json TEXT NOT NULL DEFAULT '[]',
    PRIMARY KEY (campaign_id, character_id),
    FOREIGN KEY (campaign_id, character_id)
        REFERENCES characters(campaign_id, id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS daggerheart_snapshots (
    campaign_id TEXT PRIMARY KEY,
    gm_fear INTEGER NOT NULL DEFAULT 0,
    consecutive_short_rests INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS daggerheart_countdowns (
    campaign_id TEXT NOT NULL,
    countdown_id TEXT NOT NULL,
    session_id TEXT NOT NULL DEFAULT '',
    scene_id TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL,
    tone TEXT NOT NULL,
    advancement_policy TEXT NOT NULL,
    starting_value INTEGER NOT NULL,
    remaining_value INTEGER NOT NULL,
    loop_behavior TEXT NOT NULL,
    status TEXT NOT NULL,
    linked_countdown_id TEXT NOT NULL DEFAULT '',
    starting_roll_min INTEGER NOT NULL DEFAULT 0,
    starting_roll_max INTEGER NOT NULL DEFAULT 0,
    starting_roll_value INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (campaign_id, countdown_id)
);

CREATE TABLE IF NOT EXISTS daggerheart_adversaries (
    campaign_id TEXT NOT NULL,
    adversary_id TEXT NOT NULL,
    adversary_entry_id TEXT NOT NULL,
    name TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT '',
    session_id TEXT NOT NULL,
    scene_id TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    hp INTEGER NOT NULL DEFAULT 6,
    hp_max INTEGER NOT NULL DEFAULT 6,
    stress INTEGER NOT NULL DEFAULT 0,
    stress_max INTEGER NOT NULL DEFAULT 6,
    evasion INTEGER NOT NULL DEFAULT 10,
    major_threshold INTEGER NOT NULL DEFAULT 8,
    severe_threshold INTEGER NOT NULL DEFAULT 12,
    armor INTEGER NOT NULL DEFAULT 0,
    conditions_json TEXT NOT NULL DEFAULT '[]',
    feature_state_json TEXT NOT NULL DEFAULT '[]',
    pending_experience_json TEXT NOT NULL DEFAULT '',
    spotlight_gate_id TEXT NOT NULL DEFAULT '',
    spotlight_count INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (campaign_id, adversary_id),
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS daggerheart_environment_entities (
    campaign_id TEXT NOT NULL,
    environment_entity_id TEXT NOT NULL,
    environment_id TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT '',
    tier INTEGER NOT NULL DEFAULT 0,
    difficulty INTEGER NOT NULL DEFAULT 0,
    session_id TEXT NOT NULL,
    scene_id TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (campaign_id, environment_entity_id),
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

-- Session governance
CREATE TABLE IF NOT EXISTS session_gates (
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
    response_authority TEXT NOT NULL DEFAULT '',
    metadata_extra_json BLOB,
    resolution_decision TEXT NOT NULL DEFAULT '',
    resolution_extra_json BLOB,
    PRIMARY KEY (campaign_id, session_id, gate_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_session_gates_open
    ON session_gates(campaign_id, session_id)
    WHERE status = 'open';

CREATE TABLE IF NOT EXISTS session_gate_eligible_participants (
    campaign_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    gate_id TEXT NOT NULL,
    position INTEGER NOT NULL,
    participant_id TEXT NOT NULL,
    PRIMARY KEY (campaign_id, session_id, gate_id, participant_id)
);

CREATE INDEX IF NOT EXISTS idx_session_gate_eligible_participants_gate
    ON session_gate_eligible_participants(campaign_id, session_id, gate_id, position);

CREATE TABLE IF NOT EXISTS session_gate_options (
    campaign_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    gate_id TEXT NOT NULL,
    position INTEGER NOT NULL,
    option_value TEXT NOT NULL,
    PRIMARY KEY (campaign_id, session_id, gate_id, option_value)
);

CREATE INDEX IF NOT EXISTS idx_session_gate_options_gate
    ON session_gate_options(campaign_id, session_id, gate_id, position);

CREATE TABLE IF NOT EXISTS session_gate_responses (
    campaign_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    gate_id TEXT NOT NULL,
    participant_id TEXT NOT NULL,
    decision TEXT NOT NULL DEFAULT '',
    response_json BLOB,
    recorded_at INTEGER,
    actor_type TEXT NOT NULL DEFAULT '',
    actor_id TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (campaign_id, session_id, gate_id, participant_id)
);

CREATE INDEX IF NOT EXISTS idx_session_gate_responses_gate
    ON session_gate_responses(campaign_id, session_id, gate_id, participant_id);

CREATE TABLE IF NOT EXISTS session_spotlight (
    campaign_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    spotlight_type TEXT NOT NULL,
    character_id TEXT NOT NULL DEFAULT '',
    updated_at INTEGER NOT NULL,
    updated_by_actor_type TEXT NOT NULL,
    updated_by_actor_id TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (campaign_id, session_id)
);

CREATE INDEX IF NOT EXISTS idx_session_spotlight_session ON session_spotlight(campaign_id, session_id);


ALTER TABLE campaigns ADD COLUMN cover_asset_id TEXT NOT NULL DEFAULT '';

-- +migrate Up

CREATE TABLE IF NOT EXISTS projection_apply_checkpoints (
    campaign_id TEXT NOT NULL,
    seq INTEGER NOT NULL,
    event_type TEXT NOT NULL,
    applied_at INTEGER NOT NULL,
    PRIMARY KEY (campaign_id, seq)
);

CREATE INDEX IF NOT EXISTS idx_projection_apply_checkpoints_campaign
    ON projection_apply_checkpoints (campaign_id, seq);


ALTER TABLE campaigns ADD COLUMN cover_set_id TEXT NOT NULL DEFAULT '';
ALTER TABLE participants ADD COLUMN avatar_set_id TEXT NOT NULL DEFAULT '';
ALTER TABLE participants ADD COLUMN avatar_asset_id TEXT NOT NULL DEFAULT '';
ALTER TABLE characters ADD COLUMN avatar_set_id TEXT NOT NULL DEFAULT '';
ALTER TABLE characters ADD COLUMN avatar_asset_id TEXT NOT NULL DEFAULT '';

-- +migrate Up

CREATE TABLE IF NOT EXISTS projection_watermarks (
    campaign_id TEXT PRIMARY KEY,
    applied_seq INTEGER NOT NULL DEFAULT 0,
    updated_at INTEGER NOT NULL
);


-- +migrate Up

ALTER TABLE projection_watermarks
ADD COLUMN expected_next_seq INTEGER NOT NULL DEFAULT 0;


ALTER TABLE characters ADD COLUMN owner_participant_id TEXT NOT NULL DEFAULT '';

UPDATE characters
SET owner_participant_id = COALESCE(controller_participant_id, '')
WHERE owner_participant_id = '';

ALTER TABLE campaigns ADD COLUMN ai_agent_id TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_campaigns_ai_agent_id ON campaigns(ai_agent_id);

ALTER TABLE campaigns ADD COLUMN ai_auth_epoch INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_campaigns_ai_auth_epoch ON campaigns(ai_auth_epoch);

-- +migrate Up
ALTER TABLE daggerheart_character_profiles ADD COLUMN description TEXT NOT NULL DEFAULT '';

-- +migrate Up

DROP TABLE IF EXISTS daggerheart_character_profiles;

CREATE TABLE IF NOT EXISTS daggerheart_character_profiles (
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
    class_id TEXT NOT NULL DEFAULT '',
    subclass_id TEXT NOT NULL DEFAULT '',
    subclass_tracks_json TEXT NOT NULL DEFAULT '[]',
    subclass_creation_requirements_json TEXT NOT NULL DEFAULT '[]',
    heritage_json TEXT NOT NULL DEFAULT '{}',
    companion_sheet_json TEXT NOT NULL DEFAULT '',
    equipped_armor_id TEXT NOT NULL DEFAULT '',
    spellcast_roll_bonus INTEGER NOT NULL DEFAULT 0,
    traits_assigned INTEGER NOT NULL DEFAULT 0,
    details_recorded INTEGER NOT NULL DEFAULT 0,
    starting_weapon_ids_json TEXT NOT NULL DEFAULT '[]',
    starting_armor_id TEXT NOT NULL DEFAULT '',
    starting_potion_item_id TEXT NOT NULL DEFAULT '',
    background TEXT NOT NULL DEFAULT '',
    domain_card_ids_json TEXT NOT NULL DEFAULT '[]',
    connections TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (campaign_id, character_id),
    FOREIGN KEY (campaign_id, character_id)
        REFERENCES characters(campaign_id, id) ON DELETE CASCADE
);


-- +migrate Up
CREATE TABLE scenes (
    campaign_id TEXT NOT NULL,
    scene_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    active INTEGER NOT NULL DEFAULT 1,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    ended_at INTEGER,
    PRIMARY KEY (campaign_id, scene_id)
);

CREATE INDEX idx_scenes_session ON scenes(campaign_id, session_id);
CREATE INDEX idx_scenes_active ON scenes(campaign_id, active);

CREATE TABLE scene_characters (
    campaign_id TEXT NOT NULL,
    scene_id TEXT NOT NULL,
    character_id TEXT NOT NULL,
    added_at INTEGER NOT NULL,
    PRIMARY KEY (campaign_id, scene_id, character_id)
);

CREATE TABLE scene_gates (
    campaign_id TEXT NOT NULL,
    scene_id TEXT NOT NULL,
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
    PRIMARY KEY (campaign_id, scene_id, gate_id)
);

CREATE INDEX idx_scene_gates_open ON scene_gates(campaign_id, scene_id, status);

CREATE TABLE scene_spotlight (
    campaign_id TEXT NOT NULL,
    scene_id TEXT NOT NULL,
    spotlight_type TEXT NOT NULL,
    character_id TEXT NOT NULL DEFAULT '',
    updated_at INTEGER NOT NULL,
    updated_by_actor_type TEXT NOT NULL,
    updated_by_actor_id TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (campaign_id, scene_id)
);


-- +migrate Up

CREATE TABLE scenes_v2 (
    campaign_id TEXT NOT NULL,
    scene_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    active INTEGER NOT NULL DEFAULT 1,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    ended_at INTEGER,
    PRIMARY KEY (campaign_id, scene_id),
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

INSERT INTO scenes_v2 (campaign_id, scene_id, session_id, name, description, active, created_at, updated_at, ended_at)
SELECT campaign_id, scene_id, session_id, name, description, active, created_at, updated_at, ended_at
FROM scenes;

CREATE TABLE scene_characters_v2 (
    campaign_id TEXT NOT NULL,
    scene_id TEXT NOT NULL,
    character_id TEXT NOT NULL,
    added_at INTEGER NOT NULL,
    PRIMARY KEY (campaign_id, scene_id, character_id),
    FOREIGN KEY (campaign_id, scene_id) REFERENCES scenes_v2(campaign_id, scene_id) ON DELETE CASCADE
);

INSERT INTO scene_characters_v2 (campaign_id, scene_id, character_id, added_at)
SELECT campaign_id, scene_id, character_id, added_at
FROM scene_characters;

CREATE TABLE scene_gates_v2 (
    campaign_id TEXT NOT NULL,
    scene_id TEXT NOT NULL,
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
    PRIMARY KEY (campaign_id, scene_id, gate_id),
    FOREIGN KEY (campaign_id, scene_id) REFERENCES scenes_v2(campaign_id, scene_id) ON DELETE CASCADE
);

INSERT INTO scene_gates_v2 (campaign_id, scene_id, gate_id, gate_type, status, reason, created_at, created_by_actor_type, created_by_actor_id, resolved_at, resolved_by_actor_type, resolved_by_actor_id, metadata_json, resolution_json)
SELECT campaign_id, scene_id, gate_id, gate_type, status, reason, created_at, created_by_actor_type, created_by_actor_id, resolved_at, resolved_by_actor_type, resolved_by_actor_id, metadata_json, resolution_json
FROM scene_gates;

CREATE TABLE scene_spotlight_v2 (
    campaign_id TEXT NOT NULL,
    scene_id TEXT NOT NULL,
    spotlight_type TEXT NOT NULL,
    character_id TEXT NOT NULL DEFAULT '',
    updated_at INTEGER NOT NULL,
    updated_by_actor_type TEXT NOT NULL,
    updated_by_actor_id TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (campaign_id, scene_id),
    FOREIGN KEY (campaign_id, scene_id) REFERENCES scenes_v2(campaign_id, scene_id) ON DELETE CASCADE
);

INSERT INTO scene_spotlight_v2 (campaign_id, scene_id, spotlight_type, character_id, updated_at, updated_by_actor_type, updated_by_actor_id)
SELECT campaign_id, scene_id, spotlight_type, character_id, updated_at, updated_by_actor_type, updated_by_actor_id
FROM scene_spotlight;

DROP INDEX IF EXISTS idx_scene_gates_open;
DROP INDEX IF EXISTS idx_scenes_active;
DROP INDEX IF EXISTS idx_scenes_session;

DROP TABLE scene_spotlight;
DROP TABLE scene_gates;
DROP TABLE scene_characters;
DROP TABLE scenes;

ALTER TABLE scenes_v2 RENAME TO scenes;
ALTER TABLE scene_characters_v2 RENAME TO scene_characters;
ALTER TABLE scene_gates_v2 RENAME TO scene_gates;
ALTER TABLE scene_spotlight_v2 RENAME TO scene_spotlight;

CREATE INDEX idx_scenes_session ON scenes(campaign_id, session_id);
CREATE INDEX idx_scenes_active ON scenes(campaign_id, active);
CREATE INDEX idx_scene_gates_open ON scene_gates(campaign_id, scene_id, status);


-- +migrate Up

CREATE TABLE session_interactions (
    campaign_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    active_scene_id TEXT NOT NULL DEFAULT '',
    gm_authority_participant_id TEXT NOT NULL DEFAULT '',
    ooc_opened INTEGER NOT NULL DEFAULT 0,
    ooc_requested_by_participant_id TEXT NOT NULL DEFAULT '',
    ooc_reason TEXT NOT NULL DEFAULT '',
    ooc_interrupted_scene_id TEXT NOT NULL DEFAULT '',
    ooc_interrupted_phase_id TEXT NOT NULL DEFAULT '',
    ooc_interrupted_phase_status TEXT NOT NULL DEFAULT '',
    ooc_resolution_pending INTEGER NOT NULL DEFAULT 0,
    ooc_posts_json BLOB,
    ready_to_resume_json BLOB,
    ai_turn_status TEXT NOT NULL DEFAULT 'idle',
    ai_turn_token TEXT NOT NULL DEFAULT '',
    ai_turn_owner_participant_id TEXT NOT NULL DEFAULT '',
    ai_turn_source_event_type TEXT NOT NULL DEFAULT '',
    ai_turn_source_scene_id TEXT NOT NULL DEFAULT '',
    ai_turn_source_phase_id TEXT NOT NULL DEFAULT '',
    ai_turn_last_error TEXT NOT NULL DEFAULT '',
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (campaign_id, session_id),
    FOREIGN KEY (campaign_id, session_id) REFERENCES sessions(campaign_id, id) ON DELETE CASCADE
);

CREATE TABLE scene_interactions (
    campaign_id TEXT NOT NULL,
    scene_id TEXT NOT NULL,
    session_id TEXT NOT NULL DEFAULT '',
    phase_open INTEGER NOT NULL DEFAULT 0,
    phase_id TEXT NOT NULL DEFAULT '',
    phase_status TEXT NOT NULL DEFAULT '',
    acting_character_ids_json BLOB,
    acting_participant_ids_json BLOB,
    slots_json BLOB,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (campaign_id, scene_id),
    FOREIGN KEY (campaign_id, scene_id) REFERENCES scenes(campaign_id, scene_id) ON DELETE CASCADE
);

CREATE INDEX idx_session_interactions_active_scene ON session_interactions(campaign_id, active_scene_id);
CREATE INDEX idx_scene_interactions_session ON scene_interactions(campaign_id, session_id);

CREATE TABLE scene_gm_interactions (
    campaign_id TEXT NOT NULL,
    scene_id TEXT NOT NULL,
    session_id TEXT NOT NULL DEFAULT '',
    interaction_id TEXT NOT NULL,
    phase_id TEXT NOT NULL DEFAULT '',
    participant_id TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL DEFAULT '',
    character_ids_json BLOB,
    illustration_json BLOB,
    beats_json BLOB,
    created_at INTEGER NOT NULL,
    PRIMARY KEY (campaign_id, interaction_id),
    FOREIGN KEY (campaign_id, scene_id) REFERENCES scenes(campaign_id, scene_id) ON DELETE CASCADE
);

CREATE INDEX idx_scene_gm_interactions_scene_created ON scene_gm_interactions(campaign_id, scene_id, created_at DESC);
