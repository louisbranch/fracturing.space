CREATE TABLE IF NOT EXISTS capability_overrides (
    service    TEXT NOT NULL,
    capability TEXT NOT NULL,
    status     INTEGER NOT NULL DEFAULT 0,
    reason     INTEGER NOT NULL DEFAULT 0,
    detail     TEXT NOT NULL DEFAULT '',
    set_at     INTEGER NOT NULL,
    PRIMARY KEY (service, capability)
);
