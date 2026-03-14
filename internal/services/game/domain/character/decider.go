package character

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

const (
	CommandTypeCreate command.Type = "character.create"
	CommandTypeUpdate command.Type = "character.update"
	CommandTypeDelete command.Type = "character.delete"
	EventTypeCreated  event.Type   = "character.created"
	EventTypeUpdated  event.Type   = "character.updated"
	EventTypeDeleted  event.Type   = "character.deleted"

	rejectionCodeCharacterAlreadyExists      = "CHARACTER_ALREADY_EXISTS"
	rejectionCodeCharacterIDRequired         = "CHARACTER_ID_REQUIRED"
	rejectionCodeCharacterNameEmpty          = "CHARACTER_NAME_EMPTY"
	rejectionCodeCharacterKindInvalid        = "CHARACTER_KIND_INVALID"
	rejectionCodeCharacterAvatarSetInvalid   = "CHARACTER_INVALID_AVATAR_SET"
	rejectionCodeCharacterAvatarAssetInvalid = "CHARACTER_INVALID_AVATAR_ASSET"
	rejectionCodeCharacterNotCreated         = "CHARACTER_NOT_CREATED"
	rejectionCodeCharacterUpdateEmpty        = "CHARACTER_UPDATE_EMPTY"
	rejectionCodeCharacterUpdateFieldInvalid = "CHARACTER_UPDATE_FIELD_INVALID"
	rejectionCodeCharacterAliasesInvalid     = "CHARACTER_ALIASES_INVALID"
	rejectionCodeCharacterOwnerParticipantID = "CHARACTER_OWNER_PARTICIPANT_ID_REQUIRED"
)

// Decide returns the decision for a character command against current state.
//
// Character changes are intentionally event-driven so ownership and profile edits
// can be replayed and projected consistently across tools and clients.
func Decide(state State, cmd command.Command, now func() time.Time) command.Decision {
	now = command.NowFunc(now)

	switch cmd.Type {
	case CommandTypeCreate:
		return decideCreate(state, cmd, now)

	case CommandTypeUpdate:
		if !state.Created || state.Deleted {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCharacterNotCreated,
				Message: "character not created",
			})
		}
		var payload UpdatePayload
		if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
			return command.Reject(command.Rejection{
				Code:    command.RejectionCodePayloadDecodeFailed,
				Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
			})
		}
		characterID := strings.TrimSpace(payload.CharacterID.String())
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
			case "pronouns":
				normalizedFields[key] = strings.TrimSpace(value)
			case "aliases":
				normalizedAliases, err := normalizeAliasesField(value)
				if err != nil {
					return command.Reject(command.Rejection{
						Code:    rejectionCodeCharacterAliasesInvalid,
						Message: "character aliases are invalid",
					})
				}
				aliasesJSON, _ := json.Marshal(normalizedAliases)
				normalizedFields[key] = string(aliasesJSON)
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
		normalizedPayload := UpdatePayload{CharacterID: ids.CharacterID(characterID), Fields: normalizedFields}
		payloadJSON, _ := json.Marshal(normalizedPayload)
		evt := command.NewEvent(cmd, EventTypeUpdated, "character", characterID, payloadJSON, now().UTC())

		return command.Accept(evt)

	case CommandTypeDelete:
		if !state.Created || state.Deleted {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCharacterNotCreated,
				Message: "character not created",
			})
		}
		var payload DeletePayload
		if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
			return command.Reject(command.Rejection{
				Code:    command.RejectionCodePayloadDecodeFailed,
				Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
			})
		}
		characterID := strings.TrimSpace(payload.CharacterID.String())
		if characterID == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCharacterIDRequired,
				Message: "character id is required",
			})
		}
		reason := strings.TrimSpace(payload.Reason)

		normalizedPayload := DeletePayload{CharacterID: ids.CharacterID(characterID), Reason: reason}
		payloadJSON, _ := json.Marshal(normalizedPayload)
		evt := command.NewEvent(cmd, EventTypeDeleted, "character", characterID, payloadJSON, now().UTC())

		return command.Accept(evt)

	default:
		return command.Reject(command.Rejection{
			Code:    command.RejectionCodeCommandTypeUnsupported,
			Message: fmt.Sprintf("command type %s is not supported by character decider", cmd.Type),
		})
	}
}

func decideCreate(state State, cmd command.Command, now func() time.Time) command.Decision {
	if state.Created {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeCharacterAlreadyExists,
			Message: "character already exists",
		})
	}
	var payload CreatePayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{
			Code:    command.RejectionCodePayloadDecodeFailed,
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}
	characterID := strings.TrimSpace(payload.CharacterID.String())
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
	pronouns := strings.TrimSpace(payload.Pronouns)
	participantID := strings.TrimSpace(payload.ParticipantID.String())
	ownerParticipantID := strings.TrimSpace(payload.OwnerParticipantID.String())
	aliases := normalizeAliases(payload.Aliases)
	avatarSetID, avatarAssetID, err := resolveCharacterAvatarSelection(
		characterID,
		payload.AvatarSetID,
		payload.AvatarAssetID,
	)
	if err != nil {
		return command.Reject(characterAvatarRejection(err))
	}

	normalizedPayload := CreatePayload{
		CharacterID:        ids.CharacterID(characterID),
		OwnerParticipantID: ids.ParticipantID(ownerParticipantID),
		ParticipantID:      ids.ParticipantID(participantID),
		Name:               name,
		Kind:               kind,
		Notes:              notes,
		AvatarSetID:        avatarSetID,
		AvatarAssetID:      avatarAssetID,
		Pronouns:           pronouns,
		Aliases:            aliases,
	}
	payloadJSON, _ := json.Marshal(normalizedPayload)
	evt := command.NewEvent(cmd, EventTypeCreated, "character", characterID, payloadJSON, now().UTC())
	return command.Accept(evt)
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
