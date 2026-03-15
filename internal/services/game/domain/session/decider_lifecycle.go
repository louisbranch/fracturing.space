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

	normalizedPayload := StartPayload{SessionID: ids.SessionID(sessionID), SessionName: sessionName}
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
