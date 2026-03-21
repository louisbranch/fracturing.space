CREATE TABLE IF NOT EXISTS invites (
    id TEXT PRIMARY KEY,
    campaign_id TEXT NOT NULL,
    participant_id TEXT NOT NULL,
    recipient_user_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending',
    created_by_participant_id TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_invites_campaign_id ON invites(campaign_id);
CREATE INDEX IF NOT EXISTS idx_invites_recipient_status ON invites(recipient_user_id, status);

CREATE TABLE IF NOT EXISTS outbox (
    id TEXT PRIMARY KEY,
    event_type TEXT NOT NULL,
    payload_json TEXT NOT NULL,
    dedupe_key TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_outbox_dedupe ON outbox(dedupe_key) WHERE dedupe_key != '';
