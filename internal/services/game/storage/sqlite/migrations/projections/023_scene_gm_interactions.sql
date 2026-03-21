-- +migrate Up

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
