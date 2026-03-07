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

-- +migrate Down
DROP TABLE IF EXISTS scene_spotlight;
DROP INDEX IF EXISTS idx_scene_gates_open;
DROP TABLE IF EXISTS scene_gates;
DROP TABLE IF EXISTS scene_characters;
DROP INDEX IF EXISTS idx_scenes_active;
DROP INDEX IF EXISTS idx_scenes_session;
DROP TABLE IF EXISTS scenes;
