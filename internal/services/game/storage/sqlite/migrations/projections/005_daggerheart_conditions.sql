-- +migrate Up

DROP TABLE IF EXISTS daggerheart_character_states;
CREATE TABLE daggerheart_character_states (
    campaign_id TEXT NOT NULL,
    character_id TEXT NOT NULL,
    hp INTEGER NOT NULL DEFAULT 6,
    hope INTEGER NOT NULL DEFAULT 2,
    stress INTEGER NOT NULL DEFAULT 0,
    armor INTEGER NOT NULL DEFAULT 0,
    conditions_json TEXT NOT NULL DEFAULT '[]',
    PRIMARY KEY (campaign_id, character_id),
    FOREIGN KEY (campaign_id, character_id)
        REFERENCES characters(campaign_id, id) ON DELETE CASCADE
);

-- +migrate Down
