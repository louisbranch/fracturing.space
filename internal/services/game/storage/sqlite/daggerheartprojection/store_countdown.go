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

	return s.q.PutDaggerheartCountdown(ctx, db.PutDaggerheartCountdownParams{
		CampaignID:        countdown.CampaignID,
		CountdownID:       countdown.CountdownID,
		SessionID:         countdown.SessionID,
		SceneID:           countdown.SceneID,
		Name:              countdown.Name,
		Tone:              countdown.Tone,
		AdvancementPolicy: countdown.AdvancementPolicy,
		StartingValue:     int64(countdown.StartingValue),
		RemainingValue:    int64(countdown.RemainingValue),
		LoopBehavior:      countdown.LoopBehavior,
		Status:            countdown.Status,
		LinkedCountdownID: countdown.LinkedCountdownID,
		StartingRollMin:   int64(countdown.StartingRollMin),
		StartingRollMax:   int64(countdown.StartingRollMax),
		StartingRollValue: int64(countdown.StartingRollValue),
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
		CampaignID:        row.CampaignID,
		CountdownID:       row.CountdownID,
		SessionID:         row.SessionID,
		SceneID:           row.SceneID,
		Name:              row.Name,
		Tone:              row.Tone,
		AdvancementPolicy: row.AdvancementPolicy,
		StartingValue:     int(row.StartingValue),
		RemainingValue:    int(row.RemainingValue),
		LoopBehavior:      row.LoopBehavior,
		Status:            row.Status,
		LinkedCountdownID: row.LinkedCountdownID,
		StartingRollMin:   int(row.StartingRollMin),
		StartingRollMax:   int(row.StartingRollMax),
		StartingRollValue: int(row.StartingRollValue),
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
			CampaignID:        row.CampaignID,
			CountdownID:       row.CountdownID,
			SessionID:         row.SessionID,
			SceneID:           row.SceneID,
			Name:              row.Name,
			Tone:              row.Tone,
			AdvancementPolicy: row.AdvancementPolicy,
			StartingValue:     int(row.StartingValue),
			RemainingValue:    int(row.RemainingValue),
			LoopBehavior:      row.LoopBehavior,
			Status:            row.Status,
			LinkedCountdownID: row.LinkedCountdownID,
			StartingRollMin:   int(row.StartingRollMin),
			StartingRollMax:   int(row.StartingRollMax),
			StartingRollValue: int(row.StartingRollValue),
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
