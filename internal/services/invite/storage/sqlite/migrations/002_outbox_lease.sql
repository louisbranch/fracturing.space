-- Add lease/ack columns so the worker can poll the invite outbox.
ALTER TABLE outbox ADD COLUMN status TEXT NOT NULL DEFAULT 'pending';
ALTER TABLE outbox ADD COLUMN attempt_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE outbox ADD COLUMN next_attempt_at TEXT NOT NULL DEFAULT '';
ALTER TABLE outbox ADD COLUMN lease_owner TEXT NOT NULL DEFAULT '';
ALTER TABLE outbox ADD COLUMN lease_expires_at TEXT NOT NULL DEFAULT '';
ALTER TABLE outbox ADD COLUMN last_error TEXT NOT NULL DEFAULT '';
ALTER TABLE outbox ADD COLUMN processed_at TEXT NOT NULL DEFAULT '';
ALTER TABLE outbox ADD COLUMN updated_at TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_outbox_lease ON outbox(status, next_attempt_at, lease_expires_at, id);
