CREATE TABLE IF NOT EXISTS discovery_entries (
  entry_id TEXT PRIMARY KEY,
  kind INTEGER NOT NULL,
  source_id TEXT NOT NULL,
  title TEXT NOT NULL,
  description TEXT NOT NULL,
  recommended_participants_min INTEGER NOT NULL CHECK (recommended_participants_min > 0),
  recommended_participants_max INTEGER NOT NULL CHECK (recommended_participants_max >= recommended_participants_min),
  difficulty_tier INTEGER NOT NULL,
  expected_duration_label TEXT NOT NULL,
  system INTEGER NOT NULL,
  gm_mode INTEGER NOT NULL DEFAULT 0,
  intent INTEGER NOT NULL DEFAULT 0,
  level INTEGER NOT NULL DEFAULT 0,
  character_count INTEGER NOT NULL DEFAULT 0,
  storyline TEXT NOT NULL DEFAULT '',
  tags TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

INSERT OR IGNORE INTO discovery_entries (
  entry_id,
  kind,
  source_id,
  title,
  description,
  recommended_participants_min,
  recommended_participants_max,
  difficulty_tier,
  expected_duration_label,
  system,
  gm_mode,
  intent,
  level,
  character_count,
  storyline,
  tags,
  created_at,
  updated_at
)
SELECT
  campaign_id,
  1,
  campaign_id,
  title,
  description,
  recommended_participants_min,
  recommended_participants_max,
  difficulty_tier,
  expected_duration_label,
  system,
  gm_mode,
  intent,
  level,
  character_count,
  storyline,
  tags,
  created_at,
  updated_at
FROM campaign_listings;

DROP TABLE IF EXISTS campaign_listings;
