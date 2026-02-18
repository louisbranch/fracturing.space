package projection

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// Apply routes domain events into denormalized read-model stores.
//
// The projection layer is the reason projections remain current for APIs and
// query use-cases: every event that changes campaign/world state in the domain
// gets mirrored here according to projection semantics.
func (a Applier) Apply(ctx context.Context, evt event.Event) error {
	switch evt.Type {
	case event.Type("campaign.created"):
		return a.applyCampaignCreated(ctx, evt)
	case event.Type("campaign.updated"):
		return a.applyCampaignUpdated(ctx, evt)
	case event.Type("campaign.forked"):
		return a.applyCampaignForked(ctx, evt)
	case event.Type("participant.joined"):
		return a.applyParticipantJoined(ctx, evt)
	case event.Type("participant.updated"):
		return a.applyParticipantUpdated(ctx, evt)
	case event.Type("participant.left"):
		return a.applyParticipantLeft(ctx, evt)
	case event.Type("participant.bound"):
		return a.applyParticipantBound(ctx, evt)
	case event.Type("participant.unbound"):
		return a.applyParticipantUnbound(ctx, evt)
	case event.Type("seat.reassigned"), event.Type("participant.seat_reassigned"):
		return a.applySeatReassigned(ctx, evt)
	case event.Type("character.created"):
		return a.applyCharacterCreated(ctx, evt)
	case event.Type("character.updated"):
		return a.applyCharacterUpdated(ctx, evt)
	case event.Type("character.deleted"):
		return a.applyCharacterDeleted(ctx, evt)
	case event.Type("character.profile_updated"):
		return a.applyCharacterProfileUpdated(ctx, evt)
	case event.Type("invite.created"):
		return a.applyInviteCreated(ctx, evt)
	case event.Type("invite.claimed"):
		return a.applyInviteClaimed(ctx, evt)
	case event.Type("invite.revoked"):
		return a.applyInviteRevoked(ctx, evt)
	case event.Type("invite.updated"):
		return a.applyInviteUpdated(ctx, evt)
	case event.Type("session.started"):
		return a.applySessionStarted(ctx, evt)
	case event.Type("session.ended"):
		return a.applySessionEnded(ctx, evt)
	case event.Type("session.gate_opened"):
		return a.applySessionGateOpened(ctx, evt)
	case event.Type("session.gate_resolved"):
		return a.applySessionGateResolved(ctx, evt)
	case event.Type("session.gate_abandoned"):
		return a.applySessionGateAbandoned(ctx, evt)
	case event.Type("session.spotlight_set"):
		return a.applySessionSpotlightSet(ctx, evt)
	case event.Type("session.spotlight_cleared"):
		return a.applySessionSpotlightCleared(ctx, evt)
	default:
		if strings.TrimSpace(evt.SystemID) != "" {
			return a.applySystemEvent(ctx, evt)
		}
		return fmt.Errorf("unhandled projection event type: %s", evt.Type)
	}
}

