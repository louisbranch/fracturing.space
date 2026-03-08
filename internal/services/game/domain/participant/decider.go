package participant

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
	CommandTypeJoin         command.Type = "participant.join"
	CommandTypeUpdate       command.Type = "participant.update"
	CommandTypeLeave        command.Type = "participant.leave"
	CommandTypeBind         command.Type = "participant.bind"
	CommandTypeUnbind       command.Type = "participant.unbind"
	CommandTypeSeatReassign command.Type = "participant.seat.reassign"
	EventTypeJoined         event.Type   = "participant.joined"
	EventTypeUpdated        event.Type   = "participant.updated"
	EventTypeLeft           event.Type   = "participant.left"
	EventTypeBound          event.Type   = "participant.bound"
	EventTypeUnbound        event.Type   = "participant.unbound"
	EventTypeSeatReassigned event.Type   = "participant.seat_reassigned"

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
	rejectionCodeParticipantAIRoleRequired     = "PARTICIPANT_AI_ROLE_REQUIRED"
	rejectionCodeParticipantAIAccessRequired   = "PARTICIPANT_AI_ACCESS_REQUIRED"
	rejectionCodeParticipantAIUserIDForbidden  = "PARTICIPANT_AI_USER_ID_FORBIDDEN"
	rejectionCodeParticipantAIIdentityLocked   = "PARTICIPANT_AI_IDENTITY_LOCKED"
)

type participantDecisionHandler func(State, command.Command, func() time.Time) command.Decision

var participantDecisionHandlers = map[command.Type]participantDecisionHandler{
	CommandTypeJoin:         decideJoin,
	CommandTypeUpdate:       decideUpdate,
	CommandTypeLeave:        decideLeave,
	CommandTypeBind:         decideBind,
	CommandTypeUnbind:       decideUnbind,
	CommandTypeSeatReassign: decideSeatReassign,
}

// Decide returns the decision for a participant command against current state.
//
// Participant commands define membership and authorization context. This decider keeps
// that context explicit by emitting identity/role/capability changes as immutable
// events rather than mutating shared storage directly.
func Decide(state State, cmd command.Command, now func() time.Time) command.Decision {
	handler, ok := participantDecisionHandlers[cmd.Type]
	if !ok {
		return command.Reject(command.Rejection{
			Code:    "COMMAND_TYPE_UNSUPPORTED",
			Message: fmt.Sprintf("command type %s is not supported by participant decider", cmd.Type),
		})
	}
	return handler(state, cmd, now)
}

func ensureParticipantActive(state State) (command.Rejection, bool) {
	if !state.Joined || state.Left {
		return command.Rejection{
			Code:    rejectionCodeParticipantNotJoined,
			Message: "participant not joined",
		}, false
	}
	return command.Rejection{}, true
}

func decodeCommandPayload[T any](cmd command.Command) (T, error) {
	var payload T
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return payload, err
	}
	return payload, nil
}

func decideJoin(state State, cmd command.Command, now func() time.Time) command.Decision {
	if state.Joined {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeParticipantAlreadyJoined,
			Message: "participant already joined",
		})
	}
	var payload JoinPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{
			Code:    "PAYLOAD_DECODE_FAILED",
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
	userID := strings.TrimSpace(payload.UserID.String())
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
	if rejection, ok := validateAISeatInvariant(userID, role, controller, access); !ok {
		return command.Reject(rejection)
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
	pronouns := strings.TrimSpace(payload.Pronouns)

	if now == nil {
		now = time.Now
	}

	normalizedPayload := JoinPayload{
		ParticipantID:  ids.ParticipantID(participantID),
		UserID:         ids.UserID(userID),
		Name:           name,
		Role:           role,
		Controller:     controller,
		CampaignAccess: access,
		AvatarSetID:    avatarSetID,
		AvatarAssetID:  avatarAssetID,
		Pronouns:       pronouns,
	}
	payloadJSON, _ := json.Marshal(normalizedPayload)

	evt := command.NewEvent(cmd, EventTypeJoined, "participant", participantID, payloadJSON, now().UTC())
	return command.Accept(evt)
}

func decideUpdate(state State, cmd command.Command, now func() time.Time) command.Decision {
	if !state.Joined || state.Left {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeParticipantNotJoined,
			Message: "participant not joined",
		})
	}
	var payload UpdatePayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{
			Code:    "PAYLOAD_DECODE_FAILED",
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
	if now == nil {
		now = time.Now
	}

	normalizedPayload := UpdatePayload{ParticipantID: ids.ParticipantID(participantID), Fields: normalizedFields}
	payloadJSON, _ := json.Marshal(normalizedPayload)
	evt := command.NewEvent(cmd, EventTypeUpdated, "participant", participantID, payloadJSON, now().UTC())
	return command.Accept(evt)
}

func decideLeave(state State, cmd command.Command, now func() time.Time) command.Decision {
	if !state.Joined || state.Left {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeParticipantNotJoined,
			Message: "participant not joined",
		})
	}
	var payload LeavePayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{
			Code:    "PAYLOAD_DECODE_FAILED",
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
	reason := strings.TrimSpace(payload.Reason)
	if now == nil {
		now = time.Now
	}

	normalizedPayload := LeavePayload{ParticipantID: ids.ParticipantID(participantID), Reason: reason}
	payloadJSON, _ := json.Marshal(normalizedPayload)
	evt := command.NewEvent(cmd, EventTypeLeft, "participant", participantID, payloadJSON, now().UTC())
	return command.Accept(evt)
}

