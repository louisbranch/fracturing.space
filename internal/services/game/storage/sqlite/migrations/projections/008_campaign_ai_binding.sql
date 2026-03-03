ALTER TABLE campaigns ADD COLUMN ai_agent_id TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_campaigns_ai_agent_id ON campaigns(ai_agent_id);
