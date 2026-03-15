package action

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
)

func decideRollResolve(cmd command.Command, now func() time.Time) command.Decision {
	var payload RollResolvePayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{
			Code:    command.RejectionCodePayloadDecodeFailed,
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}
	requestID := strings.TrimSpace(payload.RequestID)
	if requestID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeRequestIDRequired,
			Message: "request_id is required",
		})
	}
	if payload.RollSeq == 0 {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeRollSeqRequired,
			Message: "roll_seq must be greater than zero",
		})
	}
	return acceptActionEvent(cmd, now, EventTypeRollResolved, "roll", requestID, payload)
}
