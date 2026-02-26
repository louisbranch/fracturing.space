package projection

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func (a Applier) applyCharacterCreated(ctx context.Context, evt event.Event, payload character.CreatePayload) error {
	characterID := strings.TrimSpace(evt.EntityID)

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
	ownerParticipantID := strings.TrimSpace(payload.OwnerParticipantID)
	if ownerParticipantID == "" {
		ownerParticipantID = strings.TrimSpace(evt.ActorID)
	}
	ch := storage.CharacterRecord{
		ID:                 characterID,
		CampaignID:         strings.TrimSpace(evt.CampaignID),
		OwnerParticipantID: ownerParticipantID,
		Name:               name,
		Kind:               kind,
		Notes:              strings.TrimSpace(payload.Notes),
		AvatarSetID:        strings.TrimSpace(payload.AvatarSetID),
		AvatarAssetID:      strings.TrimSpace(payload.AvatarAssetID),
		CreatedAt:          createdAt,
		UpdatedAt:          createdAt,
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

func (a Applier) applyCharacterUpdated(ctx context.Context, evt event.Event, payload character.UpdatePayload) error {
	characterID := strings.TrimSpace(evt.EntityID)

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
		case "owner_participant_id":
			updated.OwnerParticipantID = strings.TrimSpace(value)
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

func (a Applier) applyCharacterDeleted(ctx context.Context, evt event.Event, payload character.DeletePayload) error {
	characterID := strings.TrimSpace(evt.EntityID)

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

func (a Applier) applyCharacterProfileUpdated(ctx context.Context, evt event.Event, payload character.ProfileUpdatePayload) error {
	characterID := strings.TrimSpace(evt.EntityID)

	if payload.CharacterID != "" && strings.TrimSpace(payload.CharacterID) != characterID {
		return fmt.Errorf("character_id mismatch")
	}
	if payload.SystemProfile == nil {
		return nil
	}

	for systemName, profileData := range payload.SystemProfile {
		adapter, ok := a.Adapters.GetOptional(systemName, "")
		if !ok {
			// Skip systems without a registered adapter; a future replay
			// will pick them up once the adapter is configured.
			continue
		}
		profileAdapter, ok := adapter.(bridge.ProfileAdapter)
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
