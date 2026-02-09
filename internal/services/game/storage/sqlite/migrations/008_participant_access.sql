-- +migrate Up

DROP TABLE IF EXISTS invites;
DROP TABLE IF EXISTS participants;

CREATE TABLE participants (
    campaign_id TEXT NOT NULL,
    id TEXT NOT NULL,
    user_id TEXT NOT NULL DEFAULT '',
    display_name TEXT NOT NULL,
    role TEXT NOT NULL,
    controller TEXT NOT NULL,
    campaign_access TEXT NOT NULL DEFAULT 'MEMBER',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (campaign_id, id),
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

CREATE INDEX idx_participants_user_id ON participants(user_id);
CREATE UNIQUE INDEX idx_participants_campaign_user
    ON participants(campaign_id, user_id)
    WHERE user_id != '';

CREATE TABLE invites (
    id TEXT PRIMARY KEY,
    campaign_id TEXT NOT NULL,
    participant_id TEXT NOT NULL,
    status TEXT NOT NULL,
    created_by_participant_id TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE,
    FOREIGN KEY (campaign_id, participant_id) REFERENCES participants(campaign_id, id) ON DELETE CASCADE
);

CREATE INDEX idx_invites_campaign ON invites(campaign_id);
CREATE INDEX idx_invites_participant ON invites(participant_id);

-- +migrate Down
DROP INDEX IF EXISTS idx_invites_participant;
DROP INDEX IF EXISTS idx_invites_campaign;
DROP TABLE IF EXISTS invites;
DROP INDEX IF EXISTS idx_participants_campaign_user;
DROP INDEX IF EXISTS idx_participants_user_id;
DROP TABLE IF EXISTS participants;
