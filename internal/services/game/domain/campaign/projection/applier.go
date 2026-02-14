package projection

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// Applier applies event journal entries to projection stores.
type Applier struct {
	Campaign     storage.CampaignStore
	Character    storage.CharacterStore
	CampaignFork storage.CampaignForkStore
	Daggerheart  storage.DaggerheartStore
	ClaimIndex   storage.ClaimIndexStore
	Invite       storage.InviteStore
	Participant  storage.ParticipantStore
	Session      storage.SessionStore
	Adapters     *systems.AdapterRegistry
}

// Apply applies an event to projection stores.
func (a Applier) Apply(ctx context.Context, evt event.Event) error {
	switch evt.Type {
	case event.TypeCampaignCreated:
		return a.applyCampaignCreated(ctx, evt)
	case event.TypeCampaignForked:
		return a.applyCampaignForked(ctx, evt)
	case event.TypeCampaignUpdated:
		return a.applyCampaignUpdated(ctx, evt)
	case event.TypeParticipantJoined:
		return a.applyParticipantJoined(ctx, evt)
	case event.TypeParticipantUpdated:
		return a.applyParticipantUpdated(ctx, evt)
	case event.TypeParticipantLeft:
		return a.applyParticipantLeft(ctx, evt)
	case event.TypeParticipantBound:
		return a.applyParticipantBound(ctx, evt)
	case event.TypeParticipantUnbound:
		return a.applyParticipantUnbound(ctx, evt)
	case event.TypeSeatReassigned:
		return a.applySeatReassigned(ctx, evt)
	case event.TypeInviteCreated:
		return a.applyInviteCreated(ctx, evt)
	case event.TypeInviteClaimed:
		return a.applyInviteClaimed(ctx, evt)
	case event.TypeInviteRevoked:
		return a.applyInviteRevoked(ctx, evt)
	case event.TypeCharacterCreated:
		return a.applyCharacterCreated(ctx, evt)
	case event.TypeCharacterUpdated:
		return a.applyCharacterUpdated(ctx, evt)
	case event.TypeCharacterDeleted:
		return a.applyCharacterDeleted(ctx, evt)
	case event.TypeProfileUpdated:
		return a.applyProfileUpdated(ctx, evt)
	case event.TypeInviteUpdated:
		return a.applyInviteUpdated(ctx, evt)
	case event.TypeSessionStarted:
		return a.applySessionStarted(ctx, evt)
	case event.TypeSessionEnded:
		return a.applySessionEnded(ctx, evt)
	default:
		if strings.TrimSpace(evt.SystemID) != "" {
			return a.applySystemEvent(ctx, evt)
		}
		return nil
	}
}

func (a Applier) applyCampaignCreated(ctx context.Context, evt event.Event) error {
	if a.Campaign == nil {
		return fmt.Errorf("campaign store is not configured")
	}
	if strings.TrimSpace(evt.EntityID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	var payload event.CampaignCreatedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode campaign.created payload: %w", err)
	}

	system, err := parseGameSystem(payload.GameSystem)
	if err != nil {
		return err
	}
	gmMode, err := parseGmMode(payload.GmMode)
	if err != nil {
		return err
	}

	input := campaign.CreateCampaignInput{
		Name:        payload.Name,
		System:      system,
		GmMode:      gmMode,
		ThemePrompt: payload.ThemePrompt,
	}
	normalized, err := campaign.NormalizeCreateCampaignInput(input)
	if err != nil {
		return err
	}

	createdAt := ensureTimestamp(evt.Timestamp)
	c := campaign.Campaign{
		ID:               evt.EntityID,
		Name:             normalized.Name,
		System:           normalized.System,
		Status:           campaign.CampaignStatusDraft,
		GmMode:           normalized.GmMode,
		ParticipantCount: 0,
		CharacterCount:   0,
		ThemePrompt:      normalized.ThemePrompt,
		CreatedAt:        createdAt,
		UpdatedAt:        createdAt,
	}

	return a.Campaign.Put(ctx, c)
}

func (a Applier) applyCampaignForked(ctx context.Context, evt event.Event) error {
	if a.CampaignFork == nil {
		return fmt.Errorf("campaign fork store is not configured")
	}
	if strings.TrimSpace(evt.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	var payload event.CampaignForkedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode campaign.forked payload: %w", err)
	}
	return a.CampaignFork.SetCampaignForkMetadata(ctx, evt.CampaignID, storage.ForkMetadata{
		ParentCampaignID: payload.ParentCampaignID,
		ForkEventSeq:     payload.ForkEventSeq,
		OriginCampaignID: payload.OriginCampaignID,
	})
}

