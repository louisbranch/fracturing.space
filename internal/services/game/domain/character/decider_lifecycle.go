package character

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func decideDelete(state State, cmd command.Command, now func() time.Time) command.Decision {
	if !state.Created || state.Deleted {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeCharacterNotCreated,
			Message: "character not created",
		})
	}
	var payload DeletePayload
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
	reason := strings.TrimSpace(payload.Reason)

	normalizedPayload := DeletePayload{CharacterID: ids.CharacterID(characterID), Reason: reason}
	payloadJSON, _ := json.Marshal(normalizedPayload)
	evt := command.NewEvent(cmd, EventTypeDeleted, "character", characterID, payloadJSON, now().UTC())

	return command.Accept(evt)
}
