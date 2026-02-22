package projection

import (
	"context"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func (a Applier) applyInviteCreated(ctx context.Context, evt event.Event) error {
	var payload invite.CreatePayload
	if err := decodePayload(evt.PayloadJSON, &payload, "invite.created"); err != nil {
		return err
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

	createdAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	inv := storage.InviteRecord{
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
	inviteID := strings.TrimSpace(evt.EntityID)

	var payload invite.ClaimPayload
	if err := decodePayload(evt.PayloadJSON, &payload, "invite.claimed"); err != nil {
		return err
	}
	if payload.InviteID != "" && strings.TrimSpace(payload.InviteID) != inviteID {
		return fmt.Errorf("invite.claimed invite_id mismatch")
	}

	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
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
	inviteID := strings.TrimSpace(evt.EntityID)

	var payload invite.RevokePayload
	if err := decodePayload(evt.PayloadJSON, &payload, "invite.revoked"); err != nil {
		return err
	}
	if payload.InviteID != "" && strings.TrimSpace(payload.InviteID) != inviteID {
		return fmt.Errorf("invite.revoked invite_id mismatch")
	}

	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
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
	var payload invite.UpdatePayload
	if err := decodePayload(evt.PayloadJSON, &payload, "invite.updated"); err != nil {
		return err
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
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	return a.Invite.UpdateInviteStatus(ctx, inviteID, status, updatedAt)
}
