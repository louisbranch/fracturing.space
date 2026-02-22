package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

// Invite methods.

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
