package projection

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// ProjectionHandledTypes returns the core event types handled by the projection
// layer. The list is derived from the handler registry map so there is a single
// source of truth for which event types have projection handlers.
func ProjectionHandledTypes() []event.Type {
	return registeredHandlerTypes()
}

// Apply routes domain events into denormalized read-model stores.
//
// The projection layer is the reason projections remain current for APIs and
// query use-cases: every event that changes campaign/world state in the domain
// gets mirrored here according to projection semantics.
func (a Applier) Apply(ctx context.Context, evt event.Event) error {
	if a.Events != nil {
		resolved := a.Events.Resolve(evt.Type)
		evt.Type = resolved
		// Skip audit-only and replay-only events — they do not affect read-model
		// state and must not reach the default error case.  The aggregate applier
		// has a similar guard; adding it here makes the projection applier
		// self-guarding.
		if def, ok := a.Events.Definition(resolved); ok && (def.Intent == event.IntentAuditOnly || def.Intent == event.IntentReplayOnly) {
			return nil
		}
	}
	if err := a.routeEvent(ctx, evt); err != nil {
		return err
	}
	if a.Watermarks != nil && evt.Seq > 0 && strings.TrimSpace(evt.CampaignID) != "" {
		if err := a.Watermarks.SaveProjectionWatermark(ctx, storage.ProjectionWatermark{
			CampaignID: evt.CampaignID,
			AppliedSeq: evt.Seq,
			UpdatedAt:  time.Now().UTC(),
		}); err != nil {
			return fmt.Errorf("save projection watermark: %w", err)
		}
	}
	return nil
}

// routeEvent dispatches a single event to the appropriate projection handler
// using the handler registry map. Core event types are looked up in the
// registry; events with a non-empty SystemID fall through to the system adapter
// path; anything else is rejected.
func (a Applier) routeEvent(ctx context.Context, evt event.Event) error {
	if h, ok := handlers[evt.Type]; ok {
		if err := a.validatePreconditions(h, evt); err != nil {
			return err
		}
		return h.apply(a, ctx, evt)
	}
	if strings.TrimSpace(evt.SystemID) != "" {
		return a.applySystemEvent(ctx, evt)
	}
	return fmt.Errorf("unhandled projection event type: %s", evt.Type)
}

func (a Applier) applyCampaignCreated(ctx context.Context, evt event.Event) error {
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

	createdAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
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
		CoverAssetID:     strings.TrimSpace(payload.CoverAssetID),
		CoverSetID:       strings.TrimSpace(payload.CoverSetID),
		CreatedAt:        createdAt,
		UpdatedAt:        createdAt,
	})
}

func (a Applier) applyCampaignUpdated(ctx context.Context, evt event.Event) error {
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
			statusTS, tsErr := ensureTimestamp(evt.Timestamp)
			if tsErr != nil {
				return tsErr
			}
			updated, err = applyCampaignStatusTransition(updated, status, statusTS)
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
		case "cover_asset_id":
			updated.CoverAssetID = strings.TrimSpace(value)
		case "cover_set_id":
			updated.CoverSetID = strings.TrimSpace(value)
		}
	}

	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	updated.UpdatedAt = updatedAt

	return a.Campaign.Put(ctx, updated)
}

