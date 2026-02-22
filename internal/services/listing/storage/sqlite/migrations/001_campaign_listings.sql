CREATE TABLE IF NOT EXISTS campaign_listings (
  campaign_id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  description TEXT NOT NULL,
  recommended_participants_min INTEGER NOT NULL CHECK (recommended_participants_min > 0),
  recommended_participants_max INTEGER NOT NULL CHECK (recommended_participants_max >= recommended_participants_min),
  difficulty_tier INTEGER NOT NULL,
  expected_duration_label TEXT NOT NULL,
  system INTEGER NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
