package projection

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/invite"
)

func (a Applier) applyInviteCreated(ctx context.Context, evt event.Event) error {
	if a.Invite == nil {
		return fmt.Errorf("invite store is not configured")
	}
	if a.Campaign == nil {
		return fmt.Errorf("campaign store is not configured")
	}
	if strings.TrimSpace(evt.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	var payload event.InviteCreatedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode invite.created payload: %w", err)
	}
	inviteID := strings.TrimSpace(payload.InviteID)
	if inviteID == "" {
		inviteID = strings.TrimSpace(evt.EntityID)
	}
	if inviteID == "" {
		return fmt.Errorf("invite id is required")
	}
	participantID := strings.TrimSpace(payload.ParticipantID)
	if participantID == "" {
		return fmt.Errorf("invite.created participant_id is required")
	}
	status, err := parseInviteStatus(payload.Status)
	if err != nil {
		return err
	}

	createdAt := ensureTimestamp(evt.Timestamp)
	inv := invite.Invite{
		ID:                     inviteID,
		CampaignID:             evt.CampaignID,
		ParticipantID:          participantID,
		RecipientUserID:        strings.TrimSpace(payload.RecipientUserID),
		Status:                 status,
		CreatedByParticipantID: strings.TrimSpace(payload.CreatedByParticipantID),
		CreatedAt:              createdAt,
		UpdatedAt:              createdAt,
	}

	if err := a.Invite.PutInvite(ctx, inv); err != nil {
		return err
	}

	campaignRecord, err := a.Campaign.Get(ctx, evt.CampaignID)
	if err != nil {
		return err
	}
	campaignRecord.UpdatedAt = createdAt

	return a.Campaign.Put(ctx, campaignRecord)
}

func (a Applier) applyInviteClaimed(ctx context.Context, evt event.Event) error {
	if a.Invite == nil {
		return fmt.Errorf("invite store is not configured")
	}
	if a.Campaign == nil {
		return fmt.Errorf("campaign store is not configured")
	}
	inviteID := strings.TrimSpace(evt.EntityID)
	if inviteID == "" {
		return fmt.Errorf("invite id is required")
	}

	var payload event.InviteClaimedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode invite.claimed payload: %w", err)
	}
	if payload.InviteID != "" && strings.TrimSpace(payload.InviteID) != inviteID {
		return fmt.Errorf("invite.claimed invite_id mismatch")
	}

	updatedAt := ensureTimestamp(evt.Timestamp)
	if err := a.Invite.UpdateInviteStatus(ctx, inviteID, invite.StatusClaimed, updatedAt); err != nil {
		return err
	}

	campaignRecord, err := a.Campaign.Get(ctx, evt.CampaignID)
	if err != nil {
		return err
	}
	campaignRecord.UpdatedAt = updatedAt

	return a.Campaign.Put(ctx, campaignRecord)
}

func (a Applier) applyInviteRevoked(ctx context.Context, evt event.Event) error {
	if a.Invite == nil {
		return fmt.Errorf("invite store is not configured")
	}
	if a.Campaign == nil {
		return fmt.Errorf("campaign store is not configured")
	}
	inviteID := strings.TrimSpace(evt.EntityID)
	if inviteID == "" {
		return fmt.Errorf("invite id is required")
	}

	var payload event.InviteRevokedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode invite.revoked payload: %w", err)
	}
	if payload.InviteID != "" && strings.TrimSpace(payload.InviteID) != inviteID {
		return fmt.Errorf("invite.revoked invite_id mismatch")
	}

	updatedAt := ensureTimestamp(evt.Timestamp)
	if err := a.Invite.UpdateInviteStatus(ctx, inviteID, invite.StatusRevoked, updatedAt); err != nil {
		return err
	}

	campaignRecord, err := a.Campaign.Get(ctx, evt.CampaignID)
	if err != nil {
		return err
	}
	campaignRecord.UpdatedAt = updatedAt

	return a.Campaign.Put(ctx, campaignRecord)
}

func (a Applier) applyInviteUpdated(ctx context.Context, evt event.Event) error {
	if a.Invite == nil {
		return fmt.Errorf("invite store is not configured")
	}
	var payload event.InviteUpdatedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode invite.updated payload: %w", err)
	}
	inviteID := strings.TrimSpace(payload.InviteID)
	if inviteID == "" {
		inviteID = strings.TrimSpace(evt.EntityID)
	}
	if inviteID == "" {
		return fmt.Errorf("invite id is required")
	}
	status, err := parseInviteStatus(payload.Status)
	if err != nil {
		return err
	}
	updatedAt := ensureTimestamp(evt.Timestamp)
	return a.Invite.UpdateInviteStatus(ctx, inviteID, status, updatedAt)
}
