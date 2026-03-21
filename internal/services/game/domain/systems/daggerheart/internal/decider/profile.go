package decider

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/snapstate"
)

func decideCharacterProfileReplace(cmd command.Command, now func() time.Time) command.Decision {
	var p snapstate.CharacterProfileReplacePayload
	if err := json.Unmarshal(cmd.PayloadJSON, &p); err != nil {
		return command.Reject(command.Rejection{
			Code:    rejectionCodePayloadDecodeFailed,
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}

	characterID := strings.TrimSpace(p.CharacterID.String())
	if characterID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodePayloadDecodeFailed,
			Message: "character_id is required",
		})
	}
	if err := p.Profile.Validate(); err != nil {
		return command.Reject(command.Rejection{
			Code:    rejectionCodePayloadDecodeFailed,
			Message: err.Error(),
		})
	}

	normalized := snapstate.CharacterProfileReplacePayload{
		CharacterID: ids.CharacterID(characterID),
		Profile:     p.Profile.Normalized(),
	}
	payloadJSON, _ := json.Marshal(normalized)
	evt := command.NewEvent(cmd, payload.EventTypeCharacterProfileReplaced, "character", characterID, payloadJSON, now().UTC())
	return command.Accept(evt)
}

func decideCharacterProfileDelete(cmd command.Command, now func() time.Time) command.Decision {
	var p snapstate.CharacterProfileDeletePayload
	if err := json.Unmarshal(cmd.PayloadJSON, &p); err != nil {
		return command.Reject(command.Rejection{
			Code:    rejectionCodePayloadDecodeFailed,
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}

	characterID := strings.TrimSpace(p.CharacterID.String())
	if characterID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodePayloadDecodeFailed,
			Message: "character_id is required",
		})
	}

	normalized := snapstate.CharacterProfileDeletePayload{
		CharacterID: ids.CharacterID(characterID),
		Reason:      strings.TrimSpace(p.Reason),
	}
	payloadJSON, _ := json.Marshal(normalized)
	evt := command.NewEvent(cmd, payload.EventTypeCharacterProfileDeleted, "character", characterID, payloadJSON, now().UTC())
	evt.ActorType = event.ActorType(cmd.ActorType)
	return command.Accept(evt)
}
