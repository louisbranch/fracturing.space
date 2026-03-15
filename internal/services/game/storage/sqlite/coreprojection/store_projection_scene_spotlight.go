package coreprojection

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// PutSceneSpotlight persists a scene spotlight projection.
func (s *Store) PutSceneSpotlight(ctx context.Context, spotlight storage.SceneSpotlight) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(spotlight.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(spotlight.SceneID) == "" {
		return fmt.Errorf("scene id is required")
	}
	spotlightType := strings.TrimSpace(string(spotlight.SpotlightType))
	if spotlightType == "" {
		return fmt.Errorf("spotlight type is required")
	}

	_, err := s.projectionQueryable().ExecContext(ctx,
		`INSERT INTO scene_spotlight (campaign_id, scene_id, spotlight_type, character_id, updated_at, updated_by_actor_type, updated_by_actor_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (campaign_id, scene_id) DO UPDATE SET
		   spotlight_type = excluded.spotlight_type,
		   character_id = excluded.character_id,
		   updated_at = excluded.updated_at,
		   updated_by_actor_type = excluded.updated_by_actor_type,
		   updated_by_actor_id = excluded.updated_by_actor_id`,
		spotlight.CampaignID, spotlight.SceneID, spotlightType, spotlight.CharacterID,
		sqliteutil.ToMillis(spotlight.UpdatedAt), spotlight.UpdatedByActorType, spotlight.UpdatedByActorID,
	)
	if err != nil {
		return fmt.Errorf("put scene spotlight: %w", err)
	}
	return nil
}

// GetSceneSpotlight retrieves a scene spotlight by scene id.
func (s *Store) GetSceneSpotlight(ctx context.Context, campaignID, sceneID string) (storage.SceneSpotlight, error) {
	if err := ctx.Err(); err != nil {
		return storage.SceneSpotlight{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.SceneSpotlight{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.SceneSpotlight{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sceneID) == "" {
		return storage.SceneSpotlight{}, fmt.Errorf("scene id is required")
	}

	row := s.projectionQueryable().QueryRowContext(ctx,
		`SELECT campaign_id, scene_id, spotlight_type, character_id, updated_at, updated_by_actor_type, updated_by_actor_id
		 FROM scene_spotlight WHERE campaign_id = ? AND scene_id = ?`,
		campaignID, sceneID,
	)

	var spotlight storage.SceneSpotlight
	var updatedAt int64
	var spotlightType string
	err := row.Scan(&spotlight.CampaignID, &spotlight.SceneID, &spotlightType, &spotlight.CharacterID,
		&updatedAt, &spotlight.UpdatedByActorType, &spotlight.UpdatedByActorID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.SceneSpotlight{}, storage.ErrNotFound
		}
		return storage.SceneSpotlight{}, fmt.Errorf("get scene spotlight: %w", err)
	}
	spotlight.SpotlightType = scene.SpotlightType(strings.ToLower(strings.TrimSpace(spotlightType)))
	spotlight.UpdatedAt = sqliteutil.FromMillis(updatedAt)
	return spotlight, nil
}

// ClearSceneSpotlight removes the current spotlight for a scene.
func (s *Store) ClearSceneSpotlight(ctx context.Context, campaignID, sceneID string) error {
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

	_, err := s.projectionQueryable().ExecContext(ctx,
		`DELETE FROM scene_spotlight WHERE campaign_id = ? AND scene_id = ?`,
		campaignID, sceneID,
	)
	if err != nil {
		return fmt.Errorf("clear scene spotlight: %w", err)
	}
	return nil
}
