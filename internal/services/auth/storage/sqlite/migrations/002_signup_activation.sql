-- +migrate Up

ALTER TABLE registration_sessions ADD COLUMN credential_id TEXT NOT NULL DEFAULT '';
ALTER TABLE registration_sessions ADD COLUMN credential_json TEXT NOT NULL DEFAULT '';

-- +migrate Down
