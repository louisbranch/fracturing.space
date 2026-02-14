-- +migrate Up

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

-- +migrate Down

DROP TABLE IF EXISTS daggerheart_countdowns;
