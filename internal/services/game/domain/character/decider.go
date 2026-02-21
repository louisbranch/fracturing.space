package character

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

const (
	commandTypeCreate        command.Type = "character.create"
	commandTypeUpdate        command.Type = "character.update"
	commandTypeDelete        command.Type = "character.delete"
	commandTypeProfileUpdate command.Type = "character.profile_update"
	EventTypeCreated         event.Type   = "character.created"
	EventTypeUpdated         event.Type   = "character.updated"
	EventTypeDeleted         event.Type   = "character.deleted"
	EventTypeProfileUpdated  event.Type   = "character.profile_updated"

	rejectionCodeCharacterAlreadyExists      = "CHARACTER_ALREADY_EXISTS"
	rejectionCodeCharacterIDRequired         = "CHARACTER_ID_REQUIRED"
	rejectionCodeCharacterNameEmpty          = "CHARACTER_NAME_EMPTY"
	rejectionCodeCharacterKindInvalid        = "CHARACTER_KIND_INVALID"
	rejectionCodeCharacterAvatarSetInvalid   = "CHARACTER_INVALID_AVATAR_SET"
	rejectionCodeCharacterAvatarAssetInvalid = "CHARACTER_INVALID_AVATAR_ASSET"
	rejectionCodeCharacterNotCreated         = "CHARACTER_NOT_CREATED"
	rejectionCodeCharacterUpdateEmpty        = "CHARACTER_UPDATE_EMPTY"
	rejectionCodeCharacterUpdateFieldInvalid = "CHARACTER_UPDATE_FIELD_INVALID"
	rejectionCodeCharacterOwnerParticipantID = "CHARACTER_OWNER_PARTICIPANT_ID_REQUIRED"
)

