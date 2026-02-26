ALTER TABLE characters ADD COLUMN owner_participant_id TEXT NOT NULL DEFAULT '';

UPDATE characters
SET owner_participant_id = COALESCE(controller_participant_id, '')
WHERE owner_participant_id = '';
