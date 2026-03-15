package action

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
)

func decideNoteAdd(cmd command.Command, now func() time.Time) command.Decision {
	var payload NoteAddPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{
			Code:    command.RejectionCodePayloadDecodeFailed,
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}
	return acceptActionEvent(cmd, now, EventTypeNoteAdded, "note", cmd.EntityID, payload)
}
