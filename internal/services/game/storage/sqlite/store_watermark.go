package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// GetProjectionWatermark returns the watermark for a campaign.
// Returns storage.ErrNotFound if no watermark exists.
func (s *Store) GetProjectionWatermark(ctx context.Context, campaignID string) (storage.ProjectionWatermark, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return storage.ProjectionWatermark{}, fmt.Errorf("campaign id is required")
	}
	row := s.sqlDB.QueryRowContext(ctx,
		`SELECT campaign_id, applied_seq, expected_next_seq, updated_at FROM projection_watermarks WHERE campaign_id = ?`,
		campaignID,
	)
	var wm storage.ProjectionWatermark
	var updatedAtMillis int64
	err := row.Scan(&wm.CampaignID, &wm.AppliedSeq, &wm.ExpectedNextSeq, &updatedAtMillis)
	if errors.Is(err, sql.ErrNoRows) {
		return storage.ProjectionWatermark{}, storage.ErrNotFound
	}
	if err != nil {
		return storage.ProjectionWatermark{}, fmt.Errorf("get projection watermark: %w", err)
	}
	wm.UpdatedAt = fromMillis(updatedAtMillis)
	return wm, nil
}

// SaveProjectionWatermark upserts the watermark for a campaign.
func (s *Store) SaveProjectionWatermark(ctx context.Context, wm storage.ProjectionWatermark) error {
	wm.CampaignID = strings.TrimSpace(wm.CampaignID)
	if wm.CampaignID == "" {
		return fmt.Errorf("campaign id is required")
	}
	_, err := s.sqlDB.ExecContext(ctx,
		`INSERT INTO projection_watermarks (campaign_id, applied_seq, expected_next_seq, updated_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT (campaign_id) DO UPDATE SET
		     applied_seq = excluded.applied_seq,
		     expected_next_seq = excluded.expected_next_seq,
		     updated_at = excluded.updated_at`,
		wm.CampaignID,
		int64(wm.AppliedSeq),
		int64(wm.ExpectedNextSeq),
		toMillis(wm.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("save projection watermark: %w", err)
	}
	return nil
}

// ListProjectionWatermarks returns all watermarks ordered by campaign id.
func (s *Store) ListProjectionWatermarks(ctx context.Context) ([]storage.ProjectionWatermark, error) {
	rows, err := s.sqlDB.QueryContext(ctx,
		`SELECT campaign_id, applied_seq, expected_next_seq, updated_at FROM projection_watermarks ORDER BY campaign_id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list projection watermarks: %w", err)
	}
	defer rows.Close()
	var watermarks []storage.ProjectionWatermark
	for rows.Next() {
		var wm storage.ProjectionWatermark
		var updatedAtMillis int64
		if err := rows.Scan(&wm.CampaignID, &wm.AppliedSeq, &wm.ExpectedNextSeq, &updatedAtMillis); err != nil {
			return nil, fmt.Errorf("scan projection watermark: %w", err)
		}
		wm.UpdatedAt = fromMillis(updatedAtMillis)
		watermarks = append(watermarks, wm)
	}
	return watermarks, rows.Err()
}