func (a Applier) applyCampaignCreated(ctx context.Context, evt event.Event) error {
	if a.Campaign == nil {
		return fmt.Errorf("campaign store is not configured")
	}
	if strings.TrimSpace(evt.EntityID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	var payload campaign.CreatePayload
	if err := decodePayload(evt.PayloadJSON, &payload, "campaign.created"); err != nil {
		return err
	}

	system, err := parseGameSystem(payload.GameSystem)
	if err != nil {
		return err
	}
	gmMode, err := parseGmMode(payload.GmMode)
	if err != nil {
		return err
	}
	intent := parseCampaignIntent(payload.Intent)
	accessPolicy := parseCampaignAccessPolicy(payload.AccessPolicy)
	locale := platformi18n.DefaultLocale()
	if parsed, ok := platformi18n.ParseLocale(payload.Locale); ok {
		locale = parsed
	}

	input := campaign.CreateInput{
		Name:         payload.Name,
		Locale:       locale,
		System:       system,
		GmMode:       gmMode,
		Intent:       intent,
		AccessPolicy: accessPolicy,
		ThemePrompt:  payload.ThemePrompt,
	}
	normalized, err := campaign.NormalizeCreateInput(input)
	if err != nil {
		return err
	}

	createdAt := ensureTimestamp(evt.Timestamp)
	return a.Campaign.Put(ctx, storage.CampaignRecord{
		ID:               evt.EntityID,
		Name:             normalized.Name,
		Locale:           normalized.Locale,
		System:           normalized.System,
		Status:           campaign.StatusDraft,
		GmMode:           normalized.GmMode,
		Intent:           normalized.Intent,
		AccessPolicy:     normalized.AccessPolicy,
		ParticipantCount: 0,
		CharacterCount:   0,
		ThemePrompt:      normalized.ThemePrompt,
		CreatedAt:        createdAt,
		UpdatedAt:        createdAt,
	})
}

func (a Applier) applyCampaignUpdated(ctx context.Context, evt event.Event) error {
	if a.Campaign == nil {
		return fmt.Errorf("campaign store is not configured")
	}
	if strings.TrimSpace(evt.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	var payload campaign.UpdatePayload
	if err := decodePayload(evt.PayloadJSON, &payload, "campaign.updated"); err != nil {
		return err
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
			status, err := parseCampaignStatus(value)
			if err != nil {
				return err
			}
			updated, err = applyCampaignStatusTransition(updated, status, ensureTimestamp(evt.Timestamp))
			if err != nil {
				return err
			}
		case "name":
			name := strings.TrimSpace(value)
			if name == "" {
				return fmt.Errorf("campaign name is required")
			}
			updated.Name = name
		case "theme_prompt":
			updated.ThemePrompt = strings.TrimSpace(value)
		}
	}

	updatedAt := ensureTimestamp(evt.Timestamp)
	updated.UpdatedAt = updatedAt

	return a.Campaign.Put(ctx, updated)
}

func (a Applier) applyCampaignForked(ctx context.Context, evt event.Event) error {
	if a.CampaignFork == nil {
		return fmt.Errorf("campaign fork store is not configured")
	}
	if strings.TrimSpace(evt.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	var payload campaign.ForkPayload
	if err := decodePayload(evt.PayloadJSON, &payload, "campaign.forked"); err != nil {
		return err
	}
	return a.CampaignFork.SetCampaignForkMetadata(ctx, evt.CampaignID, storage.ForkMetadata{
		ParentCampaignID: payload.ParentCampaignID,
		ForkEventSeq:     payload.ForkEventSeq,
		OriginCampaignID: payload.OriginCampaignID,
	})
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

	var payload participant.JoinPayload
	if err := decodePayload(evt.PayloadJSON, &payload, "participant.joined"); err != nil {
		return err
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
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		return fmt.Errorf("name is required")
	}
	userID := strings.TrimSpace(payload.UserID)

	createdAt := ensureTimestamp(evt.Timestamp)
	if err := a.Participant.PutParticipant(ctx, storage.ParticipantRecord{
		ID:             participantID,
		CampaignID:     strings.TrimSpace(evt.CampaignID),
		UserID:         userID,
		Name:           name,
		Role:           role,
		Controller:     controller,
		CampaignAccess: access,
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}); err != nil {
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

	var payload participant.UpdatePayload
	if err := decodePayload(evt.PayloadJSON, &payload, "participant.updated"); err != nil {
		return err
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

	var payload participant.BindPayload
	if err := decodePayload(evt.PayloadJSON, &payload, "participant.bound"); err != nil {
		return err
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

	var payload participant.UnbindPayload
	if err := decodePayload(evt.PayloadJSON, &payload, "participant.unbound"); err != nil {
		return err
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

	var payload participant.SeatReassignPayload
	payloadType := "seat.reassigned"
	if strings.TrimSpace(string(evt.Type)) == "participant.seat_reassigned" {
		payloadType = "participant.seat_reassigned"
	}
	if err := decodePayload(evt.PayloadJSON, &payload, payloadType); err != nil {
		return err
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

	var payload character.CreatePayload
	if err := decodePayload(evt.PayloadJSON, &payload, "character.created"); err != nil {
		return err
	}
	if payload.CharacterID != "" && strings.TrimSpace(payload.CharacterID) != characterID {
		return fmt.Errorf("character_id mismatch")
	}

	kind, err := parseCharacterKind(payload.Kind)
	if err != nil {
		return err
	}

	name := strings.TrimSpace(payload.Name)
	if name == "" {
		return fmt.Errorf("character name is required")
	}

	createdAt := ensureTimestamp(evt.Timestamp)
	ch := storage.CharacterRecord{
		ID:         characterID,
		CampaignID: strings.TrimSpace(evt.CampaignID),
		Name:       name,
		Kind:       kind,
		Notes:      strings.TrimSpace(payload.Notes),
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

	var payload character.UpdatePayload
	if err := decodePayload(evt.PayloadJSON, &payload, "character.updated"); err != nil {
		return err
	}
	if payload.CharacterID != "" && strings.TrimSpace(payload.CharacterID) != characterID {
		return fmt.Errorf("character_id mismatch")
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
			name := strings.TrimSpace(value)
			if name == "" {
				return fmt.Errorf("character name is required")
			}
			updated.Name = name
		case "kind":
			kind, err := parseCharacterKind(value)
			if err != nil {
				return err
			}
			updated.Kind = kind
		case "notes":
			updated.Notes = strings.TrimSpace(value)
		case "participant_id":
			updated.ParticipantID = strings.TrimSpace(value)
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

	var payload character.DeletePayload
	if err := decodePayload(evt.PayloadJSON, &payload, "character.deleted"); err != nil {
		return err
	}
	if payload.CharacterID != "" && strings.TrimSpace(payload.CharacterID) != characterID {
		return fmt.Errorf("character_id mismatch")
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

func (a Applier) applyCharacterProfileUpdated(ctx context.Context, evt event.Event) error {
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

	var payload character.ProfileUpdatePayload
	if err := decodePayload(evt.PayloadJSON, &payload, "character.profile_updated"); err != nil {
		return err
	}
	if payload.CharacterID != "" && strings.TrimSpace(payload.CharacterID) != characterID {
		return fmt.Errorf("character_id mismatch")
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
	experienceStorage := make([]storage.DaggerheartExperience, 0, len(dhProfile.Experiences))
	for _, experience := range dhProfile.Experiences {
		experienceStorage = append(experienceStorage, storage.DaggerheartExperience{
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

	createdAt := ensureTimestamp(evt.Timestamp)
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

	var payload invite.ClaimPayload
	if err := decodePayload(evt.PayloadJSON, &payload, "invite.claimed"); err != nil {
		return err
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

	var payload invite.RevokePayload
	if err := decodePayload(evt.PayloadJSON, &payload, "invite.revoked"); err != nil {
		return err
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
	var payload session.StartPayload
	if err := decodePayload(evt.PayloadJSON, &payload, "session.started"); err != nil {
		return err
	}
	sessionID := strings.TrimSpace(payload.SessionID)
	if sessionID == "" {
		sessionID = strings.TrimSpace(evt.EntityID)
	}
	if sessionID == "" {
		return fmt.Errorf("session id is required")
	}
	startedAt := ensureTimestamp(evt.Timestamp)
	return a.Session.PutSession(ctx, storage.SessionRecord{
		ID:         sessionID,
		CampaignID: evt.CampaignID,
		Name:       strings.TrimSpace(payload.SessionName),
		Status:     session.StatusActive,
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
	var payload session.EndPayload
	if err := decodePayload(evt.PayloadJSON, &payload, "session.ended"); err != nil {
		return err
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

func (a Applier) applySessionGateOpened(ctx context.Context, evt event.Event) error {
	if a.SessionGate == nil {
		return fmt.Errorf("session gate store is not configured")
	}
	if strings.TrimSpace(evt.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(evt.SessionID) == "" {
		return fmt.Errorf("session id is required")
	}
	var payload session.GateOpenedPayload
	if err := decodePayload(evt.PayloadJSON, &payload, "session.gate_opened"); err != nil {
		return err
	}
	gateID := strings.TrimSpace(payload.GateID)
	if gateID == "" {
		gateID = strings.TrimSpace(evt.EntityID)
	}
	if gateID == "" {
		return fmt.Errorf("gate id is required")
	}
	gateType, err := session.NormalizeGateType(payload.GateType)
	if err != nil {
		return err
	}
	reason := session.NormalizeGateReason(payload.Reason)
	metadataJSON, err := marshalOptionalMap(payload.Metadata)
	if err != nil {
		return fmt.Errorf("encode gate metadata: %w", err)
	}
	createdAt := ensureTimestamp(evt.Timestamp)
	return a.SessionGate.PutSessionGate(ctx, storage.SessionGate{
		CampaignID:         evt.CampaignID,
		SessionID:          evt.SessionID,
		GateID:             gateID,
		GateType:           gateType,
		Status:             session.GateStatusOpen,
		Reason:             reason,
		CreatedAt:          createdAt,
		CreatedByActorType: string(evt.ActorType),
		CreatedByActorID:   evt.ActorID,
		MetadataJSON:       metadataJSON,
	})
}

func (a Applier) applySessionGateResolved(ctx context.Context, evt event.Event) error {
	if a.SessionGate == nil {
		return fmt.Errorf("session gate store is not configured")
	}
	if strings.TrimSpace(evt.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(evt.SessionID) == "" {
		return fmt.Errorf("session id is required")
	}
	var payload session.GateResolvedPayload
	if err := decodePayload(evt.PayloadJSON, &payload, "session.gate_resolved"); err != nil {
		return err
	}
	gateID := strings.TrimSpace(payload.GateID)
	if gateID == "" {
		gateID = strings.TrimSpace(evt.EntityID)
	}
	if gateID == "" {
		return fmt.Errorf("gate id is required")
	}
	gate, err := a.SessionGate.GetSessionGate(ctx, evt.CampaignID, evt.SessionID, gateID)
	if err != nil {
		return fmt.Errorf("get session gate: %w", err)
	}
	resolutionJSON, err := marshalResolutionPayload(payload.Decision, payload.Resolution)
	if err != nil {
		return fmt.Errorf("encode gate resolution: %w", err)
	}
	resolvedAt := ensureTimestamp(evt.Timestamp)
	gate.Status = session.GateStatusResolved
	gate.ResolvedAt = &resolvedAt
	gate.ResolvedByActorType = string(evt.ActorType)
	gate.ResolvedByActorID = evt.ActorID
	gate.ResolutionJSON = resolutionJSON
	return a.SessionGate.PutSessionGate(ctx, gate)
}

func (a Applier) applySessionGateAbandoned(ctx context.Context, evt event.Event) error {
	if a.SessionGate == nil {
		return fmt.Errorf("session gate store is not configured")
	}
	if strings.TrimSpace(evt.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(evt.SessionID) == "" {
		return fmt.Errorf("session id is required")
	}
	var payload session.GateAbandonedPayload
	if err := decodePayload(evt.PayloadJSON, &payload, "session.gate_abandoned"); err != nil {
		return err
	}
	gateID := strings.TrimSpace(payload.GateID)
	if gateID == "" {
		gateID = strings.TrimSpace(evt.EntityID)
	}
	if gateID == "" {
		return fmt.Errorf("gate id is required")
	}
	gate, err := a.SessionGate.GetSessionGate(ctx, evt.CampaignID, evt.SessionID, gateID)
	if err != nil {
		return fmt.Errorf("get session gate: %w", err)
	}
	resolutionJSON, err := marshalResolutionPayload("abandoned", map[string]any{"reason": session.NormalizeGateReason(payload.Reason)})
	if err != nil {
		return fmt.Errorf("encode gate resolution: %w", err)
	}
	resolvedAt := ensureTimestamp(evt.Timestamp)
	gate.Status = session.GateStatusAbandoned
	gate.ResolvedAt = &resolvedAt
	gate.ResolvedByActorType = string(evt.ActorType)
	gate.ResolvedByActorID = evt.ActorID
	gate.ResolutionJSON = resolutionJSON
	return a.SessionGate.PutSessionGate(ctx, gate)
}

func (a Applier) applySessionSpotlightSet(ctx context.Context, evt event.Event) error {
	if a.SessionSpotlight == nil {
		return fmt.Errorf("session spotlight store is not configured")
	}
	if strings.TrimSpace(evt.SessionID) == "" {
		return fmt.Errorf("session id is required")
	}
	var payload session.SpotlightSetPayload
	if err := decodePayload(evt.PayloadJSON, &payload, "session.spotlight_set"); err != nil {
		return err
	}
	spotlightType, err := session.NormalizeSpotlightType(payload.SpotlightType)
	if err != nil {
		return err
	}
	if err := session.ValidateSpotlightTarget(spotlightType, payload.CharacterID); err != nil {
		return err
	}

	return a.SessionSpotlight.PutSessionSpotlight(ctx, storage.SessionSpotlight{
		CampaignID:         evt.CampaignID,
		SessionID:          evt.SessionID,
		SpotlightType:      spotlightType,
		CharacterID:        strings.TrimSpace(payload.CharacterID),
		UpdatedAt:          ensureTimestamp(evt.Timestamp),
		UpdatedByActorType: string(evt.ActorType),
		UpdatedByActorID:   evt.ActorID,
	})
}

func (a Applier) applySessionSpotlightCleared(ctx context.Context, evt event.Event) error {
	if a.SessionSpotlight == nil {
		return fmt.Errorf("session spotlight store is not configured")
	}
	if strings.TrimSpace(evt.SessionID) == "" {
		return fmt.Errorf("session id is required")
	}
	return a.SessionSpotlight.ClearSessionSpotlight(ctx, evt.CampaignID, evt.SessionID)
}

// decodePayload is a guarded bridge between event envelopes and in-memory domain
// payload types, preserving a clear failure message when replay/apply input is
// malformed.
func decodePayload(payload []byte, target any, name string) error {
	if err := json.Unmarshal(payload, target); err != nil {
		return fmt.Errorf("decode %s payload: %w", name, err)
	}
	return nil
}
