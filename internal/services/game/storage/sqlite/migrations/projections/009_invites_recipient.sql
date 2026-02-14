-- +migrate Up

DROP INDEX IF EXISTS idx_invites_campaign_recipient;
DROP INDEX IF EXISTS idx_invites_recipient_user;
DROP INDEX IF EXISTS idx_invites_participant;
DROP INDEX IF EXISTS idx_invites_campaign;
DROP TABLE IF EXISTS invites;

CREATE TABLE invites (
    id TEXT PRIMARY KEY,
    campaign_id TEXT NOT NULL,
    participant_id TEXT NOT NULL,
    recipient_user_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    created_by_participant_id TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE,
    FOREIGN KEY (campaign_id, participant_id) REFERENCES participants(campaign_id, id) ON DELETE CASCADE
);

CREATE INDEX idx_invites_campaign ON invites(campaign_id);
CREATE INDEX idx_invites_participant ON invites(participant_id);
CREATE INDEX idx_invites_recipient_user ON invites(recipient_user_id);
CREATE INDEX idx_invites_campaign_recipient ON invites(campaign_id, recipient_user_id);

-- +migrate Down

DROP INDEX IF EXISTS idx_invites_campaign_recipient;
DROP INDEX IF EXISTS idx_invites_recipient_user;
DROP INDEX IF EXISTS idx_invites_participant;
DROP INDEX IF EXISTS idx_invites_campaign;
DROP TABLE IF EXISTS invites;