func (a Applier) applyCampaignUpdated(ctx context.Context, evt event.Event) error {
	if a.Campaign == nil {
		return fmt.Errorf("campaign store is not configured")
	}
	if strings.TrimSpace(evt.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	var payload event.CampaignUpdatedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode campaign.updated payload: %w", err)
	}
	if len(payload.Fields) == 0 {
		return nil
	}

	current, err := a.Campaign.Get(ctx, evt.CampaignID)
	if err != nil {
		return err
	}

	updated := current
	for key, value := range payload.Fields {
		switch key {
		case "status":
			statusLabel, ok := value.(string)
			if !ok {
				return fmt.Errorf("campaign.updated status must be string")
			}
			status, err := parseCampaignStatus(statusLabel)
			if err != nil {
				return err
			}
			updated, err = campaign.TransitionCampaignStatus(updated, status, func() time.Time {
				return ensureTimestamp(evt.Timestamp)
			})
			if err != nil {
				return err
			}
		case "name":
			name, ok := value.(string)
			if !ok {
				return fmt.Errorf("campaign.updated name must be string")
			}
			name = strings.TrimSpace(name)
			if name == "" {
				return fmt.Errorf("campaign name is required")
			}
			updated.Name = name
		case "theme_prompt":
			prompt, ok := value.(string)
			if !ok {
				return fmt.Errorf("campaign.updated theme_prompt must be string")
			}
			updated.ThemePrompt = strings.TrimSpace(prompt)
		}
	}

	updatedAt := ensureTimestamp(evt.Timestamp)
	updated.UpdatedAt = updatedAt

	return a.Campaign.Put(ctx, updated)
}

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

func (a Applier) applyCharacterCreated(ctx context.Context, evt event.Event) error {
	if a.Character == nil {
		return fmt.Errorf("character store is not configured")
	}
	if a.Campaign == nil {
		return fmt.Errorf("campaign store is not configured")
	}
	characterID := strings.TrimSpace(evt.EntityID)
	if characterID == "" {
		return fmt.Errorf("character id is required")
	}
	if strings.TrimSpace(evt.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}

	var payload event.CharacterCreatedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode character.created payload: %w", err)
	}

	kind, err := parseCharacterKind(payload.Kind)
	if err != nil {
		return err
	}

	input := character.CreateCharacterInput{
		CampaignID: evt.CampaignID,
		Name:       payload.Name,
		Kind:       kind,
		Notes:      payload.Notes,
	}
	normalized, err := character.NormalizeCreateCharacterInput(input)
	if err != nil {
		return err
	}

	createdAt := ensureTimestamp(evt.Timestamp)
	ch := character.Character{
		ID:         characterID,
		CampaignID: normalized.CampaignID,
		Name:       normalized.Name,
		Kind:       normalized.Kind,
		Notes:      normalized.Notes,
		CreatedAt:  createdAt,
		UpdatedAt:  createdAt,
	}
	if err := a.Character.PutCharacter(ctx, ch); err != nil {
		return err
	}

	campaignRecord, err := a.Campaign.Get(ctx, evt.CampaignID)
	if err != nil {
		return err
	}
	campaignRecord.CharacterCount++
	campaignRecord.UpdatedAt = createdAt

	return a.Campaign.Put(ctx, campaignRecord)
}

func (a Applier) applyCharacterUpdated(ctx context.Context, evt event.Event) error {
	if a.Character == nil {
		return fmt.Errorf("character store is not configured")
	}
	if a.Campaign == nil {
		return fmt.Errorf("campaign store is not configured")
	}
	if strings.TrimSpace(evt.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	characterID := strings.TrimSpace(evt.EntityID)
	if characterID == "" {
		return fmt.Errorf("character id is required")
	}

	var payload event.CharacterUpdatedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode character.updated payload: %w", err)
	}
	if len(payload.Fields) == 0 {
		return nil
	}

	current, err := a.Character.GetCharacter(ctx, evt.CampaignID, characterID)
	if err != nil {
		return err
	}

	updated := current
	for key, value := range payload.Fields {
		switch key {
		case "name":
			name, ok := value.(string)
			if !ok {
				return fmt.Errorf("character.updated name must be string")
			}
			name = strings.TrimSpace(name)
			if name == "" {
				return fmt.Errorf("character name is required")
			}
			updated.Name = name
		case "kind":
			kindLabel, ok := value.(string)
			if !ok {
				return fmt.Errorf("character.updated kind must be string")
			}
			kind, err := parseCharacterKind(kindLabel)
			if err != nil {
				return err
			}
			updated.Kind = kind
		case "notes":
			notes, ok := value.(string)
			if !ok {
				return fmt.Errorf("character.updated notes must be string")
			}
			updated.Notes = strings.TrimSpace(notes)
		case "participant_id":
			participantID, ok := value.(string)
			if !ok {
				return fmt.Errorf("character.updated participant_id must be string")
			}
			updated.ParticipantID = strings.TrimSpace(participantID)
		}
	}

	updatedAt := ensureTimestamp(evt.Timestamp)
	updated.UpdatedAt = updatedAt

	if err := a.Character.PutCharacter(ctx, updated); err != nil {
		return err
	}

	campaignRecord, err := a.Campaign.Get(ctx, evt.CampaignID)
	if err != nil {
		return err
	}
	campaignRecord.UpdatedAt = updatedAt

	return a.Campaign.Put(ctx, campaignRecord)
}

