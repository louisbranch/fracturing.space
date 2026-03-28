package session

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func decideRecapRecord(state State, cmd command.Command, now func() time.Time) command.Decision {
	if !state.Started {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionNotStarted,
			Message: "session not started",
		})
	}

	var payload RecapRecordedPayload
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

	markdown := strings.TrimSpace(payload.Markdown)
	if markdown == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionRecapMarkdownRequired,
			Message: "session recap markdown is required",
		})
	}

	normalizedPayload := RecapRecordedPayload{
		SessionID: ids.SessionID(sessionID),
		Markdown:  markdown,
	}
	payloadJSON, _ := json.Marshal(normalizedPayload)
	evt := command.NewEvent(cmd, EventTypeRecapRecorded, "session", sessionID, payloadJSON, now().UTC())
	evt.SessionID = ids.SessionID(sessionID)

	return command.Accept(evt)
}
