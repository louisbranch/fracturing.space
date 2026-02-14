package projection

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/participant"
)

func (a Applier) applyParticipantJoined(ctx context.Context, evt event.Event) error {
	if a.Participant == nil {
		return fmt.Errorf("participant store is not configured")
	}
	if a.Campaign == nil {
		return fmt.Errorf("campaign store is not configured")
	}
	participantID := strings.TrimSpace(evt.EntityID)
	if participantID == "" {
		return fmt.Errorf("participant id is required")
	}
	if strings.TrimSpace(evt.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}

	var payload event.ParticipantJoinedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode participant.joined payload: %w", err)
	}

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

	input := participant.CreateParticipantInput{
		CampaignID:     evt.CampaignID,
		UserID:         payload.UserID,
		DisplayName:    payload.DisplayName,
		Role:           role,
		Controller:     controller,
		CampaignAccess: access,
	}
	normalized, err := participant.NormalizeCreateParticipantInput(input)
	if err != nil {
		return err
	}

	createdAt := ensureTimestamp(evt.Timestamp)
	p := participant.Participant{
		ID:             participantID,
		CampaignID:     normalized.CampaignID,
		UserID:         normalized.UserID,
		DisplayName:    normalized.DisplayName,
		Role:           normalized.Role,
		Controller:     normalized.Controller,
		CampaignAccess: normalized.CampaignAccess,
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}
	if err := a.Participant.PutParticipant(ctx, p); err != nil {
		return err
	}

	campaignRecord, err := a.Campaign.Get(ctx, evt.CampaignID)
	if err != nil {
		return err
	}
	campaignRecord.ParticipantCount++
	campaignRecord.UpdatedAt = createdAt

	return a.Campaign.Put(ctx, campaignRecord)
}

func (a Applier) applyParticipantUpdated(ctx context.Context, evt event.Event) error {
	if a.Participant == nil {
		return fmt.Errorf("participant store is not configured")
	}
	if a.Campaign == nil {
		return fmt.Errorf("campaign store is not configured")
	}
	if strings.TrimSpace(evt.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	participantID := strings.TrimSpace(evt.EntityID)
	if participantID == "" {
		return fmt.Errorf("participant id is required")
	}

	var payload event.ParticipantUpdatedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode participant.updated payload: %w", err)
	}
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
			userID, ok := value.(string)
			if !ok {
				return fmt.Errorf("participant.updated user_id must be string")
			}
			updated.UserID = strings.TrimSpace(userID)
		case "display_name":
			name, ok := value.(string)
			if !ok {
				return fmt.Errorf("participant.updated display_name must be string")
			}
			name = strings.TrimSpace(name)
			if name == "" {
				return fmt.Errorf("display name is required")
			}
			updated.DisplayName = name
		case "role":
			roleLabel, ok := value.(string)
			if !ok {
				return fmt.Errorf("participant.updated role must be string")
			}
			role, err := parseParticipantRole(roleLabel)
			if err != nil {
				return err
			}
			updated.Role = role
		case "controller":
			controllerLabel, ok := value.(string)
			if !ok {
				return fmt.Errorf("participant.updated controller must be string")
			}
			controller, err := parseParticipantController(controllerLabel)
			if err != nil {
				return err
			}
			updated.Controller = controller
		case "campaign_access":
			accessLabel, ok := value.(string)
			if !ok {
				return fmt.Errorf("participant.updated campaign_access must be string")
			}
			access, err := parseCampaignAccess(accessLabel)
			if err != nil {
				return err
			}
			updated.CampaignAccess = access
		}
	}

	updatedAt := ensureTimestamp(evt.Timestamp)
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
	if a.Participant == nil {
		return fmt.Errorf("participant store is not configured")
	}
	if a.Campaign == nil {
		return fmt.Errorf("campaign store is not configured")
	}
	if strings.TrimSpace(evt.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	participantID := strings.TrimSpace(evt.EntityID)
	if participantID == "" {
		return fmt.Errorf("participant id is required")
	}

	if err := a.Participant.DeleteParticipant(ctx, evt.CampaignID, participantID); err != nil {
		return err
	}

	campaignRecord, err := a.Campaign.Get(ctx, evt.CampaignID)
	if err != nil {
		return err
	}
	if campaignRecord.ParticipantCount > 0 {
		campaignRecord.ParticipantCount--
	}
	updatedAt := ensureTimestamp(evt.Timestamp)
	campaignRecord.UpdatedAt = updatedAt

	return a.Campaign.Put(ctx, campaignRecord)
}

func (a Applier) applyParticipantBound(ctx context.Context, evt event.Event) error {
	if a.Participant == nil {
		return fmt.Errorf("participant store is not configured")
	}
	if a.Campaign == nil {
		return fmt.Errorf("campaign store is not configured")
	}
	if strings.TrimSpace(evt.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	participantID := strings.TrimSpace(evt.EntityID)
	if participantID == "" {
		return fmt.Errorf("participant id is required")
	}

	var payload event.ParticipantBoundPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode participant.bound payload: %w", err)
	}
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
	updatedAt := ensureTimestamp(evt.Timestamp)
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

func (a Applier) applyParticipantUnbound(ctx context.Context, evt event.Event) error {
	if a.Participant == nil {
		return fmt.Errorf("participant store is not configured")
	}
	if a.Campaign == nil {
		return fmt.Errorf("campaign store is not configured")
	}
	if strings.TrimSpace(evt.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	participantID := strings.TrimSpace(evt.EntityID)
	if participantID == "" {
		return fmt.Errorf("participant id is required")
	}

	var payload event.ParticipantUnboundPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode participant.unbound payload: %w", err)
	}
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
	updatedAt := ensureTimestamp(evt.Timestamp)
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

func (a Applier) applySeatReassigned(ctx context.Context, evt event.Event) error {
	if a.Participant == nil {
		return fmt.Errorf("participant store is not configured")
	}
	if a.Campaign == nil {
		return fmt.Errorf("campaign store is not configured")
	}
	if strings.TrimSpace(evt.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	participantID := strings.TrimSpace(evt.EntityID)
	if participantID == "" {
		return fmt.Errorf("participant id is required")
	}

	var payload event.SeatReassignedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode seat.reassigned payload: %w", err)
	}
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

	if a.ClaimIndex != nil {
		if currentUserID != "" {
			if err := a.ClaimIndex.DeleteParticipantClaim(ctx, evt.CampaignID, currentUserID); err != nil {
				return err
			}
		}
		if err := a.ClaimIndex.PutParticipantClaim(ctx, evt.CampaignID, newUserID, participantID, ensureTimestamp(evt.Timestamp)); err != nil {
			return err
		}
	}

	updated := current
	updated.UserID = newUserID
	updatedAt := ensureTimestamp(evt.Timestamp)
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
