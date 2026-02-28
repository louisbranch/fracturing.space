-- +migrate Up

DROP INDEX IF EXISTS notification_deliveries_notification_channel_status_idx;
DROP INDEX IF EXISTS notification_deliveries_channel_status_next_attempt_idx;
DROP TABLE IF EXISTS notification_deliveries;
DROP INDEX IF EXISTS notifications_recipient_created_idx;
DROP INDEX IF EXISTS notifications_recipient_dedupe_unique_idx;
DROP TABLE IF EXISTS notifications;

CREATE TABLE notifications (
    id TEXT PRIMARY KEY,
    recipient_user_id TEXT NOT NULL,
    message_type TEXT NOT NULL,
    payload_json TEXT NOT NULL DEFAULT '{}',
    dedupe_key TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    read_at INTEGER
);

CREATE UNIQUE INDEX notifications_recipient_dedupe_unique_idx
    ON notifications(recipient_user_id, dedupe_key)
    WHERE dedupe_key <> '';

CREATE INDEX notifications_recipient_created_idx
    ON notifications(recipient_user_id, created_at DESC, id DESC);

CREATE TABLE notification_deliveries (
    notification_id TEXT NOT NULL,
    channel TEXT NOT NULL,
    status TEXT NOT NULL,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    next_attempt_at INTEGER NOT NULL,
    last_error TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    delivered_at INTEGER,
    PRIMARY KEY (notification_id, channel),
    FOREIGN KEY (notification_id) REFERENCES notifications(id) ON DELETE CASCADE
);

CREATE INDEX notification_deliveries_channel_status_next_attempt_idx
    ON notification_deliveries(channel, status, next_attempt_at, notification_id);

CREATE INDEX notification_deliveries_notification_channel_status_idx
    ON notification_deliveries(notification_id, channel, status);

-- +migrate Down
DROP INDEX IF EXISTS notification_deliveries_notification_channel_status_idx;
DROP INDEX IF EXISTS notification_deliveries_channel_status_next_attempt_idx;
DROP TABLE IF EXISTS notification_deliveries;
DROP INDEX IF EXISTS notifications_recipient_created_idx;
DROP INDEX IF EXISTS notifications_recipient_dedupe_unique_idx;
DROP TABLE IF EXISTS notifications;