func (a Applier) applyCharacterDeleted(ctx context.Context, evt event.Event) error {
	if a.Character == nil {
		return fmt.Errorf("character store is not configured")
	}
	if a.Campaign == nil {
		return fmt.Errorf("campaign store is not configured")
	}
	if strings.TrimSpace(evt.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	characterID := strings.TrimSpace(evt.EntityID)
	if characterID == "" {
		return fmt.Errorf("character id is required")
	}

	if err := a.Character.DeleteCharacter(ctx, evt.CampaignID, characterID); err != nil {
		return err
	}

	campaignRecord, err := a.Campaign.Get(ctx, evt.CampaignID)
	if err != nil {
		return err
	}
	if campaignRecord.CharacterCount > 0 {
		campaignRecord.CharacterCount--
	}
	updatedAt := ensureTimestamp(evt.Timestamp)
	campaignRecord.UpdatedAt = updatedAt

	return a.Campaign.Put(ctx, campaignRecord)
}

func (a Applier) applyProfileUpdated(ctx context.Context, evt event.Event) error {
	if a.Daggerheart == nil {
		return fmt.Errorf("daggerheart store is not configured")
	}
	if strings.TrimSpace(evt.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	characterID := strings.TrimSpace(evt.EntityID)
	if characterID == "" {
		return fmt.Errorf("character id is required")
	}

	var payload event.ProfileUpdatedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode character.profile_updated payload: %w", err)
	}
	if payload.SystemProfile == nil {
		return nil
	}
	profileData, ok := payload.SystemProfile["daggerheart"]
	if !ok {
		return nil
	}
	rawProfile, err := json.Marshal(profileData)
	if err != nil {
		return fmt.Errorf("marshal daggerheart profile payload: %w", err)
	}
	var dhProfile daggerheartProfilePayload
	if err := json.Unmarshal(rawProfile, &dhProfile); err != nil {
		return fmt.Errorf("decode daggerheart profile payload: %w", err)
	}
	experiences := make([]daggerheart.Experience, 0, len(dhProfile.Experiences))
	for _, experience := range dhProfile.Experiences {
		experiences = append(experiences, daggerheart.Experience{
			Name:     experience.Name,
			Modifier: experience.Modifier,
		})
	}
	level := dhProfile.Level
	if level == 0 {
		level = daggerheart.PCLevelDefault
	}
	if err := daggerheart.ValidateProfile(
		level,
		dhProfile.HpMax,
		dhProfile.StressMax,
		dhProfile.Evasion,
		dhProfile.MajorThreshold,
		dhProfile.SevereThreshold,
		dhProfile.Proficiency,
		dhProfile.ArmorScore,
		dhProfile.ArmorMax,
		daggerheart.Traits{
			Agility:   dhProfile.Agility,
			Strength:  dhProfile.Strength,
			Finesse:   dhProfile.Finesse,
			Instinct:  dhProfile.Instinct,
			Presence:  dhProfile.Presence,
			Knowledge: dhProfile.Knowledge,
		},
		experiences,
	); err != nil {
		return fmt.Errorf("validate daggerheart profile payload: %w", err)
	}

	experienceStorage := make([]storage.DaggerheartExperience, 0, len(dhProfile.Experiences))
	for _, experience := range dhProfile.Experiences {
		experienceStorage = append(experienceStorage, storage.DaggerheartExperience{
			Name:     experience.Name,
			Modifier: experience.Modifier,
		})
	}
	return a.Daggerheart.PutDaggerheartCharacterProfile(ctx, storage.DaggerheartCharacterProfile{
		CampaignID:      evt.CampaignID,
		CharacterID:     characterID,
		Level:           level,
		HpMax:           dhProfile.HpMax,
		StressMax:       dhProfile.StressMax,
		Evasion:         dhProfile.Evasion,
		MajorThreshold:  dhProfile.MajorThreshold,
		SevereThreshold: dhProfile.SevereThreshold,
		Proficiency:     dhProfile.Proficiency,
		ArmorScore:      dhProfile.ArmorScore,
		ArmorMax:        dhProfile.ArmorMax,
		Experiences:     experienceStorage,
		Agility:         dhProfile.Agility,
		Strength:        dhProfile.Strength,
		Finesse:         dhProfile.Finesse,
		Instinct:        dhProfile.Instinct,
		Presence:        dhProfile.Presence,
		Knowledge:       dhProfile.Knowledge,
	})
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

func (a Applier) applySessionStarted(ctx context.Context, evt event.Event) error {
	if a.Session == nil {
		return fmt.Errorf("session store is not configured")
	}
	if strings.TrimSpace(evt.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	var payload event.SessionStartedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode session.started payload: %w", err)
	}
	sessionID := strings.TrimSpace(payload.SessionID)
	if sessionID == "" {
		sessionID = strings.TrimSpace(evt.EntityID)
	}
	if sessionID == "" {
		return fmt.Errorf("session id is required")
	}
	startedAt := ensureTimestamp(evt.Timestamp)
	return a.Session.PutSession(ctx, session.Session{
		ID:         sessionID,
		CampaignID: evt.CampaignID,
		Name:       strings.TrimSpace(payload.SessionName),
		Status:     session.SessionStatusActive,
		StartedAt:  startedAt,
		UpdatedAt:  startedAt,
	})
}

func (a Applier) applySessionEnded(ctx context.Context, evt event.Event) error {
	if a.Session == nil {
		return fmt.Errorf("session store is not configured")
	}
	if strings.TrimSpace(evt.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	var payload event.SessionEndedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode session.ended payload: %w", err)
	}
	sessionID := strings.TrimSpace(payload.SessionID)
	if sessionID == "" {
		sessionID = strings.TrimSpace(evt.EntityID)
	}
	if sessionID == "" {
		return fmt.Errorf("session id is required")
	}
	_, _, err := a.Session.EndSession(ctx, evt.CampaignID, sessionID, ensureTimestamp(evt.Timestamp))
	return err
}

func parseGameSystem(value string) (commonv1.GameSystem, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED, fmt.Errorf("game system is required")
	}
	if system, ok := commonv1.GameSystem_value[trimmed]; ok {
		return commonv1.GameSystem(system), nil
	}
	upper := strings.ToUpper(trimmed)
	if system, ok := commonv1.GameSystem_value["GAME_SYSTEM_"+upper]; ok {
		return commonv1.GameSystem(system), nil
	}
	return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED, fmt.Errorf("unknown game system: %s", trimmed)
}

func (a Applier) applySystemEvent(ctx context.Context, evt event.Event) error {
	if a.Adapters == nil {
		return fmt.Errorf("system adapters are not configured")
	}
	if strings.TrimSpace(evt.SystemID) == "" {
		return fmt.Errorf("system_id is required for system events")
	}
	gameSystem, err := parseGameSystem(evt.SystemID)
	if err != nil {
		return err
	}
	adapter := a.Adapters.Get(gameSystem, evt.SystemVersion)
	if adapter == nil {
		return fmt.Errorf("system adapter not found for %s (%s)", evt.SystemID, evt.SystemVersion)
	}
	return adapter.ApplyEvent(ctx, evt)
}

func parseCampaignStatus(value string) (campaign.CampaignStatus, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return campaign.CampaignStatusUnspecified, fmt.Errorf("campaign status is required")
	}
	upper := strings.ToUpper(trimmed)
	switch upper {
	case "DRAFT", "CAMPAIGN_STATUS_DRAFT":
		return campaign.CampaignStatusDraft, nil
	case "ACTIVE", "CAMPAIGN_STATUS_ACTIVE":
		return campaign.CampaignStatusActive, nil
	case "COMPLETED", "CAMPAIGN_STATUS_COMPLETED":
		return campaign.CampaignStatusCompleted, nil
	case "ARCHIVED", "CAMPAIGN_STATUS_ARCHIVED":
		return campaign.CampaignStatusArchived, nil
	default:
		return campaign.CampaignStatusUnspecified, fmt.Errorf("unknown campaign status: %s", trimmed)
	}
}

