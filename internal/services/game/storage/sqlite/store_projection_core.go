package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

func (s *Store) Put(ctx context.Context, c storage.CampaignRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(c.ID) == "" {
		return fmt.Errorf("campaign id is required")
	}

	completedAt := toNullMillis(c.CompletedAt)
	archivedAt := toNullMillis(c.ArchivedAt)

	return s.q.PutCampaign(ctx, db.PutCampaignParams{
		ID:               c.ID,
		Name:             c.Name,
		Locale:           platformi18n.LocaleString(c.Locale),
		GameSystem:       gameSystemToString(c.System),
		Status:           campaignStatusToString(c.Status),
		GmMode:           gmModeToString(c.GmMode),
		Intent:           campaignIntentToString(c.Intent),
		AccessPolicy:     campaignAccessPolicyToString(c.AccessPolicy),
		ParticipantCount: int64(c.ParticipantCount),
		CharacterCount:   int64(c.CharacterCount),
		ThemePrompt:      c.ThemePrompt,
		CoverAssetID:     c.CoverAssetID,
		CoverSetID:       c.CoverSetID,
		CreatedAt:        toMillis(c.CreatedAt),
		UpdatedAt:        toMillis(c.UpdatedAt),
		CompletedAt:      completedAt,
		ArchivedAt:       archivedAt,
	})
}

// Get fetches a campaign record by ID.
func (s *Store) Get(ctx context.Context, id string) (storage.CampaignRecord, error) {
	if err := ctx.Err(); err != nil {
		return storage.CampaignRecord{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.CampaignRecord{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return storage.CampaignRecord{}, fmt.Errorf("campaign id is required")
	}

	row, err := s.q.GetCampaign(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.CampaignRecord{}, storage.ErrNotFound
		}
		return storage.CampaignRecord{}, fmt.Errorf("get campaign: %w", err)
	}

	return dbGetCampaignRowToDomain(row)
}

// List returns a page of campaign records ordered by storage key.
func (s *Store) List(ctx context.Context, pageSize int, pageToken string) (storage.CampaignPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.CampaignPage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.CampaignPage{}, fmt.Errorf("storage is not configured")
	}
	if pageSize <= 0 {
		return storage.CampaignPage{}, fmt.Errorf("page size must be greater than zero")
	}

	page := storage.CampaignPage{
		Campaigns: make([]storage.CampaignRecord, 0, pageSize),
	}

	if pageToken == "" {
		rows, err := s.q.ListAllCampaigns(ctx, int64(pageSize+1))
		if err != nil {
			return storage.CampaignPage{}, fmt.Errorf("list campaigns: %w", err)
		}
		for i, row := range rows {
			if i >= pageSize {
				page.NextPageToken = rows[pageSize-1].ID
				break
			}
			c, err := dbListAllCampaignsRowToDomain(row)
			if err != nil {
				return storage.CampaignPage{}, err
			}
			page.Campaigns = append(page.Campaigns, c)
		}
	} else {
		rows, err := s.q.ListCampaigns(ctx, db.ListCampaignsParams{
			ID:    pageToken,
			Limit: int64(pageSize + 1),
		})
		if err != nil {
			return storage.CampaignPage{}, fmt.Errorf("list campaigns: %w", err)
		}
		for i, row := range rows {
			if i >= pageSize {
				page.NextPageToken = rows[pageSize-1].ID
				break
			}
			c, err := dbListCampaignsRowToDomain(row)
			if err != nil {
				return storage.CampaignPage{}, err
			}
			page.Campaigns = append(page.Campaigns, c)
		}
	}

	return page, nil
}

// Participant methods

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

// Invite methods

