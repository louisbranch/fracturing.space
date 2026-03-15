-- +migrate Up

CREATE TABLE session_interactions (
    campaign_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    active_scene_id TEXT NOT NULL DEFAULT '',
    gm_authority_participant_id TEXT NOT NULL DEFAULT '',
    ooc_paused INTEGER NOT NULL DEFAULT 0,
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
    frame_text TEXT NOT NULL DEFAULT '',
    acting_character_ids_json BLOB,
    acting_participant_ids_json BLOB,
    posts_json BLOB,
    yielded_participant_ids_json BLOB,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (campaign_id, scene_id),
    FOREIGN KEY (campaign_id, scene_id) REFERENCES scenes(campaign_id, scene_id) ON DELETE CASCADE
);

CREATE INDEX idx_session_interactions_active_scene ON session_interactions(campaign_id, active_scene_id);
CREATE INDEX idx_scene_interactions_session ON scene_interactions(campaign_id, session_id);

-- +migrate Down

DROP INDEX IF EXISTS idx_scene_interactions_session;
DROP INDEX IF EXISTS idx_session_interactions_active_scene;
DROP TABLE IF EXISTS scene_interactions;
DROP TABLE IF EXISTS session_interactions;
