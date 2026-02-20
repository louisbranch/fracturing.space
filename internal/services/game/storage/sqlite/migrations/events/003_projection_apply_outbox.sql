-- +migrate Up

CREATE TABLE IF NOT EXISTS projection_apply_outbox (
    campaign_id TEXT NOT NULL,
    seq INTEGER NOT NULL,
    event_type TEXT NOT NULL,
    status TEXT NOT NULL,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    next_attempt_at INTEGER NOT NULL,
    last_error TEXT NOT NULL DEFAULT '',
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (campaign_id, seq),
    FOREIGN KEY (campaign_id, seq) REFERENCES events(campaign_id, seq) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_projection_apply_outbox_status_next_attempt
    ON projection_apply_outbox (status, next_attempt_at, seq);

CREATE INDEX IF NOT EXISTS idx_projection_apply_outbox_campaign
    ON projection_apply_outbox (campaign_id, seq);

-- +migrate Down
DROP INDEX IF EXISTS idx_projection_apply_outbox_campaign;
DROP INDEX IF EXISTS idx_projection_apply_outbox_status_next_attempt;
DROP TABLE IF EXISTS projection_apply_outbox;
