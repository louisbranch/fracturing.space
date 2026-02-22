-- +migrate Up

CREATE TABLE public_profiles (
    user_id TEXT NOT NULL PRIMARY KEY,
    name TEXT NOT NULL,
    avatar_set_id TEXT NOT NULL DEFAULT '',
    avatar_asset_id TEXT NOT NULL DEFAULT '',
    bio TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- +migrate Down

DROP TABLE IF EXISTS public_profiles;
