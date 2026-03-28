package session

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func decideStart(state State, cmd command.Command, now func() time.Time) command.Decision {
	if state.Started {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionAlreadyStarted,
			Message: "session already started",
		})
	}
	var payload StartPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{Code: command.RejectionCodePayloadDecodeFailed, Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
	}
	sessionID := strings.TrimSpace(payload.SessionID.String())
	if sessionID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionIDRequired,
			Message: "session id is required",
		})
	}
	sessionName := strings.TrimSpace(payload.SessionName)
	if sessionName == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionNameRequired,
			Message: "session name is required",
		})
	}
	normalizedControllers := make([]CharacterControllerAssignment, 0, len(payload.CharacterControllers))
	seenCharacters := make(map[string]struct{}, len(payload.CharacterControllers))
	for _, assignment := range payload.CharacterControllers {
		characterID := strings.TrimSpace(assignment.CharacterID.String())
		if characterID == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeSessionCharacterRequired,
				Message: "character id is required",
			})
		}
		participantID := strings.TrimSpace(assignment.ParticipantID.String())
		if participantID == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeSessionCharacterControllerRequired,
				Message: "character controller participant id is required",
			})
		}
		if _, exists := seenCharacters[characterID]; exists {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeSessionCharacterControllerUnchanged,
				Message: "character controller assignments must be unique by character",
			})
		}
		seenCharacters[characterID] = struct{}{}
		normalizedControllers = append(normalizedControllers, CharacterControllerAssignment{
			CharacterID:   ids.CharacterID(characterID),
			ParticipantID: ids.ParticipantID(participantID),
		})
	}

	normalizedPayload := StartPayload{
		SessionID:            ids.SessionID(sessionID),
		SessionName:          sessionName,
		CharacterControllers: normalizedControllers,
	}
	payloadJSON, _ := json.Marshal(normalizedPayload)
	evt := command.NewEvent(cmd, EventTypeStarted, "session", sessionID, payloadJSON, now().UTC())
	evt.SessionID = ids.SessionID(sessionID)

	return command.Accept(evt)
}

func decideEnd(state State, cmd command.Command, now func() time.Time) command.Decision {
	if !state.Started {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionNotStarted,
			Message: "session not started",
		})
	}
	var payload EndPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{Code: command.RejectionCodePayloadDecodeFailed, Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
	}
	sessionID := strings.TrimSpace(payload.SessionID.String())
	if sessionID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionIDRequired,
			Message: "session id is required",
		})
	}

	normalizedPayload := EndPayload{SessionID: ids.SessionID(sessionID)}
	payloadJSON, _ := json.Marshal(normalizedPayload)
	evt := command.NewEvent(cmd, EventTypeEnded, "session", sessionID, payloadJSON, now().UTC())
	evt.SessionID = ids.SessionID(sessionID)

	return command.Accept(evt)
}
