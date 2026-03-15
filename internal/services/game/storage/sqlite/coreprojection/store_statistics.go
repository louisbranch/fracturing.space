package coreprojection

import (
	"context"
	"fmt"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// GetGameStatistics returns aggregate counts across the projection data set.
func (s *Store) GetGameStatistics(ctx context.Context, since *time.Time) (storage.GameStatistics, error) {
	if err := ctx.Err(); err != nil {
		return storage.GameStatistics{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.GameStatistics{}, fmt.Errorf("storage is not configured")
	}

	sinceValue := sqliteutil.ToNullMillis(since)
	row, err := s.q.GetGameStatistics(ctx, sinceValue)
	if err != nil {
		return storage.GameStatistics{}, fmt.Errorf("get game statistics: %w", err)
	}

	return storage.GameStatistics{
		CampaignCount:    row.CampaignCount,
		SessionCount:     row.SessionCount,
		CharacterCount:   row.CharacterCount,
		ParticipantCount: row.ParticipantCount,
	}, nil
}
