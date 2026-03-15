-- +migrate Up
CREATE TABLE IF NOT EXISTS transcript_messages (
    campaign_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    sequence_id INTEGER NOT NULL,
    message_id TEXT NOT NULL,
    sent_at_utc TEXT NOT NULL,
    participant_id TEXT NOT NULL,
    participant_name TEXT NOT NULL,
    body TEXT NOT NULL,
    client_message_id TEXT,
    PRIMARY KEY (campaign_id, session_id, sequence_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_transcript_message_id
    ON transcript_messages (campaign_id, session_id, message_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_transcript_client_message_id
    ON transcript_messages (campaign_id, session_id, client_message_id)
    WHERE client_message_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_transcript_sent_at
    ON transcript_messages (campaign_id, session_id, sent_at_utc, sequence_id);

-- +migrate Down
DROP INDEX IF EXISTS idx_transcript_sent_at;
DROP INDEX IF EXISTS idx_transcript_client_message_id;
DROP INDEX IF EXISTS idx_transcript_message_id;
DROP TABLE IF EXISTS transcript_messages;