func decideBind(state State, cmd command.Command, now func() time.Time) command.Decision {
	if rejection, ok := ensureParticipantActive(state); !ok {
		return command.Reject(rejection)
	}
	if isAIController(string(state.Controller)) {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeParticipantAIIdentityLocked,
			Message: "ai-controlled participants cannot change user identity bindings",
		})
	}
	payload, err := decodeCommandPayload[BindPayload](cmd)
	if err != nil {
		return command.Reject(command.Rejection{
			Code:    "PAYLOAD_DECODE_FAILED",
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
	userID := strings.TrimSpace(payload.UserID.String())
	if userID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeParticipantUserIDRequired,
			Message: "user id is required",
		})
	}
	if now == nil {
		now = time.Now
	}

	normalizedPayload := BindPayload{ParticipantID: ids.ParticipantID(participantID), UserID: ids.UserID(userID)}
	payloadJSON, _ := json.Marshal(normalizedPayload)
	evt := command.NewEvent(cmd, EventTypeBound, "participant", participantID, payloadJSON, now().UTC())
	return command.Accept(evt)
}

func decideUnbind(state State, cmd command.Command, now func() time.Time) command.Decision {
	if rejection, ok := ensureParticipantActive(state); !ok {
		return command.Reject(rejection)
	}
	if isAIController(string(state.Controller)) {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeParticipantAIIdentityLocked,
			Message: "ai-controlled participants cannot change user identity bindings",
		})
	}
	payload, err := decodeCommandPayload[UnbindPayload](cmd)
	if err != nil {
		return command.Reject(command.Rejection{
			Code:    "PAYLOAD_DECODE_FAILED",
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
	userID := strings.TrimSpace(payload.UserID.String())
	if userID != "" && userID != strings.TrimSpace(string(state.UserID)) {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeParticipantUserIDMismatch,
			Message: "participant user id mismatch",
		})
	}
	reason := strings.TrimSpace(payload.Reason)
	if now == nil {
		now = time.Now
	}

	normalizedPayload := UnbindPayload{ParticipantID: ids.ParticipantID(participantID), UserID: ids.UserID(userID), Reason: reason}
	payloadJSON, _ := json.Marshal(normalizedPayload)
	evt := command.NewEvent(cmd, EventTypeUnbound, "participant", participantID, payloadJSON, now().UTC())
	return command.Accept(evt)
}

func decideSeatReassign(state State, cmd command.Command, now func() time.Time) command.Decision {
	if rejection, ok := ensureParticipantActive(state); !ok {
		return command.Reject(rejection)
	}
	if isAIController(string(state.Controller)) {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeParticipantAIIdentityLocked,
			Message: "ai-controlled participants cannot change user identity bindings",
		})
	}
	payload, err := decodeCommandPayload[SeatReassignPayload](cmd)
	if err != nil {
		return command.Reject(command.Rejection{
			Code:    "PAYLOAD_DECODE_FAILED",
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
	priorUserID := strings.TrimSpace(payload.PriorUserID.String())
	if priorUserID != "" && priorUserID != strings.TrimSpace(string(state.UserID)) {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeParticipantUserIDMismatch,
			Message: "participant user id mismatch",
		})
	}
	userID := strings.TrimSpace(payload.UserID.String())
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
		ParticipantID: ids.ParticipantID(participantID),
		PriorUserID:   ids.UserID(priorUserID),
		UserID:        ids.UserID(userID),
		Reason:        reason,
	}
	payloadJSON, _ := json.Marshal(normalizedPayload)
	evt := command.NewEvent(cmd, EventTypeSeatReassigned, "participant", participantID, payloadJSON, now().UTC())
	return command.Accept(evt)
}

func validateAISeatInvariant(userID, role, controller, access string) (command.Rejection, bool) {
	normalizedController, ok := normalizeControllerLabel(controller)
	if !ok || normalizedController != "ai" {
		return command.Rejection{}, true
	}
	normalizedRole, ok := normalizeRoleLabel(role)
	if !ok || normalizedRole != "gm" {
		return command.Rejection{
			Code:    rejectionCodeParticipantAIRoleRequired,
			Message: "ai-controlled participants must use gm role",
		}, false
	}
	normalizedAccess, ok := normalizeCampaignAccessLabel(access)
	if !ok || normalizedAccess != "member" {
		return command.Rejection{
			Code:    rejectionCodeParticipantAIAccessRequired,
			Message: "ai-controlled participants must use member campaign access",
		}, false
	}
	if strings.TrimSpace(userID) != "" {
		return command.Rejection{
			Code:    rejectionCodeParticipantAIUserIDForbidden,
			Message: "ai-controlled participants must not have a user id",
		}, false
	}
	return command.Rejection{}, true
}

func isAIController(controller string) bool {
	normalized, ok := normalizeControllerLabel(controller)
	return ok && normalized == "ai"
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
