package projection

import (
	"context"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func (a Applier) applyParticipantJoined(ctx context.Context, evt event.Event, payload participant.JoinPayload) error {
	participantID := strings.TrimSpace(evt.EntityID)

	role, err := parseParticipantRole(payload.Role)
	if err != nil {
		return err
	}
	controller, err := parseParticipantController(payload.Controller)
	if err != nil {
		return err
	}
	access, err := parseCampaignAccess(payload.CampaignAccess)
	if err != nil {
		return err
	}
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		return fmt.Errorf("name is required")
	}
	userID := strings.TrimSpace(payload.UserID)

	createdAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	if err := a.Participant.PutParticipant(ctx, storage.ParticipantRecord{
		ID:             participantID,
		CampaignID:     strings.TrimSpace(evt.CampaignID),
		UserID:         userID,
		Name:           name,
		Role:           role,
		Controller:     controller,
		CampaignAccess: access,
		AvatarSetID:    strings.TrimSpace(payload.AvatarSetID),
		AvatarAssetID:  strings.TrimSpace(payload.AvatarAssetID),
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}); err != nil {
		return err
	}

	campaignRecord, err := a.Campaign.Get(ctx, evt.CampaignID)
	if err != nil {
		return err
	}
	pCount, err := a.Participant.CountParticipants(ctx, evt.CampaignID)
	if err != nil {
		return err
	}
	campaignRecord.ParticipantCount = pCount
	campaignRecord.UpdatedAt = createdAt

	return a.Campaign.Put(ctx, campaignRecord)
}

func (a Applier) applyParticipantUpdated(ctx context.Context, evt event.Event, payload participant.UpdatePayload) error {
	participantID := strings.TrimSpace(evt.EntityID)

	if len(payload.Fields) == 0 {
		return nil
	}

	current, err := a.Participant.GetParticipant(ctx, evt.CampaignID, participantID)
	if err != nil {
		return err
	}

	updated := current
	for key, value := range payload.Fields {
		switch key {
		case "user_id":
			updated.UserID = strings.TrimSpace(value)
		case "name":
			name := strings.TrimSpace(value)
			if name == "" {
				return fmt.Errorf("name is required")
			}
			updated.Name = name
		case "role":
			role, err := parseParticipantRole(value)
			if err != nil {
				return err
			}
			updated.Role = role
		case "controller":
			controller, err := parseParticipantController(value)
			if err != nil {
				return err
			}
			updated.Controller = controller
		case "campaign_access":
			access, err := parseCampaignAccess(value)
			if err != nil {
				return err
			}
			updated.CampaignAccess = access
		case "avatar_set_id":
			updated.AvatarSetID = strings.TrimSpace(value)
		case "avatar_asset_id":
			updated.AvatarAssetID = strings.TrimSpace(value)
		}
	}

	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	updated.UpdatedAt = updatedAt

	if err := a.Participant.PutParticipant(ctx, updated); err != nil {
		return err
	}

	campaignRecord, err := a.Campaign.Get(ctx, evt.CampaignID)
	if err != nil {
		return err
	}
	campaignRecord.UpdatedAt = updatedAt

	return a.Campaign.Put(ctx, campaignRecord)
}

func (a Applier) applyParticipantLeft(ctx context.Context, evt event.Event) error {
	participantID := strings.TrimSpace(evt.EntityID)

	if err := a.Participant.DeleteParticipant(ctx, evt.CampaignID, participantID); err != nil {
		return err
	}

	campaignRecord, err := a.Campaign.Get(ctx, evt.CampaignID)
	if err != nil {
		return err
	}
	pCount, err := a.Participant.CountParticipants(ctx, evt.CampaignID)
	if err != nil {
		return err
	}
	campaignRecord.ParticipantCount = pCount
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	campaignRecord.UpdatedAt = updatedAt

	return a.Campaign.Put(ctx, campaignRecord)
}

