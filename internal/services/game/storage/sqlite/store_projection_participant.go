package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

// Participant methods.

// PutParticipant persists a participant record.
func (s *Store) PutParticipant(ctx context.Context, p storage.ParticipantRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(p.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(p.ID) == "" {
		return fmt.Errorf("participant id is required")
	}

	if err := s.q.PutParticipant(ctx, db.PutParticipantParams{
		CampaignID:     p.CampaignID,
		ID:             p.ID,
		UserID:         p.UserID,
		DisplayName:    p.Name,
		Role:           participantRoleToString(p.Role),
		Controller:     participantControllerToString(p.Controller),
		CampaignAccess: participantAccessToString(p.CampaignAccess),
		AvatarSetID:    p.AvatarSetID,
		AvatarAssetID:  p.AvatarAssetID,
		CreatedAt:      toMillis(p.CreatedAt),
		UpdatedAt:      toMillis(p.UpdatedAt),
	}); err != nil {
		if isParticipantUserConflict(err) {
			return apperrors.WithMetadata(
				apperrors.CodeParticipantUserAlreadyClaimed,
				"participant user already claimed",
				map[string]string{
					"CampaignID": p.CampaignID,
					"UserID":     p.UserID,
				},
			)
		}
		return err
	}
	return nil
}

// DeleteParticipant deletes a participant record by IDs.
func (s *Store) DeleteParticipant(ctx context.Context, campaignID, participantID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(participantID) == "" {
		return fmt.Errorf("participant id is required")
	}

	return s.q.DeleteParticipant(ctx, db.DeleteParticipantParams{
		CampaignID: campaignID,
		ID:         participantID,
	})
}

// GetParticipant fetches a participant record by IDs.
func (s *Store) GetParticipant(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
	if err := ctx.Err(); err != nil {
		return storage.ParticipantRecord{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.ParticipantRecord{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.ParticipantRecord{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(participantID) == "" {
		return storage.ParticipantRecord{}, fmt.Errorf("participant id is required")
	}

	row, err := s.q.GetParticipant(ctx, db.GetParticipantParams{
		CampaignID: campaignID,
		ID:         participantID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ParticipantRecord{}, storage.ErrNotFound
		}
		return storage.ParticipantRecord{}, fmt.Errorf("get participant: %w", err)
	}

	return dbGetParticipantRowToDomain(row)
}

// ListParticipantsByCampaign returns all participants for a campaign.
func (s *Store) ListParticipantsByCampaign(ctx context.Context, campaignID string) ([]storage.ParticipantRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}

	rows, err := s.q.ListParticipantsByCampaign(ctx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("list participants: %w", err)
	}

	participants := make([]storage.ParticipantRecord, 0, len(rows))
	for _, row := range rows {
		p, err := dbListParticipantsByCampaignRowToDomain(row)
		if err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}

	return participants, nil
}

// ListCampaignIDsByUser returns campaigns where the given user participates.
func (s *Store) ListCampaignIDsByUser(ctx context.Context, userID string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, fmt.Errorf("user id is required")
	}

	rows, err := s.q.ListCampaignIDsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list campaign IDs by user: %w", err)
	}

	campaignIDs := make([]string, 0, len(rows))
	for _, campaignID := range rows {
		campaignID = strings.TrimSpace(campaignID)
		if campaignID != "" {
			campaignIDs = append(campaignIDs, campaignID)
		}
	}
	return campaignIDs, nil
}

// ListCampaignIDsByParticipant returns campaign IDs where the given participant id exists.
func (s *Store) ListCampaignIDsByParticipant(ctx context.Context, participantID string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	participantID = strings.TrimSpace(participantID)
	if participantID == "" {
		return nil, fmt.Errorf("participant id is required")
	}

	rows, err := s.q.ListCampaignIDsByParticipant(ctx, participantID)
	if err != nil {
		return nil, fmt.Errorf("list campaign IDs by participant: %w", err)
	}

	campaignIDs := make([]string, 0, len(rows))
	for _, campaignID := range rows {
		campaignID = strings.TrimSpace(campaignID)
		if campaignID != "" {
			campaignIDs = append(campaignIDs, campaignID)
		}
	}
	return campaignIDs, nil
}

// ListParticipants returns a page of participant records.
func (s *Store) ListParticipants(ctx context.Context, campaignID string, pageSize int, pageToken string) (storage.ParticipantPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.ParticipantPage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.ParticipantPage{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.ParticipantPage{}, fmt.Errorf("campaign id is required")
	}
	if pageSize <= 0 {
		return storage.ParticipantPage{}, fmt.Errorf("page size must be greater than zero")
	}

	page := storage.ParticipantPage{
		Participants: make([]storage.ParticipantRecord, 0, pageSize),
	}

	if pageToken == "" {
		rows, err := s.q.ListParticipantsByCampaignPagedFirst(ctx, db.ListParticipantsByCampaignPagedFirstParams{
			CampaignID: campaignID,
			Limit:      int64(pageSize + 1),
		})
		if err != nil {
			return storage.ParticipantPage{}, fmt.Errorf("list participants: %w", err)
		}
		for i, row := range rows {
			if i >= pageSize {
				page.NextPageToken = rows[pageSize-1].ID
				break
			}
			p, err := dbListParticipantsByCampaignPagedFirstRowToDomain(row)
			if err != nil {
				return storage.ParticipantPage{}, err
			}
			page.Participants = append(page.Participants, p)
		}
		return page, nil
	}

	rows, err := s.q.ListParticipantsByCampaignPaged(ctx, db.ListParticipantsByCampaignPagedParams{
		CampaignID: campaignID,
		ID:         pageToken,
		Limit:      int64(pageSize + 1),
	})
	if err != nil {
		return storage.ParticipantPage{}, fmt.Errorf("list participants: %w", err)
	}
	for i, row := range rows {
		if i >= pageSize {
			page.NextPageToken = rows[pageSize-1].ID
			break
		}
		p, err := dbListParticipantsByCampaignPagedRowToDomain(row)
		if err != nil {
			return storage.ParticipantPage{}, err
		}
		page.Participants = append(page.Participants, p)
	}

	return page, nil
}

// CountParticipants returns the number of participants for a campaign.
func (s *Store) CountParticipants(ctx context.Context, campaignID string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if s == nil || s.sqlDB == nil {
		return 0, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return 0, fmt.Errorf("campaign id is required")
	}
	var count int64
	err := s.sqlDB.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM participants WHERE campaign_id = ?", campaignID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count participants: %w", err)
	}
	return int(count), nil
}
