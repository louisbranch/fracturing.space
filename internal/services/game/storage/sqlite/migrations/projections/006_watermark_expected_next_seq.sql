-- +migrate Up

DROP TABLE IF EXISTS projection_watermarks;

CREATE TABLE projection_watermarks (
    campaign_id TEXT PRIMARY KEY,
    applied_seq INTEGER NOT NULL DEFAULT 0,
    expected_next_seq INTEGER NOT NULL DEFAULT 0,
    updated_at INTEGER NOT NULL
);

-- +migrate Down

DROP TABLE IF EXISTS projection_watermarks;

CREATE TABLE projection_watermarks (
    campaign_id TEXT PRIMARY KEY,
    applied_seq INTEGER NOT NULL DEFAULT 0,
    updated_at INTEGER NOT NULL
);
