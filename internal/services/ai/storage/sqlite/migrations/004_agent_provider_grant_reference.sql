-- +migrate Up

ALTER TABLE ai_agents ADD COLUMN provider_grant_id TEXT NOT NULL DEFAULT '';

CREATE INDEX ai_agents_owner_provider_grant_id_idx ON ai_agents(owner_user_id, provider_grant_id);

-- +migrate Down
DROP INDEX IF EXISTS ai_agents_owner_provider_grant_id_idx;
