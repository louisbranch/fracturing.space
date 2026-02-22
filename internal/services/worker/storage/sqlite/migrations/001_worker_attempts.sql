-- +migrate Up

CREATE TABLE worker_attempts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    consumer TEXT NOT NULL,
    outcome TEXT NOT NULL,
    attempt_count INTEGER NOT NULL,
    last_error TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL
);

CREATE INDEX worker_attempts_created_idx
ON worker_attempts(created_at DESC, id DESC);

-- +migrate Down
DROP INDEX IF EXISTS worker_attempts_created_idx;
DROP TABLE IF EXISTS worker_attempts;
