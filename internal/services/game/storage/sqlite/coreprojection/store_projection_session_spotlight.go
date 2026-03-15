package coreprojection

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

// PutSessionSpotlight persists a session spotlight projection.
func (s *Store) PutSessionSpotlight(ctx context.Context, spotlight storage.SessionSpotlight) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(spotlight.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(spotlight.SessionID) == "" {
		return fmt.Errorf("session id is required")
	}
	spotlightType := strings.TrimSpace(string(spotlight.SpotlightType))
	if spotlightType == "" {
		return fmt.Errorf("spotlight type is required")
	}

	return s.q.PutSessionSpotlight(ctx, db.PutSessionSpotlightParams{
		CampaignID:         spotlight.CampaignID,
		SessionID:          spotlight.SessionID,
		SpotlightType:      spotlightType,
		CharacterID:        spotlight.CharacterID,
		UpdatedAt:          toMillis(spotlight.UpdatedAt),
		UpdatedByActorType: spotlight.UpdatedByActorType,
		UpdatedByActorID:   spotlight.UpdatedByActorID,
	})
}

// GetSessionSpotlight retrieves a session spotlight by session id.
func (s *Store) GetSessionSpotlight(ctx context.Context, campaignID, sessionID string) (storage.SessionSpotlight, error) {
	if err := ctx.Err(); err != nil {
		return storage.SessionSpotlight{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.SessionSpotlight{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.SessionSpotlight{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return storage.SessionSpotlight{}, fmt.Errorf("session id is required")
	}

	row, err := s.q.GetSessionSpotlight(ctx, db.GetSessionSpotlightParams{
		CampaignID: campaignID,
		SessionID:  sessionID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.SessionSpotlight{}, storage.ErrNotFound
		}
		return storage.SessionSpotlight{}, fmt.Errorf("get session spotlight: %w", err)
	}

	return dbSessionSpotlightToStorage(row), nil
}

// ClearSessionSpotlight removes the current spotlight for a session.
func (s *Store) ClearSessionSpotlight(ctx context.Context, campaignID, sessionID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return fmt.Errorf("session id is required")
	}

	return s.q.ClearSessionSpotlight(ctx, db.ClearSessionSpotlightParams{
		CampaignID: campaignID,
		SessionID:  sessionID,
	})
}
