package daggerheartprojection

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

// PutDaggerheartEnvironmentEntity persists a Daggerheart environment entity projection.
func (s *Store) PutDaggerheartEnvironmentEntity(ctx context.Context, environmentEntity projectionstore.DaggerheartEnvironmentEntity) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(environmentEntity.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(environmentEntity.EnvironmentEntityID) == "" {
		return fmt.Errorf("environment entity id is required")
	}
	if strings.TrimSpace(environmentEntity.EnvironmentID) == "" {
		return fmt.Errorf("environment id is required")
	}
	if strings.TrimSpace(environmentEntity.Name) == "" {
		return fmt.Errorf("environment name is required")
	}
	if strings.TrimSpace(environmentEntity.Type) == "" {
		return fmt.Errorf("environment type is required")
	}
	if strings.TrimSpace(environmentEntity.SessionID) == "" {
		return fmt.Errorf("session id is required")
	}

	return s.q.PutDaggerheartEnvironmentEntity(ctx, db.PutDaggerheartEnvironmentEntityParams{
		CampaignID:          environmentEntity.CampaignID,
		EnvironmentEntityID: environmentEntity.EnvironmentEntityID,
		EnvironmentID:       environmentEntity.EnvironmentID,
		Name:                environmentEntity.Name,
		Type:                environmentEntity.Type,
		Tier:                int64(environmentEntity.Tier),
		Difficulty:          int64(environmentEntity.Difficulty),
		SessionID:           environmentEntity.SessionID,
		SceneID:             environmentEntity.SceneID,
		Notes:               environmentEntity.Notes,
		CreatedAt:           toMillis(environmentEntity.CreatedAt),
		UpdatedAt:           toMillis(environmentEntity.UpdatedAt),
	})
}

// GetDaggerheartEnvironmentEntity retrieves a Daggerheart environment entity projection for a campaign.
func (s *Store) GetDaggerheartEnvironmentEntity(ctx context.Context, campaignID, environmentEntityID string) (projectionstore.DaggerheartEnvironmentEntity, error) {
	if err := ctx.Err(); err != nil {
		return projectionstore.DaggerheartEnvironmentEntity{}, err
	}
	if s == nil || s.sqlDB == nil {
		return projectionstore.DaggerheartEnvironmentEntity{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return projectionstore.DaggerheartEnvironmentEntity{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(environmentEntityID) == "" {
		return projectionstore.DaggerheartEnvironmentEntity{}, fmt.Errorf("environment entity id is required")
	}

	row, err := s.q.GetDaggerheartEnvironmentEntity(ctx, db.GetDaggerheartEnvironmentEntityParams{
		CampaignID:          campaignID,
		EnvironmentEntityID: environmentEntityID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return projectionstore.DaggerheartEnvironmentEntity{}, storage.ErrNotFound
		}
		return projectionstore.DaggerheartEnvironmentEntity{}, fmt.Errorf("get daggerheart environment entity: %w", err)
	}

	return projectionstore.DaggerheartEnvironmentEntity{
		CampaignID:          row.CampaignID,
		EnvironmentEntityID: row.EnvironmentEntityID,
		EnvironmentID:       row.EnvironmentID,
		Name:                row.Name,
		Type:                row.Type,
		Tier:                int(row.Tier),
		Difficulty:          int(row.Difficulty),
		SessionID:           row.SessionID,
		SceneID:             row.SceneID,
		Notes:               row.Notes,
		CreatedAt:           fromMillis(row.CreatedAt),
		UpdatedAt:           fromMillis(row.UpdatedAt),
	}, nil
}

// ListDaggerheartEnvironmentEntities retrieves environment entity projections for a campaign session.
func (s *Store) ListDaggerheartEnvironmentEntities(ctx context.Context, campaignID, sessionID, sceneID string) ([]projectionstore.DaggerheartEnvironmentEntity, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return nil, fmt.Errorf("session id is required")
	}

	var (
		rows []db.DaggerheartEnvironmentEntity
		err  error
	)
	if strings.TrimSpace(sceneID) == "" {
		rows, err = s.q.ListDaggerheartEnvironmentEntitiesBySession(ctx, db.ListDaggerheartEnvironmentEntitiesBySessionParams{
			CampaignID: campaignID,
			SessionID:  sessionID,
		})
	} else {
		rows, err = s.q.ListDaggerheartEnvironmentEntitiesByScene(ctx, db.ListDaggerheartEnvironmentEntitiesBySceneParams{
			CampaignID: campaignID,
			SessionID:  sessionID,
			SceneID:    sceneID,
		})
	}
	if err != nil {
		return nil, fmt.Errorf("list daggerheart environment entities: %w", err)
	}

	environmentEntities := make([]projectionstore.DaggerheartEnvironmentEntity, 0, len(rows))
	for _, row := range rows {
		environmentEntities = append(environmentEntities, projectionstore.DaggerheartEnvironmentEntity{
			CampaignID:          row.CampaignID,
			EnvironmentEntityID: row.EnvironmentEntityID,
			EnvironmentID:       row.EnvironmentID,
			Name:                row.Name,
			Type:                row.Type,
			Tier:                int(row.Tier),
			Difficulty:          int(row.Difficulty),
			SessionID:           row.SessionID,
			SceneID:             row.SceneID,
			Notes:               row.Notes,
			CreatedAt:           fromMillis(row.CreatedAt),
			UpdatedAt:           fromMillis(row.UpdatedAt),
		})
	}
	return environmentEntities, nil
}

// DeleteDaggerheartEnvironmentEntity removes an environment entity projection for a campaign.
func (s *Store) DeleteDaggerheartEnvironmentEntity(ctx context.Context, campaignID, environmentEntityID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(environmentEntityID) == "" {
		return fmt.Errorf("environment entity id is required")
	}

	return s.q.DeleteDaggerheartEnvironmentEntity(ctx, db.DeleteDaggerheartEnvironmentEntityParams{
		CampaignID:          campaignID,
		EnvironmentEntityID: environmentEntityID,
	})
}
