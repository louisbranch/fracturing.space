package projection

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

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
