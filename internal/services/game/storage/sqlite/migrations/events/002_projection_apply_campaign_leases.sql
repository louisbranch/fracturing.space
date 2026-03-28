CREATE TABLE projection_apply_campaign_leases (
    campaign_id TEXT PRIMARY KEY,
    owner_id TEXT NOT NULL,
    lease_expires_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX idx_projection_apply_campaign_leases_expiry
    ON projection_apply_campaign_leases (lease_expires_at, campaign_id);