func parseGmMode(value string) (campaign.GmMode, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return campaign.GmModeUnspecified, fmt.Errorf("gm mode is required")
	}
	upper := strings.ToUpper(trimmed)
	switch upper {
	case "HUMAN", "GM_MODE_HUMAN":
		return campaign.GmModeHuman, nil
	case "AI", "GM_MODE_AI":
		return campaign.GmModeAI, nil
	case "HYBRID", "GM_MODE_HYBRID":
		return campaign.GmModeHybrid, nil
	default:
		return campaign.GmModeUnspecified, fmt.Errorf("unknown gm mode: %s", trimmed)
	}
}

func parseParticipantRole(value string) (participant.ParticipantRole, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return participant.ParticipantRoleUnspecified, fmt.Errorf("participant role is required")
	}
	upper := strings.ToUpper(trimmed)
	switch upper {
	case "GM":
		return participant.ParticipantRoleGM, nil
	case "PLAYER":
		return participant.ParticipantRolePlayer, nil
	default:
		return participant.ParticipantRoleUnspecified, fmt.Errorf("unknown participant role: %s", trimmed)
	}
}

func parseParticipantController(value string) (participant.Controller, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return participant.ControllerUnspecified, fmt.Errorf("participant controller is required")
	}
	upper := strings.ToUpper(trimmed)
	switch upper {
	case "HUMAN", "CONTROLLER_HUMAN":
		return participant.ControllerHuman, nil
	case "AI", "CONTROLLER_AI":
		return participant.ControllerAI, nil
	default:
		return participant.ControllerUnspecified, fmt.Errorf("unknown participant controller: %s", trimmed)
	}
}