// PutInvite persists an invite record.
func (s *Store) PutInvite(ctx context.Context, inv storage.InviteRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(inv.ID) == "" {
		return fmt.Errorf("invite id is required")
	}
	if strings.TrimSpace(inv.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(inv.ParticipantID) == "" {
		return fmt.Errorf("participant id is required")
	}

	return s.q.PutInvite(ctx, db.PutInviteParams{
		ID:                     inv.ID,
		CampaignID:             inv.CampaignID,
		ParticipantID:          inv.ParticipantID,
		RecipientUserID:        strings.TrimSpace(inv.RecipientUserID),
		Status:                 inviteStatusToString(inv.Status),
		CreatedByParticipantID: inv.CreatedByParticipantID,
		CreatedAt:              toMillis(inv.CreatedAt),
		UpdatedAt:              toMillis(inv.UpdatedAt),
	})
}

// GetInvite fetches an invite record by ID.
func (s *Store) GetInvite(ctx context.Context, inviteID string) (storage.InviteRecord, error) {
	if err := ctx.Err(); err != nil {
		return storage.InviteRecord{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.InviteRecord{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(inviteID) == "" {
		return storage.InviteRecord{}, fmt.Errorf("invite id is required")
	}

	row, err := s.q.GetInvite(ctx, inviteID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.InviteRecord{}, storage.ErrNotFound
		}
		return storage.InviteRecord{}, fmt.Errorf("get invite: %w", err)
	}

	return dbInviteToDomain(row)
}

// ListInvites returns a page of invite records for a campaign.
func (s *Store) ListInvites(ctx context.Context, campaignID string, recipientUserID string, status invite.Status, pageSize int, pageToken string) (storage.InvitePage, error) {
	if err := ctx.Err(); err != nil {
		return storage.InvitePage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.InvitePage{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.InvitePage{}, fmt.Errorf("campaign id is required")
	}
	if pageSize <= 0 {
		return storage.InvitePage{}, fmt.Errorf("page size must be greater than zero")
	}
	recipientUserID = strings.TrimSpace(recipientUserID)
	statusFilter := ""
	if status != invite.StatusUnspecified {
		statusFilter = inviteStatusToString(status)
	}

	var rows []db.Invite
	var err error
	if pageToken == "" {
		rows, err = s.q.ListInvitesByCampaignPagedFirst(ctx, db.ListInvitesByCampaignPagedFirstParams{
			CampaignID:      campaignID,
			Column2:         recipientUserID,
			RecipientUserID: recipientUserID,
			Column4:         statusFilter,
			Status:          statusFilter,
			Limit:           int64(pageSize + 1),
		})
	} else {
		rows, err = s.q.ListInvitesByCampaignPaged(ctx, db.ListInvitesByCampaignPagedParams{
			CampaignID:      campaignID,
			ID:              pageToken,
			Column3:         recipientUserID,
			RecipientUserID: recipientUserID,
			Column5:         statusFilter,
			Status:          statusFilter,
			Limit:           int64(pageSize + 1),
		})
	}
	if err != nil {
		return storage.InvitePage{}, fmt.Errorf("list invites: %w", err)
	}

	page := storage.InvitePage{Invites: make([]storage.InviteRecord, 0, pageSize)}
	for i, row := range rows {
		if i >= pageSize {
			page.NextPageToken = rows[pageSize-1].ID
			break
		}
		inv, err := dbInviteToDomain(row)
		if err != nil {
			return storage.InvitePage{}, err
		}
		page.Invites = append(page.Invites, inv)
	}

	return page, nil
}

// ListPendingInvites returns a page of pending invite records for a campaign.
func (s *Store) ListPendingInvites(ctx context.Context, campaignID string, pageSize int, pageToken string) (storage.InvitePage, error) {
	if err := ctx.Err(); err != nil {
		return storage.InvitePage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.InvitePage{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.InvitePage{}, fmt.Errorf("campaign id is required")
	}
	if pageSize <= 0 {
		return storage.InvitePage{}, fmt.Errorf("page size must be greater than zero")
	}

	status := inviteStatusToString(invite.StatusPending)
	var rows []db.Invite
	var err error
	if pageToken == "" {
		rows, err = s.q.ListPendingInvitesByCampaignPagedFirst(ctx, db.ListPendingInvitesByCampaignPagedFirstParams{
			CampaignID: campaignID,
			Status:     status,
			Limit:      int64(pageSize + 1),
		})
	} else {
		rows, err = s.q.ListPendingInvitesByCampaignPaged(ctx, db.ListPendingInvitesByCampaignPagedParams{
			CampaignID: campaignID,
			Status:     status,
			ID:         pageToken,
			Limit:      int64(pageSize + 1),
		})
	}
	if err != nil {
		return storage.InvitePage{}, fmt.Errorf("list pending invites: %w", err)
	}

	page := storage.InvitePage{Invites: make([]storage.InviteRecord, 0, pageSize)}
	for i, row := range rows {
		if i >= pageSize {
			page.NextPageToken = rows[pageSize-1].ID
			break
		}
		inv, err := dbInviteToDomain(row)
		if err != nil {
			return storage.InvitePage{}, err
		}
		page.Invites = append(page.Invites, inv)
	}

	return page, nil
}

// ListPendingInvitesForRecipient returns a page of pending invite records for a user.
func (s *Store) ListPendingInvitesForRecipient(ctx context.Context, userID string, pageSize int, pageToken string) (storage.InvitePage, error) {
	if err := ctx.Err(); err != nil {
		return storage.InvitePage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.InvitePage{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(userID) == "" {
		return storage.InvitePage{}, fmt.Errorf("user id is required")
	}
	if pageSize <= 0 {
		return storage.InvitePage{}, fmt.Errorf("page size must be greater than zero")
	}

	status := inviteStatusToString(invite.StatusPending)
	var rows []db.Invite
	var err error
	if pageToken == "" {
		rows, err = s.q.ListPendingInvitesByRecipientPagedFirst(ctx, db.ListPendingInvitesByRecipientPagedFirstParams{
			RecipientUserID: userID,
			Status:          status,
			Limit:           int64(pageSize + 1),
		})
	} else {
		rows, err = s.q.ListPendingInvitesByRecipientPaged(ctx, db.ListPendingInvitesByRecipientPagedParams{
			RecipientUserID: userID,
			Status:          status,
			ID:              pageToken,
			Limit:           int64(pageSize + 1),
		})
	}
	if err != nil {
		return storage.InvitePage{}, fmt.Errorf("list pending invites for recipient: %w", err)
	}

	page := storage.InvitePage{Invites: make([]storage.InviteRecord, 0, pageSize)}
	for i, row := range rows {
		if i >= pageSize {
			page.NextPageToken = rows[pageSize-1].ID
			break
		}
		inv, err := dbInviteToDomain(row)
		if err != nil {
			return storage.InvitePage{}, err
		}
		page.Invites = append(page.Invites, inv)
	}

	return page, nil
}

// UpdateInviteStatus updates the status for an invite.
func (s *Store) UpdateInviteStatus(ctx context.Context, inviteID string, status invite.Status, updatedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(inviteID) == "" {
		return fmt.Errorf("invite id is required")
	}
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}

	return s.q.UpdateInviteStatus(ctx, db.UpdateInviteStatusParams{
		Status:    inviteStatusToString(status),
		UpdatedAt: toMillis(updatedAt),
		ID:        inviteID,
	})
}

// Character methods

// PutCharacter persists a character record.
func (s *Store) PutCharacter(ctx context.Context, c storage.CharacterRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(c.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(c.ID) == "" {
		return fmt.Errorf("character id is required")
	}

	return s.q.PutCharacter(ctx, db.PutCharacterParams{
		CampaignID:              c.CampaignID,
		ID:                      c.ID,
		ControllerParticipantID: toNullString(c.ParticipantID),
		Name:                    c.Name,
		Kind:                    characterKindToString(c.Kind),
		Notes:                   c.Notes,
		AvatarSetID:             c.AvatarSetID,
		AvatarAssetID:           c.AvatarAssetID,
		CreatedAt:               toMillis(c.CreatedAt),
		UpdatedAt:               toMillis(c.UpdatedAt),
	})
}

// DeleteCharacter deletes a character record by IDs.
func (s *Store) DeleteCharacter(ctx context.Context, campaignID, characterID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(characterID) == "" {
		return fmt.Errorf("character id is required")
	}

	return s.q.DeleteCharacter(ctx, db.DeleteCharacterParams{
		CampaignID: campaignID,
		ID:         characterID,
	})
}

// GetCharacter fetches a character record by IDs.
func (s *Store) GetCharacter(ctx context.Context, campaignID, characterID string) (storage.CharacterRecord, error) {
	if err := ctx.Err(); err != nil {
		return storage.CharacterRecord{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.CharacterRecord{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.CharacterRecord{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(characterID) == "" {
		return storage.CharacterRecord{}, fmt.Errorf("character id is required")
	}

	row, err := s.q.GetCharacter(ctx, db.GetCharacterParams{
		CampaignID: campaignID,
		ID:         characterID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.CharacterRecord{}, storage.ErrNotFound
		}
		return storage.CharacterRecord{}, fmt.Errorf("get character: %w", err)
	}

	return dbCharacterToDomain(row)
}

// ListCharacters returns a page of character records.
func (s *Store) ListCharacters(ctx context.Context, campaignID string, pageSize int, pageToken string) (storage.CharacterPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.CharacterPage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.CharacterPage{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.CharacterPage{}, fmt.Errorf("campaign id is required")
	}
	if pageSize <= 0 {
		return storage.CharacterPage{}, fmt.Errorf("page size must be greater than zero")
	}

	var rows []db.Character
	var err error

	if pageToken == "" {
		rows, err = s.q.ListCharactersByCampaignPagedFirst(ctx, db.ListCharactersByCampaignPagedFirstParams{
			CampaignID: campaignID,
			Limit:      int64(pageSize + 1),
		})
	} else {
		rows, err = s.q.ListCharactersByCampaignPaged(ctx, db.ListCharactersByCampaignPagedParams{
			CampaignID: campaignID,
			ID:         pageToken,
			Limit:      int64(pageSize + 1),
		})
	}
	if err != nil {
		return storage.CharacterPage{}, fmt.Errorf("list characters: %w", err)
	}

	page := storage.CharacterPage{
		Characters: make([]storage.CharacterRecord, 0, pageSize),
	}

	for i, row := range rows {
		if i >= pageSize {
			page.NextPageToken = rows[pageSize-1].ID
			break
		}
		c, err := dbCharacterToDomain(row)
		if err != nil {
			return storage.CharacterPage{}, err
		}
		page.Characters = append(page.Characters, c)
	}

	return page, nil
}

// CountCharacters returns the number of characters for a campaign.
func (s *Store) CountCharacters(ctx context.Context, campaignID string) (int, error) {
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
		"SELECT COUNT(*) FROM characters WHERE campaign_id = ?", campaignID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count characters: %w", err)
	}
	return int(count), nil
}

// Session methods

// PutSession atomically stores a session and sets it as the active session for the campaign.
func (s *Store) PutSession(ctx context.Context, sess storage.SessionRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(sess.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sess.ID) == "" {
		return fmt.Errorf("session id is required")
	}

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := s.q.WithTx(tx)

	if sess.Status == session.StatusActive {
		hasActive, err := qtx.HasActiveSession(ctx, sess.CampaignID)
		if err != nil {
			return fmt.Errorf("check active session: %w", err)
		}
		if hasActive != 0 {
			return storage.ErrActiveSessionExists
		}
	}

	endedAt := toNullMillis(sess.EndedAt)

	if err := qtx.PutSession(ctx, db.PutSessionParams{
		CampaignID: sess.CampaignID,
		ID:         sess.ID,
		Name:       sess.Name,
		Status:     sessionStatusToString(sess.Status),
		StartedAt:  toMillis(sess.StartedAt),
		UpdatedAt:  toMillis(sess.UpdatedAt),
		EndedAt:    endedAt,
	}); err != nil {
		return fmt.Errorf("put session: %w", err)
	}

	if sess.Status == session.StatusActive {
		if err := qtx.SetActiveSession(ctx, db.SetActiveSessionParams{
			CampaignID: sess.CampaignID,
			SessionID:  sess.ID,
		}); err != nil {
			return fmt.Errorf("set active session: %w", err)
		}
	}

	return tx.Commit()
}

// EndSession marks a session as ended and clears it as active for the campaign.
func (s *Store) EndSession(ctx context.Context, campaignID, sessionID string, endedAt time.Time) (storage.SessionRecord, bool, error) {
	if err := ctx.Err(); err != nil {
		return storage.SessionRecord{}, false, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.SessionRecord{}, false, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.SessionRecord{}, false, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return storage.SessionRecord{}, false, fmt.Errorf("session id is required")
	}

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return storage.SessionRecord{}, false, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := s.q.WithTx(tx)

	row, err := qtx.GetSession(ctx, db.GetSessionParams{
		CampaignID: campaignID,
		ID:         sessionID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.SessionRecord{}, false, storage.ErrNotFound
		}
		return storage.SessionRecord{}, false, fmt.Errorf("get session: %w", err)
	}

	sess, err := dbSessionToDomain(row)
	if err != nil {
		return storage.SessionRecord{}, false, err
	}

	transitioned := false
	if sess.Status == session.StatusActive {
		transitioned = true
		sess.Status = session.StatusEnded
		sess.UpdatedAt = endedAt.UTC()
		sess.EndedAt = &sess.UpdatedAt

		if err := qtx.UpdateSessionStatus(ctx, db.UpdateSessionStatusParams{
			Status:     sessionStatusToString(sess.Status),
			UpdatedAt:  toMillis(sess.UpdatedAt),
			EndedAt:    toNullMillis(sess.EndedAt),
			CampaignID: campaignID,
			ID:         sessionID,
		}); err != nil {
			return storage.SessionRecord{}, false, fmt.Errorf("update session status: %w", err)
		}
	}

	if err := qtx.ClearActiveSession(ctx, campaignID); err != nil {
		return storage.SessionRecord{}, false, fmt.Errorf("clear active session: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return storage.SessionRecord{}, false, fmt.Errorf("commit: %w", err)
	}

	return sess, transitioned, nil
}

// GetSession retrieves a session by campaign ID and session ID.
func (s *Store) GetSession(ctx context.Context, campaignID, sessionID string) (storage.SessionRecord, error) {
	if err := ctx.Err(); err != nil {
		return storage.SessionRecord{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.SessionRecord{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.SessionRecord{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return storage.SessionRecord{}, fmt.Errorf("session id is required")
	}

	row, err := s.q.GetSession(ctx, db.GetSessionParams{
		CampaignID: campaignID,
		ID:         sessionID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.SessionRecord{}, storage.ErrNotFound
		}
		return storage.SessionRecord{}, fmt.Errorf("get session: %w", err)
	}

	return dbSessionToDomain(row)
}

// GetActiveSession retrieves the active session for a campaign.
func (s *Store) GetActiveSession(ctx context.Context, campaignID string) (storage.SessionRecord, error) {
	if err := ctx.Err(); err != nil {
		return storage.SessionRecord{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.SessionRecord{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.SessionRecord{}, fmt.Errorf("campaign id is required")
	}

	row, err := s.q.GetActiveSession(ctx, campaignID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.SessionRecord{}, storage.ErrNotFound
		}
		return storage.SessionRecord{}, fmt.Errorf("get active session: %w", err)
	}

	return dbSessionToDomain(row)
}

// ListSessions returns a page of session records.
func (s *Store) ListSessions(ctx context.Context, campaignID string, pageSize int, pageToken string) (storage.SessionPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.SessionPage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.SessionPage{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.SessionPage{}, fmt.Errorf("campaign id is required")
	}
	if pageSize <= 0 {
		return storage.SessionPage{}, fmt.Errorf("page size must be greater than zero")
	}

	var rows []db.Session
	var err error

	if pageToken == "" {
		rows, err = s.q.ListSessionsByCampaignPagedFirst(ctx, db.ListSessionsByCampaignPagedFirstParams{
			CampaignID: campaignID,
			Limit:      int64(pageSize + 1),
		})
	} else {
		rows, err = s.q.ListSessionsByCampaignPaged(ctx, db.ListSessionsByCampaignPagedParams{
			CampaignID: campaignID,
			ID:         pageToken,
			Limit:      int64(pageSize + 1),
		})
	}
	if err != nil {
		return storage.SessionPage{}, fmt.Errorf("list sessions: %w", err)
	}

	page := storage.SessionPage{
		Sessions: make([]storage.SessionRecord, 0, pageSize),
	}

	for i, row := range rows {
		if i >= pageSize {
			page.NextPageToken = rows[pageSize-1].ID
			break
		}
		sess, err := dbSessionToDomain(row)
		if err != nil {
			return storage.SessionPage{}, err
		}
		page.Sessions = append(page.Sessions, sess)
	}

	return page, nil
}

// Session gate methods

// PutSessionGate persists a session gate projection.
func (s *Store) PutSessionGate(ctx context.Context, gate storage.SessionGate) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(gate.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(gate.SessionID) == "" {
		return fmt.Errorf("session id is required")
	}
	if strings.TrimSpace(gate.GateID) == "" {
		return fmt.Errorf("gate id is required")
	}
	if strings.TrimSpace(gate.GateType) == "" {
		return fmt.Errorf("gate type is required")
	}
	status := strings.TrimSpace(string(gate.Status))
	if status == "" {
		return fmt.Errorf("gate status is required")
	}

	return s.q.PutSessionGate(ctx, db.PutSessionGateParams{
		CampaignID:          gate.CampaignID,
		SessionID:           gate.SessionID,
		GateID:              gate.GateID,
		GateType:            gate.GateType,
		Status:              status,
		Reason:              gate.Reason,
		CreatedAt:           toMillis(gate.CreatedAt),
		CreatedByActorType:  gate.CreatedByActorType,
		CreatedByActorID:    gate.CreatedByActorID,
		ResolvedAt:          toNullMillis(gate.ResolvedAt),
		ResolvedByActorType: toNullString(gate.ResolvedByActorType),
		ResolvedByActorID:   toNullString(gate.ResolvedByActorID),
		MetadataJson:        gate.MetadataJSON,
		ResolutionJson:      gate.ResolutionJSON,
	})
}

// GetSessionGate retrieves a session gate by id.
func (s *Store) GetSessionGate(ctx context.Context, campaignID, sessionID, gateID string) (storage.SessionGate, error) {
	if err := ctx.Err(); err != nil {
		return storage.SessionGate{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.SessionGate{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.SessionGate{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return storage.SessionGate{}, fmt.Errorf("session id is required")
	}
	if strings.TrimSpace(gateID) == "" {
		return storage.SessionGate{}, fmt.Errorf("gate id is required")
	}

	row, err := s.q.GetSessionGate(ctx, db.GetSessionGateParams{
		CampaignID: campaignID,
		SessionID:  sessionID,
		GateID:     gateID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.SessionGate{}, storage.ErrNotFound
		}
		return storage.SessionGate{}, fmt.Errorf("get session gate: %w", err)
	}

	return dbSessionGateToStorage(row), nil
}

// GetOpenSessionGate retrieves the open gate for a session.
func (s *Store) GetOpenSessionGate(ctx context.Context, campaignID, sessionID string) (storage.SessionGate, error) {
	if err := ctx.Err(); err != nil {
		return storage.SessionGate{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.SessionGate{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.SessionGate{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return storage.SessionGate{}, fmt.Errorf("session id is required")
	}

	row, err := s.q.GetOpenSessionGate(ctx, db.GetOpenSessionGateParams{
		CampaignID: campaignID,
		SessionID:  sessionID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.SessionGate{}, storage.ErrNotFound
		}
		return storage.SessionGate{}, fmt.Errorf("get open session gate: %w", err)
	}

	return dbSessionGateToStorage(row), nil
}

// Session spotlight methods

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
