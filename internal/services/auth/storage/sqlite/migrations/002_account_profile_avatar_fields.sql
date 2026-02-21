-- +migrate Up

ALTER TABLE account_profiles ADD COLUMN avatar_set_id TEXT NOT NULL DEFAULT '';
ALTER TABLE account_profiles ADD COLUMN avatar_asset_id TEXT NOT NULL DEFAULT '';

-- +migrate Down
