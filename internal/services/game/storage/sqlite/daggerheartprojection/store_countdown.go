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

// PutDaggerheartCountdown persists a Daggerheart countdown projection.
func (s *Store) PutDaggerheartCountdown(ctx context.Context, countdown projectionstore.DaggerheartCountdown) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(countdown.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(countdown.CountdownID) == "" {
		return fmt.Errorf("countdown id is required")
	}

	looping := int64(0)
	if countdown.Looping {
		looping = 1
	}

	return s.q.PutDaggerheartCountdown(ctx, db.PutDaggerheartCountdownParams{
		CampaignID:  countdown.CampaignID,
		CountdownID: countdown.CountdownID,
		Name:        countdown.Name,
		Kind:        countdown.Kind,
		Current:     int64(countdown.Current),
		Max:         int64(countdown.Max),
		Direction:   countdown.Direction,
		Looping:     looping,
	})
}

// GetDaggerheartCountdown retrieves a Daggerheart countdown projection for a campaign.
func (s *Store) GetDaggerheartCountdown(ctx context.Context, campaignID, countdownID string) (projectionstore.DaggerheartCountdown, error) {
	if err := ctx.Err(); err != nil {
		return projectionstore.DaggerheartCountdown{}, err
	}
	if s == nil || s.sqlDB == nil {
		return projectionstore.DaggerheartCountdown{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return projectionstore.DaggerheartCountdown{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(countdownID) == "" {
		return projectionstore.DaggerheartCountdown{}, fmt.Errorf("countdown id is required")
	}

	row, err := s.q.GetDaggerheartCountdown(ctx, db.GetDaggerheartCountdownParams{
		CampaignID:  campaignID,
		CountdownID: countdownID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return projectionstore.DaggerheartCountdown{}, storage.ErrNotFound
		}
		return projectionstore.DaggerheartCountdown{}, fmt.Errorf("get daggerheart countdown: %w", err)
	}

	return projectionstore.DaggerheartCountdown{
		CampaignID:  row.CampaignID,
		CountdownID: row.CountdownID,
		Name:        row.Name,
		Kind:        row.Kind,
		Current:     int(row.Current),
		Max:         int(row.Max),
		Direction:   row.Direction,
		Looping:     row.Looping != 0,
	}, nil
}

// ListDaggerheartCountdowns retrieves countdown projections for a campaign.
func (s *Store) ListDaggerheartCountdowns(ctx context.Context, campaignID string) ([]projectionstore.DaggerheartCountdown, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}

	rows, err := s.q.ListDaggerheartCountdowns(ctx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("list daggerheart countdowns: %w", err)
	}

	countdowns := make([]projectionstore.DaggerheartCountdown, 0, len(rows))
	for _, row := range rows {
		countdowns = append(countdowns, projectionstore.DaggerheartCountdown{
			CampaignID:  row.CampaignID,
			CountdownID: row.CountdownID,
			Name:        row.Name,
			Kind:        row.Kind,
			Current:     int(row.Current),
			Max:         int(row.Max),
			Direction:   row.Direction,
			Looping:     row.Looping != 0,
		})
	}

	return countdowns, nil
}

// DeleteDaggerheartCountdown removes a countdown projection for a campaign.
func (s *Store) DeleteDaggerheartCountdown(ctx context.Context, campaignID, countdownID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(countdownID) == "" {
		return fmt.Errorf("countdown id is required")
	}

	return s.q.DeleteDaggerheartCountdown(ctx, db.DeleteDaggerheartCountdownParams{
		CampaignID:  campaignID,
		CountdownID: countdownID,
	})
}
