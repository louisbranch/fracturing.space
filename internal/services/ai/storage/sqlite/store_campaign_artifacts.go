package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/ai/campaignartifact"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// PutCampaignArtifact persists one campaign-scoped GM artifact snapshot.
func (s *Store) PutCampaignArtifact(ctx context.Context, record campaignartifact.Artifact) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if record.CampaignID == "" {
		return fmt.Errorf("campaign id is required")
	}
	if record.Path == "" {
		return fmt.Errorf("artifact path is required")
	}

	_, err := s.sqlDB.ExecContext(ctx, `
INSERT INTO ai_campaign_artifacts (
	campaign_id, path, content, read_only, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(campaign_id, path) DO UPDATE SET
	content = excluded.content,
	read_only = excluded.read_only,
	updated_at = excluded.updated_at
`,
		record.CampaignID,
		record.Path,
		record.Content,
		record.ReadOnly,
		sqliteutil.ToMillis(record.CreatedAt),
		sqliteutil.ToMillis(record.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("put campaign artifact: %w", err)
	}
	return nil
}

// GetCampaignArtifact fetches one campaign artifact by path.
func (s *Store) GetCampaignArtifact(ctx context.Context, campaignID string, path string) (campaignartifact.Artifact, error) {
	if err := ctx.Err(); err != nil {
		return campaignartifact.Artifact{}, err
	}
	if s == nil || s.sqlDB == nil {
		return campaignartifact.Artifact{}, fmt.Errorf("storage is not configured")
	}
	if campaignID == "" {
		return campaignartifact.Artifact{}, fmt.Errorf("campaign id is required")
	}
	if path == "" {
		return campaignartifact.Artifact{}, fmt.Errorf("artifact path is required")
	}

	row := s.sqlDB.QueryRowContext(ctx, `
SELECT campaign_id, path, content, read_only, created_at, updated_at
FROM ai_campaign_artifacts
WHERE campaign_id = ? AND path = ?
`, campaignID, path)

	var (
		record    campaignartifact.Artifact
		createdAt int64
		updatedAt int64
	)
	if err := row.Scan(
		&record.CampaignID,
		&record.Path,
		&record.Content,
		&record.ReadOnly,
		&createdAt,
		&updatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return campaignartifact.Artifact{}, storage.ErrNotFound
		}
		return campaignartifact.Artifact{}, fmt.Errorf("get campaign artifact: %w", err)
	}
	record.CreatedAt = sqliteutil.FromMillis(createdAt)
	record.UpdatedAt = sqliteutil.FromMillis(updatedAt)
	return record, nil
}

// ListCampaignArtifacts returns all persisted artifacts for one campaign.
func (s *Store) ListCampaignArtifacts(ctx context.Context, campaignID string) ([]campaignartifact.Artifact, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if campaignID == "" {
		return nil, fmt.Errorf("campaign id is required")
	}

	rows, err := s.sqlDB.QueryContext(ctx, `
SELECT campaign_id, path, content, read_only, created_at, updated_at
FROM ai_campaign_artifacts
WHERE campaign_id = ?
ORDER BY path
`, campaignID)
	if err != nil {
		return nil, fmt.Errorf("list campaign artifacts: %w", err)
	}
	defer rows.Close()

	records := make([]campaignartifact.Artifact, 0, 4)
	for rows.Next() {
		var (
			record    campaignartifact.Artifact
			createdAt int64
			updatedAt int64
		)
		if err := rows.Scan(
			&record.CampaignID,
			&record.Path,
			&record.Content,
			&record.ReadOnly,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan campaign artifact row: %w", err)
		}
		record.CreatedAt = sqliteutil.FromMillis(createdAt)
		record.UpdatedAt = sqliteutil.FromMillis(updatedAt)
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate campaign artifact rows: %w", err)
	}
	return records, nil
}