func parseCampaignAccess(value string) (participant.CampaignAccess, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return participant.CampaignAccessUnspecified, fmt.Errorf("campaign access is required")
	}
	upper := strings.ToUpper(trimmed)
	switch upper {
	case "MEMBER", "CAMPAIGN_ACCESS_MEMBER":
		return participant.CampaignAccessMember, nil
	case "MANAGER", "CAMPAIGN_ACCESS_MANAGER":
		return participant.CampaignAccessManager, nil
	case "OWNER", "CAMPAIGN_ACCESS_OWNER":
		return participant.CampaignAccessOwner, nil
	default:
		return participant.CampaignAccessUnspecified, fmt.Errorf("unknown campaign access: %s", trimmed)
	}
}

func parseInviteStatus(value string) (invite.Status, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return invite.StatusUnspecified, fmt.Errorf("invite status is required")
	}
	status := invite.StatusFromLabel(trimmed)
	if status == invite.StatusUnspecified {
		return invite.StatusUnspecified, fmt.Errorf("unknown invite status: %s", trimmed)
	}
	return status, nil
}

func parseCharacterKind(value string) (character.CharacterKind, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return character.CharacterKindUnspecified, fmt.Errorf("character kind is required")
	}
	upper := strings.ToUpper(trimmed)
	switch upper {
	case "PC", "CHARACTER_KIND_PC":
		return character.CharacterKindPC, nil
	case "NPC", "CHARACTER_KIND_NPC":
		return character.CharacterKindNPC, nil
	default:
		return character.CharacterKindUnspecified, fmt.Errorf("unknown character kind: %s", trimmed)
	}
}

func ensureTimestamp(ts time.Time) time.Time {
	if ts.IsZero() {
		return time.Now().UTC()
	}
	return ts.UTC()
}

type daggerheartProfilePayload struct {
	Level           int                            `json:"level"`
	HpMax           int                            `json:"hp_max"`
	StressMax       int                            `json:"stress_max"`
	Evasion         int                            `json:"evasion"`
	MajorThreshold  int                            `json:"major_threshold"`
	SevereThreshold int                            `json:"severe_threshold"`
	Proficiency     int                            `json:"proficiency"`
	ArmorScore      int                            `json:"armor_score"`
	ArmorMax        int                            `json:"armor_max"`
	Experiences     []daggerheartExperiencePayload `json:"experiences"`
	Agility         int                            `json:"agility"`
	Strength        int                            `json:"strength"`
	Finesse         int                            `json:"finesse"`
	Instinct        int                            `json:"instinct"`
	Presence        int                            `json:"presence"`
	Knowledge       int                            `json:"knowledge"`
}

type daggerheartExperiencePayload struct {
	Name     string `json:"name"`
	Modifier int    `json:"modifier"`
}
