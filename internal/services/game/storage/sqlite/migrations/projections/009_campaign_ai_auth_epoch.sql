ALTER TABLE campaigns ADD COLUMN ai_auth_epoch INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_campaigns_ai_auth_epoch ON campaigns(ai_auth_epoch);