func (a Applier) applyParticipantBound(ctx context.Context, evt event.Event, payload participant.BindPayload) error {
	participantID := strings.TrimSpace(evt.EntityID)

	userID := strings.TrimSpace(payload.UserID)
	if userID == "" {
		return fmt.Errorf("participant.bound user_id is required")
	}

	current, err := a.Participant.GetParticipant(ctx, evt.CampaignID, participantID)
	if err != nil {
		return err
	}

	updated := current
	updated.UserID = userID
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	updated.UpdatedAt = updatedAt

	if a.ClaimIndex != nil {
		if err := a.ClaimIndex.PutParticipantClaim(ctx, evt.CampaignID, userID, participantID, updatedAt); err != nil {
			return err
		}
	}

	if err := a.Participant.PutParticipant(ctx, updated); err != nil {
		return err
	}

	campaignRecord, err := a.Campaign.Get(ctx, evt.CampaignID)
	if err != nil {
		return err
	}
	campaignRecord.UpdatedAt = updatedAt

	return a.Campaign.Put(ctx, campaignRecord)
}

func (a Applier) applyParticipantUnbound(ctx context.Context, evt event.Event, payload participant.UnbindPayload) error {
	participantID := strings.TrimSpace(evt.EntityID)

	requestedUserID := strings.TrimSpace(payload.UserID)

	current, err := a.Participant.GetParticipant(ctx, evt.CampaignID, participantID)
	if err != nil {
		return err
	}
	currentUserID := strings.TrimSpace(current.UserID)
	if requestedUserID != "" && requestedUserID != currentUserID {
		return fmt.Errorf("participant.unbound user_id mismatch")
	}

	if a.ClaimIndex != nil && currentUserID != "" {
		if err := a.ClaimIndex.DeleteParticipantClaim(ctx, evt.CampaignID, currentUserID); err != nil {
			return err
		}
	}

	updated := current
	updated.UserID = ""
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	updated.UpdatedAt = updatedAt

	if err := a.Participant.PutParticipant(ctx, updated); err != nil {
		return err
	}

	campaignRecord, err := a.Campaign.Get(ctx, evt.CampaignID)
	if err != nil {
		return err
	}
	campaignRecord.UpdatedAt = updatedAt

	return a.Campaign.Put(ctx, campaignRecord)
}

func (a Applier) applySeatReassigned(ctx context.Context, evt event.Event, payload participant.SeatReassignPayload) error {
	participantID := strings.TrimSpace(evt.EntityID)

	newUserID := strings.TrimSpace(payload.UserID)
	if newUserID == "" {
		return fmt.Errorf("seat.reassigned user_id is required")
	}
	priorUserID := strings.TrimSpace(payload.PriorUserID)

	current, err := a.Participant.GetParticipant(ctx, evt.CampaignID, participantID)
	if err != nil {
		return err
	}
	currentUserID := strings.TrimSpace(current.UserID)
	if priorUserID != "" && priorUserID != currentUserID {
		return fmt.Errorf("seat.reassigned prior_user_id mismatch")
	}

	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}

	if a.ClaimIndex != nil {
		if currentUserID != "" {
			if err := a.ClaimIndex.DeleteParticipantClaim(ctx, evt.CampaignID, currentUserID); err != nil {
				return err
			}
		}
		if err := a.ClaimIndex.PutParticipantClaim(ctx, evt.CampaignID, newUserID, participantID, updatedAt); err != nil {
			return err
		}
	}

	updated := current
	updated.UserID = newUserID
	updated.UpdatedAt = updatedAt

	if err := a.Participant.PutParticipant(ctx, updated); err != nil {
		return err
	}

	campaignRecord, err := a.Campaign.Get(ctx, evt.CampaignID)
	if err != nil {
		return err
	}
	campaignRecord.UpdatedAt = updatedAt

	return a.Campaign.Put(ctx, campaignRecord)
}
