-- Baseline schema for fresh alpha databases.

CREATE TABLE discovery_entries (
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
, preview_hook TEXT NOT NULL DEFAULT '', preview_playstyle_label TEXT NOT NULL DEFAULT '', preview_character_name TEXT NOT NULL DEFAULT '', preview_character_summary TEXT NOT NULL DEFAULT '', campaign_theme TEXT NOT NULL DEFAULT '');
