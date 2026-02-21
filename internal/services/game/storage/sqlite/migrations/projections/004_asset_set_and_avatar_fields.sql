ALTER TABLE campaigns ADD COLUMN cover_set_id TEXT NOT NULL DEFAULT '';
ALTER TABLE participants ADD COLUMN avatar_set_id TEXT NOT NULL DEFAULT '';
ALTER TABLE participants ADD COLUMN avatar_asset_id TEXT NOT NULL DEFAULT '';
ALTER TABLE characters ADD COLUMN avatar_set_id TEXT NOT NULL DEFAULT '';
ALTER TABLE characters ADD COLUMN avatar_asset_id TEXT NOT NULL DEFAULT '';
