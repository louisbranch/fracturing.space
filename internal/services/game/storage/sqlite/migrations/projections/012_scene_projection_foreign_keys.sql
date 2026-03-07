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

-- +migrate Down

-- Copy-forward migration only; down migration intentionally omitted.
