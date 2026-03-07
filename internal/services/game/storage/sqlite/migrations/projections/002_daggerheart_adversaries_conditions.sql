-- +migrate Up

ALTER TABLE daggerheart_adversaries
ADD COLUMN conditions_json TEXT NOT NULL DEFAULT '[]';

-- +migrate Down

-- SQLite does not support dropping columns in place.
