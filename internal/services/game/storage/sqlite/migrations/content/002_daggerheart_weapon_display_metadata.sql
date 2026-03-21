-- +migrate Up

ALTER TABLE daggerheart_weapons ADD COLUMN display_order INTEGER NOT NULL DEFAULT 0;
ALTER TABLE daggerheart_weapons ADD COLUMN display_group TEXT NOT NULL DEFAULT 'physical';

-- +migrate Down

ALTER TABLE daggerheart_weapons DROP COLUMN display_group;
ALTER TABLE daggerheart_weapons DROP COLUMN display_order;
