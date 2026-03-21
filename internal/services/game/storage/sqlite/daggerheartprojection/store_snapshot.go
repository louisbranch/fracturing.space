package daggerheartprojection

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

// PutDaggerheartSnapshot persists a Daggerheart snapshot projection.
func (s *Store) PutDaggerheartSnapshot(ctx context.Context, snap projectionstore.DaggerheartSnapshot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(snap.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}

	return s.q.PutDaggerheartSnapshot(ctx, db.PutDaggerheartSnapshotParams{
		CampaignID:            snap.CampaignID,
		GmFear:                int64(snap.GMFear),
		ConsecutiveShortRests: int64(snap.ConsecutiveShortRests),
	})
}

// GetDaggerheartSnapshot retrieves the Daggerheart snapshot projection for a campaign.
func (s *Store) GetDaggerheartSnapshot(ctx context.Context, campaignID string) (projectionstore.DaggerheartSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return projectionstore.DaggerheartSnapshot{}, err
	}
	if s == nil || s.sqlDB == nil {
		return projectionstore.DaggerheartSnapshot{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return projectionstore.DaggerheartSnapshot{}, fmt.Errorf("campaign id is required")
	}

	row, err := s.q.GetDaggerheartSnapshot(ctx, campaignID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Return zero-value for not found (consistent with GetGmFear behavior)
			return projectionstore.DaggerheartSnapshot{CampaignID: campaignID, GMFear: 0, ConsecutiveShortRests: 0}, nil
		}
		return projectionstore.DaggerheartSnapshot{}, fmt.Errorf("get daggerheart snapshot: %w", err)
	}

	return projectionstore.DaggerheartSnapshot{
		CampaignID:            row.CampaignID,
		GMFear:                int(row.GmFear),
		ConsecutiveShortRests: int(row.ConsecutiveShortRests),
	}, nil
}
