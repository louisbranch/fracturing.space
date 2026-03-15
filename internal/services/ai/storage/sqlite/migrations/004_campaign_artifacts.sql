-- +migrate Up

CREATE TABLE ai_campaign_artifacts (
	campaign_id TEXT NOT NULL,
	path TEXT NOT NULL,
	content TEXT NOT NULL,
	read_only INTEGER NOT NULL DEFAULT 0,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL,
	PRIMARY KEY (campaign_id, path)
);

CREATE INDEX idx_ai_campaign_artifacts_campaign
	ON ai_campaign_artifacts(campaign_id, path);

-- +migrate Down
