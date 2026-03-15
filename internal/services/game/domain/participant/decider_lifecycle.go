package participant

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func decideLeave(state State, cmd command.Command, now func() time.Time) command.Decision {
	now = command.NowFunc(now)

	if rejection, ok := ensureParticipantActive(state); !ok {
		return command.Reject(rejection)
	}
	payload, err := decodeCommandPayload[LeavePayload](cmd)
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
	reason := strings.TrimSpace(payload.Reason)

	normalizedPayload := LeavePayload{ParticipantID: ids.ParticipantID(participantID), Reason: reason}
	payloadJSON, _ := json.Marshal(normalizedPayload)
	evt := command.NewEvent(cmd, EventTypeLeft, "participant", participantID, payloadJSON, now().UTC())
	return command.Accept(evt)
}