func (a Applier) applyCampaignForked(ctx context.Context, evt event.Event) error {
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
	participantID := strings.TrimSpace(evt.EntityID)

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

func (a Applier) applyParticipantUpdated(ctx context.Context, evt event.Event) error {
	participantID := strings.TrimSpace(evt.EntityID)

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

func (a Applier) applyParticipantBound(ctx context.Context, evt event.Event) error {
	participantID := strings.TrimSpace(evt.EntityID)

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

func (a Applier) applyParticipantUnbound(ctx context.Context, evt event.Event) error {
	participantID := strings.TrimSpace(evt.EntityID)

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

func (a Applier) applySeatReassigned(ctx context.Context, evt event.Event) error {
	participantID := strings.TrimSpace(evt.EntityID)

	var payload participant.SeatReassignPayload
	if err := decodePayload(evt.PayloadJSON, &payload, "participant.seat_reassigned"); err != nil {
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

func (a Applier) applyCharacterCreated(ctx context.Context, evt event.Event) error {
	characterID := strings.TrimSpace(evt.EntityID)

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

	createdAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	ch := storage.CharacterRecord{
		ID:            characterID,
		CampaignID:    strings.TrimSpace(evt.CampaignID),
		Name:          name,
		Kind:          kind,
		Notes:         strings.TrimSpace(payload.Notes),
		AvatarSetID:   strings.TrimSpace(payload.AvatarSetID),
		AvatarAssetID: strings.TrimSpace(payload.AvatarAssetID),
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt,
	}
	if err := a.Character.PutCharacter(ctx, ch); err != nil {
		return err
	}

	campaignRecord, err := a.Campaign.Get(ctx, evt.CampaignID)
	if err != nil {
		return err
	}
	cCount, err := a.Character.CountCharacters(ctx, evt.CampaignID)
	if err != nil {
		return err
	}
	campaignRecord.CharacterCount = cCount
	campaignRecord.UpdatedAt = createdAt

	return a.Campaign.Put(ctx, campaignRecord)
}

func (a Applier) applyCharacterUpdated(ctx context.Context, evt event.Event) error {
	characterID := strings.TrimSpace(evt.EntityID)

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
	characterID := strings.TrimSpace(evt.EntityID)

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
	cCount, err := a.Character.CountCharacters(ctx, evt.CampaignID)
	if err != nil {
		return err
	}
	campaignRecord.CharacterCount = cCount
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	campaignRecord.UpdatedAt = updatedAt

	return a.Campaign.Put(ctx, campaignRecord)
}

func (a Applier) applyCharacterProfileUpdated(ctx context.Context, evt event.Event) error {
	characterID := strings.TrimSpace(evt.EntityID)

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

	for systemName, profileData := range payload.SystemProfile {
		gameSystem, err := parseGameSystem(systemName)
		if err != nil {
			// Skip unrecognized system names for forward compatibility.
			continue
		}
		adapter := a.Adapters.Get(gameSystem, "")
		if adapter == nil {
			// Skip systems without a registered adapter; a future replay
			// will pick them up once the adapter is configured.
			continue
		}
		profileAdapter, ok := adapter.(systems.ProfileAdapter)
		if !ok {
			continue
		}
		rawProfile, err := json.Marshal(profileData)
		if err != nil {
			return fmt.Errorf("marshal %s profile payload: %w", systemName, err)
		}
		if err := profileAdapter.ApplyProfile(ctx, evt.CampaignID, characterID, rawProfile); err != nil {
			return err
		}
	}
	return nil
}

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

func (a Applier) applySessionStarted(ctx context.Context, evt event.Event) error {
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
	startedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
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
	endedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	_, _, err = a.Session.EndSession(ctx, evt.CampaignID, sessionID, endedAt)
	return err
}

func (a Applier) applySessionGateOpened(ctx context.Context, evt event.Event) error {
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
	createdAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
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
	resolvedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	gate.Status = session.GateStatusResolved
	gate.ResolvedAt = &resolvedAt
	gate.ResolvedByActorType = string(evt.ActorType)
	gate.ResolvedByActorID = evt.ActorID
	gate.ResolutionJSON = resolutionJSON
	return a.SessionGate.PutSessionGate(ctx, gate)
}

func (a Applier) applySessionGateAbandoned(ctx context.Context, evt event.Event) error {
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
	resolvedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	gate.Status = session.GateStatusAbandoned
	gate.ResolvedAt = &resolvedAt
	gate.ResolvedByActorType = string(evt.ActorType)
	gate.ResolvedByActorID = evt.ActorID
	gate.ResolutionJSON = resolutionJSON
	return a.SessionGate.PutSessionGate(ctx, gate)
}

func (a Applier) applySessionSpotlightSet(ctx context.Context, evt event.Event) error {
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

	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	return a.SessionSpotlight.PutSessionSpotlight(ctx, storage.SessionSpotlight{
		CampaignID:         evt.CampaignID,
		SessionID:          evt.SessionID,
		SpotlightType:      spotlightType,
		CharacterID:        strings.TrimSpace(payload.CharacterID),
		UpdatedAt:          updatedAt,
		UpdatedByActorType: string(evt.ActorType),
		UpdatedByActorID:   evt.ActorID,
	})
}

func (a Applier) applySessionSpotlightCleared(ctx context.Context, evt event.Event) error {
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
