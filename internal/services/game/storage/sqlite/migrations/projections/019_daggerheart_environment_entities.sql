-- +migrate Up
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

-- +migrate Down
DROP TABLE IF EXISTS daggerheart_environment_entities;
