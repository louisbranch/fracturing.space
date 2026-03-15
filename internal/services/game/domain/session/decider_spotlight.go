package session

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

func decideSpotlightSet(cmd command.Command, now func() time.Time) command.Decision {
	var payload SpotlightSetPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{Code: command.RejectionCodePayloadDecodeFailed, Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
	}
	normalizedType, err := NormalizeSpotlightType(payload.SpotlightType)
	if err != nil {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionSpotlightTypeInvalid,
			Message: err.Error(),
		})
	}
	characterID := strings.TrimSpace(payload.CharacterID.String())
	if err := ValidateSpotlightTarget(normalizedType, characterID); err != nil {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionSpotlightTargetInvalid,
			Message: err.Error(),
		})
	}

	normalizedPayload := SpotlightSetPayload{SpotlightType: string(normalizedType), CharacterID: ids.CharacterID(characterID)}
	payloadJSON, _ := json.Marshal(normalizedPayload)
	evt := command.NewEvent(cmd, EventTypeSpotlightSet, "session", cmd.SessionID.String(), payloadJSON, now().UTC())

	return command.Accept(evt)
}

func decideSpotlightClear(cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFunc(
		cmd,
		EventTypeSpotlightCleared,
		"session",
		func(_ *SpotlightClearedPayload) string {
			return cmd.SessionID.String()
		},
		func(payload *SpotlightClearedPayload, _ func() time.Time) *command.Rejection {
			payload.Reason = strings.TrimSpace(payload.Reason)
			return nil
		},
		now,
	)
}
