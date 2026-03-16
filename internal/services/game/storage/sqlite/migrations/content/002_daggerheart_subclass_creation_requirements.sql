-- +migrate Up

ALTER TABLE daggerheart_subclasses ADD COLUMN creation_requirements_json TEXT NOT NULL DEFAULT '[]';

-- +migrate Down
