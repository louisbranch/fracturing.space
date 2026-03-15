package participant

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func decideUpdate(state State, cmd command.Command, now func() time.Time) command.Decision {
	now = command.NowFunc(now)

	if rejection, ok := ensureParticipantActive(state); !ok {
		return command.Reject(rejection)
	}
	payload, err := decodeCommandPayload[UpdatePayload](cmd)
	if err != nil {
		return command.Reject(command.Rejection{
			Code:    command.RejectionCodePayloadDecodeFailed,
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}
	participantID := strings.TrimSpace(payload.ParticipantID.String())
	if participantID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeParticipantIDRequired,
			Message: "participant id is required",
		})
	}
	if len(payload.Fields) == 0 {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeParticipantUpdateEmpty,
			Message: "participant update requires fields",
		})
	}

	rawAvatarSetID, avatarSetProvided := payload.Fields["avatar_set_id"]
	rawAvatarAssetID, avatarAssetProvided := payload.Fields["avatar_asset_id"]
	rawUserID, userIDProvided := payload.Fields["user_id"]
	normalizedFields := make(map[string]string, len(payload.Fields))
	for key, value := range payload.Fields {
		switch key {
		case "user_id":
			normalizedFields[key] = strings.TrimSpace(value)
		case "name":
			trimmed := strings.TrimSpace(value)
			if trimmed == "" {
				return command.Reject(command.Rejection{
					Code:    rejectionCodeParticipantNameEmpty,
					Message: "name is required",
				})
			}
			normalizedFields[key] = trimmed
		case "role":
			normalizedRole, ok := normalizeRoleLabel(value)
			if !ok {
				return command.Reject(command.Rejection{
					Code:    rejectionCodeParticipantRoleInvalid,
					Message: "participant role is invalid",
				})
			}
			normalizedFields[key] = normalizedRole
		case "controller":
			normalizedController, ok := normalizeControllerLabel(value)
			if !ok {
				return command.Reject(command.Rejection{
					Code:    rejectionCodeParticipantControllerInvalid,
					Message: "participant controller is invalid",
				})
			}
			normalizedFields[key] = normalizedController
		case "campaign_access":
			normalizedAccess, ok := normalizeCampaignAccessLabel(value)
			if !ok {
				return command.Reject(command.Rejection{
					Code:    rejectionCodeParticipantAccessInvalid,
					Message: "campaign access is invalid",
				})
			}
			normalizedFields[key] = normalizedAccess
		case "avatar_set_id":
		case "avatar_asset_id":
		case "pronouns":
			normalizedFields[key] = strings.TrimSpace(value)
		default:
			return command.Reject(command.Rejection{
				Code:    rejectionCodeParticipantUpdateFieldInvalid,
				Message: "participant update field is invalid",
			})
		}
	}
	if avatarSetProvided || avatarAssetProvided || userIDProvided {
		avatarUserID := strings.TrimSpace(string(state.UserID))
		if userIDProvided {
			avatarUserID = strings.TrimSpace(rawUserID)
		}

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

		resolvedSetID, resolvedAssetID, err := resolveParticipantAvatarSelection(
			participantID,
			avatarUserID,
			avatarSetInput,
			avatarAssetInput,
		)
		if err != nil {
			return command.Reject(participantAvatarRejection(err))
		}
		if avatarSetProvided || userIDProvided {
			normalizedFields["avatar_set_id"] = resolvedSetID
		}
		if avatarAssetProvided || avatarSetProvided || userIDProvided {
			normalizedFields["avatar_asset_id"] = resolvedAssetID
		}
	}
	effectiveUserID := strings.TrimSpace(string(state.UserID))
	if value, ok := normalizedFields["user_id"]; ok {
		effectiveUserID = strings.TrimSpace(value)
	}
	effectiveRole := strings.TrimSpace(string(state.Role))
	if value, ok := normalizedFields["role"]; ok {
		effectiveRole = strings.TrimSpace(value)
	}
	effectiveController := strings.TrimSpace(string(state.Controller))
	if value, ok := normalizedFields["controller"]; ok {
		effectiveController = strings.TrimSpace(value)
	}
	effectiveAccess := strings.TrimSpace(string(state.CampaignAccess))
	if value, ok := normalizedFields["campaign_access"]; ok {
		effectiveAccess = strings.TrimSpace(value)
	}
	if rejection, ok := validateAISeatInvariant(effectiveUserID, effectiveRole, effectiveController, effectiveAccess); !ok {
		return command.Reject(rejection)
	}

	normalizedPayload := UpdatePayload{ParticipantID: ids.ParticipantID(participantID), Fields: normalizedFields}
	payloadJSON, _ := json.Marshal(normalizedPayload)
	evt := command.NewEvent(cmd, EventTypeUpdated, "participant", participantID, payloadJSON, now().UTC())
	return command.Accept(evt)
}
