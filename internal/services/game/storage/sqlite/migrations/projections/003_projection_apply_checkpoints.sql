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

-- +migrate Down

DROP INDEX IF EXISTS idx_projection_apply_checkpoints_campaign;
DROP TABLE IF EXISTS projection_apply_checkpoints;
