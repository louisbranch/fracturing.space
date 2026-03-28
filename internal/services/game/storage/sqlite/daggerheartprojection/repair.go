package daggerheartprojection

import (
	"context"
	"database/sql"
	"fmt"
)

// RepairLegacyConditionStateEncoding rewrites legacy string-array condition JSON
// into the structured condition-state shape expected by steady-state readers.
//
// This runs once at projections-store startup so the backend does not need to
// carry string-array compatibility in every read path indefinitely.
func RepairLegacyConditionStateEncoding(ctx context.Context, sqlDB *sql.DB) error {
	if sqlDB == nil {
		return fmt.Errorf("sql db is required")
	}

	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin legacy condition repair: %w", err)
	}

	if err := repairLegacyCharacterStateConditions(ctx, tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := repairLegacyAdversaryConditions(ctx, tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit legacy condition repair: %w", err)
	}
	return nil
}

func repairLegacyCharacterStateConditions(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `
SELECT campaign_id, character_id, conditions_json
FROM daggerheart_character_states
WHERE conditions_json != ''
  AND conditions_json != '[]'
`)
	if err != nil {
		return fmt.Errorf("query legacy daggerheart character state conditions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var campaignID, characterID, raw string
		if err := rows.Scan(&campaignID, &characterID, &raw); err != nil {
			return fmt.Errorf("scan legacy daggerheart character state conditions: %w", err)
		}
		normalized, changed, err := normalizeLegacyProjectionConditionStatesJSON(raw)
		if err != nil {
			return fmt.Errorf("normalize legacy daggerheart character state conditions for %s/%s: %w", campaignID, characterID, err)
		}
		if !changed {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE daggerheart_character_states
SET conditions_json = ?
WHERE campaign_id = ? AND character_id = ?
`, normalized, campaignID, characterID); err != nil {
			return fmt.Errorf("update legacy daggerheart character state conditions for %s/%s: %w", campaignID, characterID, err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate legacy daggerheart character state conditions: %w", err)
	}
	return nil
}

func repairLegacyAdversaryConditions(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `
SELECT campaign_id, adversary_id, conditions_json
FROM daggerheart_adversaries
WHERE conditions_json != ''
  AND conditions_json != '[]'
`)
	if err != nil {
		return fmt.Errorf("query legacy daggerheart adversary conditions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var campaignID, adversaryID, raw string
		if err := rows.Scan(&campaignID, &adversaryID, &raw); err != nil {
			return fmt.Errorf("scan legacy daggerheart adversary conditions: %w", err)
		}
		normalized, changed, err := normalizeLegacyProjectionConditionStatesJSON(raw)
		if err != nil {
			return fmt.Errorf("normalize legacy daggerheart adversary conditions for %s/%s: %w", campaignID, adversaryID, err)
		}
		if !changed {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE daggerheart_adversaries
SET conditions_json = ?
WHERE campaign_id = ? AND adversary_id = ?
`, normalized, campaignID, adversaryID); err != nil {
			return fmt.Errorf("update legacy daggerheart adversary conditions for %s/%s: %w", campaignID, adversaryID, err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate legacy daggerheart adversary conditions: %w", err)
	}
	return nil
}
