-- +migrate Up

CREATE TABLE IF NOT EXISTS cache_entries (
    cache_key TEXT PRIMARY KEY,
    scope TEXT NOT NULL,
    campaign_id TEXT NOT NULL DEFAULT '',
    user_id TEXT NOT NULL DEFAULT '',
    payload_json BLOB NOT NULL,
    source_seq INTEGER NOT NULL DEFAULT 0,
    stale INTEGER NOT NULL DEFAULT 0,
    checked_at INTEGER NOT NULL DEFAULT 0,
    refreshed_at INTEGER NOT NULL DEFAULT 0,
    expires_at INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_cache_entries_campaign_scope
    ON cache_entries(campaign_id, scope);

CREATE INDEX IF NOT EXISTS idx_cache_entries_user_scope
    ON cache_entries(user_id, scope);

CREATE TABLE IF NOT EXISTS campaign_event_cursors (
    campaign_id TEXT PRIMARY KEY,
    latest_seq INTEGER NOT NULL DEFAULT 0,
    checked_at INTEGER NOT NULL DEFAULT 0
);

-- +migrate Down

DROP TABLE IF EXISTS campaign_event_cursors;
DROP INDEX IF EXISTS idx_cache_entries_user_scope;
DROP INDEX IF EXISTS idx_cache_entries_campaign_scope;
DROP TABLE IF EXISTS cache_entries;
