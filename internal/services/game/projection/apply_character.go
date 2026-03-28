package projection

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func (a Applier) applyCharacterCreated(ctx context.Context, evt event.Event, payload character.CreatePayload) error {
	characterID := strings.TrimSpace(evt.EntityID)

	if payload.CharacterID != "" && strings.TrimSpace(payload.CharacterID.String()) != characterID {
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
	ownerParticipantID := strings.TrimSpace(payload.OwnerParticipantID.String())
	if ownerParticipantID == "" {
		ownerParticipantID = strings.TrimSpace(evt.ActorID)
	}
	ch := storage.CharacterRecord{
		ID:                 characterID,
		CampaignID:         string(evt.CampaignID),
		OwnerParticipantID: ownerParticipantID,
		Name:               name,
		Kind:               kind,
		Notes:              strings.TrimSpace(payload.Notes),
		AvatarSetID:        strings.TrimSpace(payload.AvatarSetID),
		AvatarAssetID:      strings.TrimSpace(payload.AvatarAssetID),
		Pronouns:           strings.TrimSpace(payload.Pronouns),
		Aliases:            normalizeProjectionAliases(payload.Aliases),
		CreatedAt:          createdAt,
		UpdatedAt:          createdAt,
	}
	if err := a.Character.PutCharacter(ctx, ch); err != nil {
		return err
	}

	campaignRecord, err := a.Campaign.Get(ctx, string(evt.CampaignID))
	if err != nil {
		return err
	}
	cCount, err := a.Character.CountCharacters(ctx, string(evt.CampaignID))
	if err != nil {
		return err
	}
	campaignRecord.CharacterCount = cCount
	campaignRecord.UpdatedAt = createdAt

	return a.Campaign.Put(ctx, campaignRecord)
}

func (a Applier) applyCharacterUpdated(ctx context.Context, evt event.Event, payload character.UpdatePayload) error {
	characterID := strings.TrimSpace(evt.EntityID)

	if payload.CharacterID != "" && strings.TrimSpace(payload.CharacterID.String()) != characterID {
		return fmt.Errorf("character_id mismatch")
	}
	if len(payload.Fields) == 0 {
		return nil
	}

	current, err := a.Character.GetCharacter(ctx, string(evt.CampaignID), characterID)
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
		case "owner_participant_id":
			updated.OwnerParticipantID = strings.TrimSpace(value)
		case "avatar_set_id":
			updated.AvatarSetID = strings.TrimSpace(value)
		case "avatar_asset_id":
			updated.AvatarAssetID = strings.TrimSpace(value)
		case "pronouns":
			updated.Pronouns = strings.TrimSpace(value)
		case "aliases":
			aliases, err := parseProjectionAliases(value)
			if err != nil {
				return err
			}
			updated.Aliases = aliases
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

	campaignRecord, err := a.Campaign.Get(ctx, string(evt.CampaignID))
	if err != nil {
		return err
	}
	campaignRecord.UpdatedAt = updatedAt

	return a.Campaign.Put(ctx, campaignRecord)
}

func (a Applier) applyCharacterDeleted(ctx context.Context, evt event.Event, payload character.DeletePayload) error {
	characterID := strings.TrimSpace(evt.EntityID)

	if payload.CharacterID != "" && strings.TrimSpace(payload.CharacterID.String()) != characterID {
		return fmt.Errorf("character_id mismatch")
	}

	if err := a.Character.DeleteCharacter(ctx, string(evt.CampaignID), characterID); err != nil {
		return err
	}

	campaignRecord, err := a.Campaign.Get(ctx, string(evt.CampaignID))
	if err != nil {
		return err
	}
	cCount, err := a.Character.CountCharacters(ctx, string(evt.CampaignID))
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

func normalizeProjectionAliases(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func parseProjectionAliases(value string) ([]string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	var raw []string
	if err := json.Unmarshal([]byte(value), &raw); err != nil {
		return nil, fmt.Errorf("character aliases are invalid")
	}
	return normalizeProjectionAliases(raw), nil
}