// Decide returns the decision for a character command against current state.
//
// Character changes are intentionally event-driven so ownership and profile edits
// can be replayed and projected consistently across tools and clients.
func Decide(state State, cmd command.Command, now func() time.Time) command.Decision {
	if cmd.Type == commandTypeCreate {
		if state.Created {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCharacterAlreadyExists,
				Message: "character already exists",
			})
		}
		var payload CreatePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		characterID := strings.TrimSpace(payload.CharacterID)
		if characterID == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCharacterIDRequired,
				Message: "character id is required",
			})
		}
		name := strings.TrimSpace(payload.Name)
		if name == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCharacterNameEmpty,
				Message: "character name is required",
			})
		}
		kind, ok := normalizeCharacterKindLabel(payload.Kind)
		if !ok {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCharacterKindInvalid,
				Message: "character kind is invalid",
			})
		}
		notes := strings.TrimSpace(payload.Notes)
		ownerParticipantID := strings.TrimSpace(payload.OwnerParticipantID)
		avatarSetID, avatarAssetID, err := resolveCharacterAvatarSelection(
			characterID,
			payload.AvatarSetID,
			payload.AvatarAssetID,
		)
		if err != nil {
			return command.Reject(characterAvatarRejection(err))
		}
		if now == nil {
			now = time.Now
		}

		normalizedPayload := CreatePayload{
			CharacterID:        characterID,
			OwnerParticipantID: ownerParticipantID,
			Name:               name,
			Kind:               kind,
			Notes:              notes,
			AvatarSetID:        avatarSetID,
			AvatarAssetID:      avatarAssetID,
		}
		payloadJSON, _ := json.Marshal(normalizedPayload)
		evt := command.NewEvent(cmd, EventTypeCreated, "character", characterID, payloadJSON, now().UTC())

		return command.Accept(evt)
	}

	if cmd.Type == commandTypeUpdate {
		if !state.Created || state.Deleted {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCharacterNotCreated,
				Message: "character not created",
			})
		}
		var payload UpdatePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		characterID := strings.TrimSpace(payload.CharacterID)
		if characterID == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCharacterIDRequired,
				Message: "character id is required",
			})
		}
		if len(payload.Fields) == 0 {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCharacterUpdateEmpty,
				Message: "character update requires fields",
			})
		}
		rawAvatarSetID, avatarSetProvided := payload.Fields["avatar_set_id"]
		rawAvatarAssetID, avatarAssetProvided := payload.Fields["avatar_asset_id"]
		normalizedFields := make(map[string]string, len(payload.Fields))
		for key, value := range payload.Fields {
			switch key {
			case "name":
				trimmed := strings.TrimSpace(value)
				if trimmed == "" {
					return command.Reject(command.Rejection{
						Code:    rejectionCodeCharacterNameEmpty,
						Message: "character name is required",
					})
				}
				normalizedFields[key] = trimmed
			case "kind":
				kind, ok := normalizeCharacterKindLabel(value)
				if !ok {
					return command.Reject(command.Rejection{
						Code:    rejectionCodeCharacterKindInvalid,
						Message: "character kind is invalid",
					})
				}
				normalizedFields[key] = kind
			case "notes":
				normalizedFields[key] = strings.TrimSpace(value)
			case "participant_id":
				normalizedFields[key] = strings.TrimSpace(value)
			case "owner_participant_id":
				trimmed := strings.TrimSpace(value)
				if trimmed == "" {
					return command.Reject(command.Rejection{
						Code:    rejectionCodeCharacterOwnerParticipantID,
						Message: "owner participant id is required",
					})
				}
				normalizedFields[key] = trimmed
			case "avatar_set_id":
			case "avatar_asset_id":
			default:
				return command.Reject(command.Rejection{
					Code:    rejectionCodeCharacterUpdateFieldInvalid,
					Message: "character update field is invalid",
				})
			}
		}
		if avatarSetProvided || avatarAssetProvided {
			avatarSetInput := strings.TrimSpace(state.AvatarSetID)
			if avatarSetProvided {
				avatarSetInput = rawAvatarSetID
			}

			avatarAssetInput := strings.TrimSpace(state.AvatarAssetID)
			if avatarAssetProvided {
				avatarAssetInput = rawAvatarAssetID
			} else if avatarSetProvided {
				avatarAssetInput = ""
			}

			resolvedSetID, resolvedAssetID, err := resolveCharacterAvatarSelection(
				characterID,
				avatarSetInput,
				avatarAssetInput,
			)
			if err != nil {
				return command.Reject(characterAvatarRejection(err))
			}
			if avatarSetProvided {
				normalizedFields["avatar_set_id"] = resolvedSetID
			}
			if avatarAssetProvided || avatarSetProvided {
				normalizedFields["avatar_asset_id"] = resolvedAssetID
			}
		}
		if now == nil {
			now = time.Now
		}

		normalizedPayload := UpdatePayload{CharacterID: characterID, Fields: normalizedFields}
		payloadJSON, _ := json.Marshal(normalizedPayload)
		evt := command.NewEvent(cmd, EventTypeUpdated, "character", characterID, payloadJSON, now().UTC())

		return command.Accept(evt)
	}

	if cmd.Type == commandTypeDelete {
		if !state.Created || state.Deleted {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCharacterNotCreated,
				Message: "character not created",
			})
		}
		var payload DeletePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		characterID := strings.TrimSpace(payload.CharacterID)
		if characterID == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCharacterIDRequired,
				Message: "character id is required",
			})
		}
		reason := strings.TrimSpace(payload.Reason)
		if now == nil {
			now = time.Now
		}

		normalizedPayload := DeletePayload{CharacterID: characterID, Reason: reason}
		payloadJSON, _ := json.Marshal(normalizedPayload)
		evt := command.NewEvent(cmd, EventTypeDeleted, "character", characterID, payloadJSON, now().UTC())

		return command.Accept(evt)
	}

	if cmd.Type == commandTypeProfileUpdate {
		if !state.Created || state.Deleted {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCharacterNotCreated,
				Message: "character not created",
			})
		}
		var payload ProfileUpdatePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		characterID := strings.TrimSpace(payload.CharacterID)
		if characterID == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCharacterIDRequired,
				Message: "character id is required",
			})
		}
		if now == nil {
			now = time.Now
		}

		normalizedPayload := ProfileUpdatePayload{
			CharacterID:   characterID,
			SystemProfile: payload.SystemProfile,
		}
		payloadJSON, _ := json.Marshal(normalizedPayload)
		evt := command.NewEvent(cmd, EventTypeProfileUpdated, "character", characterID, payloadJSON, now().UTC())

		return command.Accept(evt)
	}

	return command.Decision{}
}

// normalizeCharacterKindLabel returns a canonical character kind label.
//
// Character kinds flow into character-sheet and system-specific behavior, so this
// normalization prevents mismatched kind values from bifurcating state.
func normalizeCharacterKindLabel(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}
	upper := strings.ToUpper(trimmed)
	switch upper {
	case "PC", "CHARACTER_KIND_PC":
		return "pc", true
	case "NPC", "CHARACTER_KIND_NPC":
		return "npc", true
	default:
		return "", false
	}
}
