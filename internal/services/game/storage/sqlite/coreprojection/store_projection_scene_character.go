package coreprojection

import (
	"context"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// PutSceneCharacter adds a character to a scene.
func (s *Store) PutSceneCharacter(ctx context.Context, rec storage.SceneCharacterRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(rec.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(rec.SceneID) == "" {
		return fmt.Errorf("scene id is required")
	}
	if strings.TrimSpace(rec.CharacterID) == "" {
		return fmt.Errorf("character id is required")
	}

	_, err := s.projectionQueryable().ExecContext(ctx,
		`INSERT INTO scene_characters (campaign_id, scene_id, character_id, added_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT (campaign_id, scene_id, character_id) DO NOTHING`,
		rec.CampaignID, rec.SceneID, rec.CharacterID, sqliteutil.ToMillis(rec.AddedAt),
	)
	if err != nil {
		return fmt.Errorf("put scene character: %w", err)
	}
	return nil
}

// DeleteSceneCharacter removes a character from a scene.
func (s *Store) DeleteSceneCharacter(ctx context.Context, campaignID, sceneID, characterID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sceneID) == "" {
		return fmt.Errorf("scene id is required")
	}
	if strings.TrimSpace(characterID) == "" {
		return fmt.Errorf("character id is required")
	}

	_, err := s.projectionQueryable().ExecContext(ctx,
		`DELETE FROM scene_characters WHERE campaign_id = ? AND scene_id = ? AND character_id = ?`,
		campaignID, sceneID, characterID,
	)
	if err != nil {
		return fmt.Errorf("delete scene character: %w", err)
	}
	return nil
}

// ListSceneCharacters returns all characters in a scene.
func (s *Store) ListSceneCharacters(ctx context.Context, campaignID, sceneID string) ([]storage.SceneCharacterRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sceneID) == "" {
		return nil, fmt.Errorf("scene id is required")
	}

	rows, err := s.projectionQueryable().QueryContext(ctx,
		`SELECT campaign_id, scene_id, character_id, added_at
		 FROM scene_characters WHERE campaign_id = ? AND scene_id = ?
		 ORDER BY character_id ASC`,
		campaignID, sceneID,
	)
	if err != nil {
		return nil, fmt.Errorf("list scene characters: %w", err)
	}
	defer rows.Close()

	var result []storage.SceneCharacterRecord
	for rows.Next() {
		var rec storage.SceneCharacterRecord
		var addedAt int64
		if err := rows.Scan(&rec.CampaignID, &rec.SceneID, &rec.CharacterID, &addedAt); err != nil {
			return nil, fmt.Errorf("scan scene character: %w", err)
		}
		rec.AddedAt = sqliteutil.FromMillis(addedAt)
		result = append(result, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list scene characters rows: %w", err)
	}
	return result, nil
}
