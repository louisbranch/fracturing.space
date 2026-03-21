package character

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func decideUpdate(state State, cmd command.Command, now func() time.Time) command.Decision {
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
	return acceptCharacterEvent(cmd, now, EventTypeUpdated, characterID, normalizedPayload)
}
