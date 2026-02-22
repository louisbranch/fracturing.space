package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

// PutSnapshot stores a snapshot.
func (s *Store) PutSnapshot(ctx context.Context, snapshot storage.Snapshot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(snapshot.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(snapshot.SessionID) == "" {
		return fmt.Errorf("session id is required")
	}

	return s.q.PutSnapshot(ctx, db.PutSnapshotParams{
		CampaignID:          snapshot.CampaignID,
		SessionID:           snapshot.SessionID,
		EventSeq:            int64(snapshot.EventSeq),
		CharacterStatesJson: snapshot.CharacterStatesJSON,
		GmStateJson:         snapshot.GMStateJSON,
		SystemStateJson:     snapshot.SystemStateJSON,
		CreatedAt:           toMillis(snapshot.CreatedAt),
	})
}

// GetSnapshot retrieves a snapshot by campaign and session ID.
func (s *Store) GetSnapshot(ctx context.Context, campaignID, sessionID string) (storage.Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return storage.Snapshot{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.Snapshot{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.Snapshot{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return storage.Snapshot{}, fmt.Errorf("session id is required")
	}

	row, err := s.q.GetSnapshot(ctx, db.GetSnapshotParams{
		CampaignID: campaignID,
		SessionID:  sessionID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.Snapshot{}, storage.ErrNotFound
		}
		return storage.Snapshot{}, fmt.Errorf("get snapshot: %w", err)
	}

	return dbSnapshotToDomain(row)
}

// GetLatestSnapshot retrieves the most recent snapshot for a campaign.
func (s *Store) GetLatestSnapshot(ctx context.Context, campaignID string) (storage.Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return storage.Snapshot{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.Snapshot{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.Snapshot{}, fmt.Errorf("campaign id is required")
	}

	row, err := s.q.GetLatestSnapshot(ctx, campaignID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.Snapshot{}, storage.ErrNotFound
		}
		return storage.Snapshot{}, fmt.Errorf("get latest snapshot: %w", err)
	}

	return dbSnapshotToDomain(row)
}

// ListSnapshots returns snapshots ordered by event sequence descending.
func (s *Store) ListSnapshots(ctx context.Context, campaignID string, limit int) ([]storage.Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than zero")
	}

	rows, err := s.q.ListSnapshots(ctx, db.ListSnapshotsParams{
		CampaignID: campaignID,
		Limit:      int64(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("list snapshots: %w", err)
	}

	snapshots := make([]storage.Snapshot, 0, len(rows))
	for _, row := range rows {
		snapshot, err := dbSnapshotToDomain(row)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}

// GetCampaignForkMetadata retrieves fork metadata for a campaign.
func (s *Store) GetCampaignForkMetadata(ctx context.Context, campaignID string) (storage.ForkMetadata, error) {
	if err := ctx.Err(); err != nil {
		return storage.ForkMetadata{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.ForkMetadata{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.ForkMetadata{}, fmt.Errorf("campaign id is required")
	}

	row, err := s.q.GetCampaignForkMetadata(ctx, campaignID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ForkMetadata{}, storage.ErrNotFound
		}
		return storage.ForkMetadata{}, fmt.Errorf("get campaign fork metadata: %w", err)
	}

	metadata := storage.ForkMetadata{}
	if row.ParentCampaignID.Valid {
		metadata.ParentCampaignID = row.ParentCampaignID.String
	}
	if row.ForkEventSeq.Valid {
		metadata.ForkEventSeq = uint64(row.ForkEventSeq.Int64)
	}
	if row.OriginCampaignID.Valid {
		metadata.OriginCampaignID = row.OriginCampaignID.String
	}

	return metadata, nil
}

// SetCampaignForkMetadata sets fork metadata for a campaign.
func (s *Store) SetCampaignForkMetadata(ctx context.Context, campaignID string, metadata storage.ForkMetadata) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}

	var parentCampaignID sql.NullString
	if metadata.ParentCampaignID != "" {
		parentCampaignID = sql.NullString{String: metadata.ParentCampaignID, Valid: true}
	}

	var forkEventSeq sql.NullInt64
	if metadata.ForkEventSeq > 0 {
		forkEventSeq = sql.NullInt64{Int64: int64(metadata.ForkEventSeq), Valid: true}
	}

	var originCampaignID sql.NullString
	if metadata.OriginCampaignID != "" {
		originCampaignID = sql.NullString{String: metadata.OriginCampaignID, Valid: true}
	}

	return s.q.SetCampaignForkMetadata(ctx, db.SetCampaignForkMetadataParams{
		ParentCampaignID: parentCampaignID,
		ForkEventSeq:     forkEventSeq,
		OriginCampaignID: originCampaignID,
		ID:               campaignID,
	})
}

func dbSnapshotToDomain(row db.Snapshot) (storage.Snapshot, error) {
	return storage.Snapshot{
		CampaignID:          row.CampaignID,
		SessionID:           row.SessionID,
		EventSeq:            uint64(row.EventSeq),
		CharacterStatesJSON: row.CharacterStatesJson,
		GMStateJSON:         row.GmStateJson,
		SystemStateJSON:     row.SystemStateJson,
		CreatedAt:           fromMillis(row.CreatedAt),
	}, nil
}
