package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// PutParticipantClaim stores a user claim for a participant seat.
func (s *Store) PutParticipantClaim(ctx context.Context, campaignID, userID, participantID string, claimedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(userID) == "" {
		return fmt.Errorf("user id is required")
	}
	if strings.TrimSpace(participantID) == "" {
		return fmt.Errorf("participant id is required")
	}
	if claimedAt.IsZero() {
		claimedAt = time.Now().UTC()
	}

	_, err := s.sqlDB.ExecContext(
		ctx,
		"INSERT INTO participant_claims (campaign_id, user_id, participant_id, claimed_at) VALUES (?, ?, ?, ?)",
		campaignID,
		userID,
		participantID,
		toMillis(claimedAt),
	)
	if err == nil {
		return nil
	}
	if !isParticipantClaimConflict(err) {
		return fmt.Errorf("put participant claim: %w", err)
	}

	claim, claimErr := s.GetParticipantClaim(ctx, campaignID, userID)
	if claimErr != nil {
		return fmt.Errorf("get participant claim: %w", claimErr)
	}
	if claim.ParticipantID == participantID {
		return nil
	}
	return apperrors.WithMetadata(
		apperrors.CodeParticipantUserAlreadyClaimed,
		"participant user already claimed",
		map[string]string{
			"CampaignID": campaignID,
			"UserID":     userID,
		},
	)
}

// GetParticipantClaim returns the claim for a user in a campaign.
func (s *Store) GetParticipantClaim(ctx context.Context, campaignID, userID string) (storage.ParticipantClaim, error) {
	if err := ctx.Err(); err != nil {
		return storage.ParticipantClaim{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.ParticipantClaim{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.ParticipantClaim{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(userID) == "" {
		return storage.ParticipantClaim{}, fmt.Errorf("user id is required")
	}

	row := s.sqlDB.QueryRowContext(
		ctx,
		"SELECT participant_id, claimed_at FROM participant_claims WHERE campaign_id = ? AND user_id = ?",
		campaignID,
		userID,
	)
	var participantID string
	var claimedAt int64
	if err := row.Scan(&participantID, &claimedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ParticipantClaim{}, storage.ErrNotFound
		}
		return storage.ParticipantClaim{}, fmt.Errorf("get participant claim: %w", err)
	}

	return storage.ParticipantClaim{
		CampaignID:    campaignID,
		UserID:        userID,
		ParticipantID: participantID,
		ClaimedAt:     fromMillis(claimedAt),
	}, nil
}

// DeleteParticipantClaim removes a claim by user.
func (s *Store) DeleteParticipantClaim(ctx context.Context, campaignID, userID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(userID) == "" {
		return fmt.Errorf("user id is required")
	}

	_, err := s.sqlDB.ExecContext(
		ctx,
		"DELETE FROM participant_claims WHERE campaign_id = ? AND user_id = ?",
		campaignID,
		userID,
	)
	if err != nil {
		return fmt.Errorf("delete participant claim: %w", err)
	}
	return nil
}
