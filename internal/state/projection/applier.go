package projection

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/state/campaign"
	"github.com/louisbranch/fracturing.space/internal/state/character"
	"github.com/louisbranch/fracturing.space/internal/state/event"
	"github.com/louisbranch/fracturing.space/internal/state/participant"
	"github.com/louisbranch/fracturing.space/internal/storage"
)

// Applier applies event journal entries to projection stores.
type Applier struct {
	Campaign    storage.CampaignStore
	Character   storage.CharacterStore
	Control     storage.ControlDefaultStore
	Daggerheart storage.DaggerheartStore
	Participant storage.ParticipantStore
}

// Apply applies an event to projection stores.
func (a Applier) Apply(ctx context.Context, evt event.Event) error {
	switch evt.Type {
	case event.TypeCampaignCreated:
		return a.applyCampaignCreated(ctx, evt)
	case event.TypeCampaignStatusChanged:
		return a.applyCampaignStatusChanged(ctx, evt)
	case event.TypeParticipantJoined:
		return a.applyParticipantJoined(ctx, evt)
	case event.TypeParticipantUpdated:
		return a.applyParticipantUpdated(ctx, evt)
	case event.TypeParticipantLeft:
		return a.applyParticipantLeft(ctx, evt)
	case event.TypeCharacterCreated:
		return a.applyCharacterCreated(ctx, evt)
	case event.TypeCharacterUpdated:
		return a.applyCharacterUpdated(ctx, evt)
	case event.TypeCharacterDeleted:
		return a.applyCharacterDeleted(ctx, evt)
	case event.TypeControllerAssigned:
		return a.applyControllerAssigned(ctx, evt)
	case event.TypeProfileUpdated:
		return a.applyProfileUpdated(ctx, evt)
	case event.TypeCharacterStateChanged:
		return a.applyCharacterStateChanged(ctx, evt)
	case event.TypeGMFearChanged:
		return a.applyGMFearChanged(ctx, evt)
	default:
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
		LastActivityAt:   createdAt,
		UpdatedAt:        createdAt,
	}

	return a.Campaign.Put(ctx, c)
}

func (a Applier) applyCampaignStatusChanged(ctx context.Context, evt event.Event) error {
	if a.Campaign == nil {
		return fmt.Errorf("campaign store is not configured")
	}
	if strings.TrimSpace(evt.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	var payload event.CampaignStatusChangedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode campaign.status_changed payload: %w", err)
	}
	status, err := parseCampaignStatus(payload.ToStatus)
	if err != nil {
		return err
	}

	current, err := a.Campaign.Get(ctx, evt.CampaignID)
	if err != nil {
		return err
	}

	updated, err := campaign.TransitionCampaignStatus(current, status, func() time.Time {
		return ensureTimestamp(evt.Timestamp)
	})
	if err != nil {
		return err
	}
	updated.LastActivityAt = ensureTimestamp(evt.Timestamp)
	updated.UpdatedAt = ensureTimestamp(evt.Timestamp)

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

	input := participant.CreateParticipantInput{
		CampaignID:  evt.CampaignID,
		DisplayName: payload.DisplayName,
		Role:        role,
		Controller:  controller,
	}
	normalized, err := participant.NormalizeCreateParticipantInput(input)
	if err != nil {
		return err
	}

	createdAt := ensureTimestamp(evt.Timestamp)
	p := participant.Participant{
		ID:          participantID,
		CampaignID:  normalized.CampaignID,
		DisplayName: normalized.DisplayName,
		Role:        normalized.Role,
		Controller:  normalized.Controller,
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}
	if err := a.Participant.PutParticipant(ctx, p); err != nil {
		return err
	}

	campaignRecord, err := a.Campaign.Get(ctx, evt.CampaignID)
	if err != nil {
		return err
	}
	campaignRecord.ParticipantCount++
	campaignRecord.LastActivityAt = createdAt
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
	campaignRecord.LastActivityAt = updatedAt
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
	campaignRecord.LastActivityAt = updatedAt
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
	campaignRecord.LastActivityAt = createdAt
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
	campaignRecord.LastActivityAt = updatedAt
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
	campaignRecord.LastActivityAt = updatedAt
	campaignRecord.UpdatedAt = updatedAt

	return a.Campaign.Put(ctx, campaignRecord)
}

