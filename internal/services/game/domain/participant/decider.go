package participant

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

const (
	commandTypeJoin               command.Type = "participant.join"
	commandTypeUpdate             command.Type = "participant.update"
	commandTypeLeave              command.Type = "participant.leave"
	commandTypeBind               command.Type = "participant.bind"
	commandTypeUnbind             command.Type = "participant.unbind"
	commandTypeSeatReassign       command.Type = "participant.seat.reassign"
	commandTypeSeatReassignLegacy command.Type = "seat.reassign"
	EventTypeJoined               event.Type   = "participant.joined"
	EventTypeUpdated              event.Type   = "participant.updated"
	EventTypeLeft                 event.Type   = "participant.left"
	EventTypeBound                event.Type   = "participant.bound"
	EventTypeUnbound              event.Type   = "participant.unbound"
	EventTypeSeatReassigned       event.Type   = "participant.seat_reassigned"
	EventTypeSeatReassignedLegacy event.Type   = "seat.reassigned"

	rejectionCodeParticipantAlreadyJoined      = "PARTICIPANT_ALREADY_JOINED"
	rejectionCodeParticipantNotJoined          = "PARTICIPANT_NOT_JOINED"
	rejectionCodeParticipantIDRequired         = "PARTICIPANT_ID_REQUIRED"
	rejectionCodeParticipantNameEmpty          = "PARTICIPANT_NAME_EMPTY"
	rejectionCodeParticipantRoleInvalid        = "PARTICIPANT_INVALID_ROLE"
	rejectionCodeParticipantControllerInvalid  = "PARTICIPANT_INVALID_CONTROLLER"
	rejectionCodeParticipantAccessInvalid      = "PARTICIPANT_INVALID_CAMPAIGN_ACCESS"
	rejectionCodeParticipantAvatarSetInvalid   = "PARTICIPANT_INVALID_AVATAR_SET"
	rejectionCodeParticipantAvatarAssetInvalid = "PARTICIPANT_INVALID_AVATAR_ASSET"
	rejectionCodeParticipantUpdateEmpty        = "PARTICIPANT_UPDATE_EMPTY"
	rejectionCodeParticipantUpdateFieldInvalid = "PARTICIPANT_UPDATE_FIELD_INVALID"
	rejectionCodeParticipantUserIDRequired     = "PARTICIPANT_USER_ID_REQUIRED"
	rejectionCodeParticipantUserIDMismatch     = "PARTICIPANT_USER_ID_MISMATCH"
)

