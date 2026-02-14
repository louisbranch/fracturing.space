-- +migrate Up

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

-- +migrate Down

DROP TABLE IF EXISTS daggerheart_adversaries;