func (a Applier) applyControllerAssigned(ctx context.Context, evt event.Event) error {
	if a.Control == nil {
		return fmt.Errorf("control default store is not configured")
	}
	if strings.TrimSpace(evt.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	characterID := strings.TrimSpace(evt.EntityID)
	if characterID == "" {
		return fmt.Errorf("character id is required")
	}

	var payload event.ControllerAssignedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode character.controller_assigned payload: %w", err)
	}

	var controller character.CharacterController
	if payload.IsGM {
		controller = character.NewGmController()
	} else {
		ctrl, err := character.NewParticipantController(payload.ParticipantID)
		if err != nil {
			return err
		}
		controller = ctrl
	}
	if err := controller.Validate(); err != nil {
		return err
	}

	return a.Control.PutControlDefault(ctx, evt.CampaignID, characterID, controller)
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

	return a.Daggerheart.PutDaggerheartCharacterProfile(ctx, storage.DaggerheartCharacterProfile{
		CampaignID:      evt.CampaignID,
		CharacterID:     characterID,
		HpMax:           dhProfile.HpMax,
		StressMax:       dhProfile.StressMax,
		Evasion:         dhProfile.Evasion,
		MajorThreshold:  dhProfile.MajorThreshold,
		SevereThreshold: dhProfile.SevereThreshold,
		Agility:         dhProfile.Agility,
		Strength:        dhProfile.Strength,
		Finesse:         dhProfile.Finesse,
		Instinct:        dhProfile.Instinct,
		Presence:        dhProfile.Presence,
		Knowledge:       dhProfile.Knowledge,
	})
}

func (a Applier) applyCharacterStateChanged(ctx context.Context, evt event.Event) error {
	if a.Daggerheart == nil {
		return fmt.Errorf("daggerheart store is not configured")
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

	var payload event.CharacterStateChangedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode chronicle.character_state_changed payload: %w", err)
	}

	current, err := a.Daggerheart.GetDaggerheartCharacterState(ctx, evt.CampaignID, characterID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			current = storage.DaggerheartCharacterState{CampaignID: evt.CampaignID, CharacterID: characterID}
		} else {
			return err
		}
	}

	updated := current
	if payload.HpAfter != nil {
		updated.Hp = *payload.HpAfter
	}
	if dhState, ok := payload.SystemState["daggerheart"]; ok {
		stateMap, ok := dhState.(map[string]any)
		if !ok {
			return fmt.Errorf("chronicle.character_state_changed daggerheart state must be object")
		}
		if value, ok := stateMap["hope_after"]; ok {
			hope, err := parseSnapshotNumber(value, "hope_after")
			if err != nil {
				return err
			}
			updated.Hope = hope
		}
		if value, ok := stateMap["stress_after"]; ok {
			stress, err := parseSnapshotNumber(value, "stress_after")
			if err != nil {
				return err
			}
			updated.Stress = stress
		}
	}

	if err := a.Daggerheart.PutDaggerheartCharacterState(ctx, updated); err != nil {
		return err
	}

	campaignRecord, err := a.Campaign.Get(ctx, evt.CampaignID)
	if err != nil {
		return err
	}
	updatedAt := ensureTimestamp(evt.Timestamp)
	campaignRecord.LastActivityAt = updatedAt
	campaignRecord.UpdatedAt = updatedAt

	return a.Campaign.Put(ctx, campaignRecord)
}

func (a Applier) applyGMFearChanged(ctx context.Context, evt event.Event) error {
	if a.Daggerheart == nil {
		return fmt.Errorf("daggerheart store is not configured")
	}
	if a.Campaign == nil {
		return fmt.Errorf("campaign store is not configured")
	}
	if strings.TrimSpace(evt.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}

	var payload event.GMFearChangedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode chronicle.gm_fear_changed payload: %w", err)
	}

	if err := a.Daggerheart.PutDaggerheartSnapshot(ctx, storage.DaggerheartSnapshot{
		CampaignID: evt.CampaignID,
		GMFear:     payload.After,
	}); err != nil {
		return err
	}

	campaignRecord, err := a.Campaign.Get(ctx, evt.CampaignID)
	if err != nil {
		return err
	}
	updatedAt := ensureTimestamp(evt.Timestamp)
	campaignRecord.LastActivityAt = updatedAt
	campaignRecord.UpdatedAt = updatedAt

	return a.Campaign.Put(ctx, campaignRecord)
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

func parseSnapshotNumber(value any, field string) (int, error) {
	switch v := value.(type) {
	case float64:
		if v != math.Trunc(v) {
			return 0, fmt.Errorf("%s must be an integer", field)
		}
		return int(v), nil
	case float32:
		if v != float32(math.Trunc(float64(v))) {
			return 0, fmt.Errorf("%s must be an integer", field)
		}
		return int(v), nil
	case int:
		return v, nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case json.Number:
		parsed, err := v.Int64()
		if err != nil {
			return 0, fmt.Errorf("%s must be an integer", field)
		}
		return int(parsed), nil
	default:
		return 0, fmt.Errorf("%s must be a number", field)
	}
}

type daggerheartProfilePayload struct {
	HpMax           int `json:"hp_max"`
	StressMax       int `json:"stress_max"`
	Evasion         int `json:"evasion"`
	MajorThreshold  int `json:"major_threshold"`
	SevereThreshold int `json:"severe_threshold"`
	Agility         int `json:"agility"`
	Strength        int `json:"strength"`
	Finesse         int `json:"finesse"`
	Instinct        int `json:"instinct"`
	Presence        int `json:"presence"`
	Knowledge       int `json:"knowledge"`
}