// Decide returns the decision for a participant command against current state.
//
// Participant commands define membership and authorization context. This decider keeps
// that context explicit by emitting identity/role/capability changes as immutable
// events rather than mutating shared storage directly.
func Decide(state State, cmd command.Command, now func() time.Time) command.Decision {
	if cmd.Type == commandTypeJoin {
		if state.Joined {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeParticipantAlreadyJoined,
				Message: "participant already joined",
			})
		}
		var payload JoinPayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)

		participantID := strings.TrimSpace(payload.ParticipantID)
		if participantID == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeParticipantIDRequired,
				Message: "participant id is required",
			})
		}
		userID := strings.TrimSpace(payload.UserID)
		name := strings.TrimSpace(payload.Name)
		if name == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeParticipantNameEmpty,
				Message: "name is required",
			})
		}
		role, ok := normalizeRoleLabel(payload.Role)
		if !ok {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeParticipantRoleInvalid,
				Message: "participant role is required",
			})
		}
		controller, ok := normalizeControllerLabel(payload.Controller)
		if !ok {
			if strings.TrimSpace(payload.Controller) != "" {
				return command.Reject(command.Rejection{
					Code:    rejectionCodeParticipantControllerInvalid,
					Message: "participant controller is invalid",
				})
			}
			controller = "human"
		}
		access, ok := normalizeCampaignAccessLabel(payload.CampaignAccess)
		if !ok {
			if strings.TrimSpace(payload.CampaignAccess) != "" {
				return command.Reject(command.Rejection{
					Code:    rejectionCodeParticipantAccessInvalid,
					Message: "campaign access is invalid",
				})
			}
			access = "member"
		}
		avatarSetID, avatarAssetID, err := resolveParticipantAvatarSelection(
			participantID,
			userID,
			payload.AvatarSetID,
			payload.AvatarAssetID,
		)
		if err != nil {
			return command.Reject(participantAvatarRejection(err))
		}
		if now == nil {
			now = time.Now
		}

		normalizedPayload := JoinPayload{
			ParticipantID:  participantID,
			UserID:         userID,
			Name:           name,
			Role:           role,
			Controller:     controller,
			CampaignAccess: access,
			AvatarSetID:    avatarSetID,
			AvatarAssetID:  avatarAssetID,
		}
		payloadJSON, _ := json.Marshal(normalizedPayload)

		evt := command.NewEvent(cmd, EventTypeJoined, "participant", participantID, payloadJSON, now().UTC())

		return command.Accept(evt)
	}
	if cmd.Type == commandTypeUpdate {
		if !state.Joined || state.Left {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeParticipantNotJoined,
				Message: "participant not joined",
			})
		}
		var payload UpdatePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		participantID := strings.TrimSpace(payload.ParticipantID)
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
			default:
				return command.Reject(command.Rejection{
					Code:    rejectionCodeParticipantUpdateFieldInvalid,
					Message: "participant update field is invalid",
				})
			}
		}
		if avatarSetProvided || avatarAssetProvided {
			avatarUserID := strings.TrimSpace(state.UserID)
			if rawUserID, ok := normalizedFields["user_id"]; ok {
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

		normalizedPayload := UpdatePayload{ParticipantID: participantID, Fields: normalizedFields}
		payloadJSON, _ := json.Marshal(normalizedPayload)
		evt := command.NewEvent(cmd, EventTypeUpdated, "participant", participantID, payloadJSON, now().UTC())

		return command.Accept(evt)
	}

	if cmd.Type == commandTypeLeave {
		if !state.Joined || state.Left {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeParticipantNotJoined,
				Message: "participant not joined",
			})
		}
		var payload LeavePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		participantID := strings.TrimSpace(payload.ParticipantID)
		if participantID == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeParticipantIDRequired,
				Message: "participant id is required",
			})
		}
		reason := strings.TrimSpace(payload.Reason)
		if now == nil {
			now = time.Now
		}

		normalizedPayload := LeavePayload{ParticipantID: participantID, Reason: reason}
		payloadJSON, _ := json.Marshal(normalizedPayload)
		evt := command.NewEvent(cmd, EventTypeLeft, "participant", participantID, payloadJSON, now().UTC())

		return command.Accept(evt)
	}

	if cmd.Type == commandTypeBind {
		if !state.Joined || state.Left {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeParticipantNotJoined,
				Message: "participant not joined",
			})
		}
		var payload BindPayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		participantID := strings.TrimSpace(payload.ParticipantID)
		if participantID == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeParticipantIDRequired,
				Message: "participant id is required",
			})
		}
		userID := strings.TrimSpace(payload.UserID)
		if userID == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeParticipantUserIDRequired,
				Message: "user id is required",
			})
		}
		if now == nil {
			now = time.Now
		}

		normalizedPayload := BindPayload{ParticipantID: participantID, UserID: userID}
		payloadJSON, _ := json.Marshal(normalizedPayload)
		evt := command.NewEvent(cmd, EventTypeBound, "participant", participantID, payloadJSON, now().UTC())

		return command.Accept(evt)
	}

	if cmd.Type == commandTypeUnbind {
		if !state.Joined || state.Left {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeParticipantNotJoined,
				Message: "participant not joined",
			})
		}
		var payload UnbindPayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		participantID := strings.TrimSpace(payload.ParticipantID)
		if participantID == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeParticipantIDRequired,
				Message: "participant id is required",
			})
		}
		userID := strings.TrimSpace(payload.UserID)
		if userID != "" && userID != strings.TrimSpace(state.UserID) {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeParticipantUserIDMismatch,
				Message: "participant user id mismatch",
			})
		}
		reason := strings.TrimSpace(payload.Reason)
		if now == nil {
			now = time.Now
		}

		normalizedPayload := UnbindPayload{ParticipantID: participantID, UserID: userID, Reason: reason}
		payloadJSON, _ := json.Marshal(normalizedPayload)
		evt := command.NewEvent(cmd, EventTypeUnbound, "participant", participantID, payloadJSON, now().UTC())

		return command.Accept(evt)
	}

	if cmd.Type == commandTypeSeatReassign || cmd.Type == commandTypeSeatReassignLegacy {
		if !state.Joined || state.Left {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeParticipantNotJoined,
				Message: "participant not joined",
			})
		}
		var payload SeatReassignPayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		participantID := strings.TrimSpace(payload.ParticipantID)
		if participantID == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeParticipantIDRequired,
				Message: "participant id is required",
			})
		}
		priorUserID := strings.TrimSpace(payload.PriorUserID)
		if priorUserID != "" && priorUserID != strings.TrimSpace(state.UserID) {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeParticipantUserIDMismatch,
				Message: "participant user id mismatch",
			})
		}
		userID := strings.TrimSpace(payload.UserID)
		if userID == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeParticipantUserIDRequired,
				Message: "user id is required",
			})
		}
		reason := strings.TrimSpace(payload.Reason)
		if now == nil {
			now = time.Now
		}

		normalizedPayload := SeatReassignPayload{
			ParticipantID: participantID,
			PriorUserID:   priorUserID,
			UserID:        userID,
			Reason:        reason,
		}
		payloadJSON, _ := json.Marshal(normalizedPayload)
		evt := command.NewEvent(cmd, EventTypeSeatReassigned, "participant", participantID, payloadJSON, now().UTC())

		return command.Accept(evt)
	}

	return command.Decision{}
}

// normalizeRoleLabel returns a canonical role label.
func normalizeRoleLabel(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}
	upper := strings.ToUpper(trimmed)
	switch upper {
	case "GM", "ROLE_GM", "PARTICIPANT_ROLE_GM":
		return "gm", true
	case "PLAYER", "ROLE_PLAYER", "PARTICIPANT_ROLE_PLAYER":
		return "player", true
	default:
		return "", false
	}
}

// normalizeControllerLabel returns a canonical controller label.
func normalizeControllerLabel(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}
	upper := strings.ToUpper(trimmed)
	switch upper {
	case "HUMAN", "CONTROLLER_HUMAN":
		return "human", true
	case "AI", "CONTROLLER_AI":
		return "ai", true
	default:
		return "", false
	}
}

// normalizeCampaignAccessLabel returns a canonical access label.
func normalizeCampaignAccessLabel(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}
	upper := strings.ToUpper(trimmed)
	switch upper {
	case "MEMBER", "CAMPAIGN_ACCESS_MEMBER":
		return "member", true
	case "MANAGER", "CAMPAIGN_ACCESS_MANAGER":
		return "manager", true
	case "OWNER", "CAMPAIGN_ACCESS_OWNER":
		return "owner", true
	default:
		return "", false
	}
}
