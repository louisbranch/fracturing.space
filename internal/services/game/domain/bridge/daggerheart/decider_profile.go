package daggerheart

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func decideCharacterProfileReplace(cmd command.Command, now func() time.Time) command.Decision {
	var payload CharacterProfileReplacePayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{
			Code:    rejectionCodePayloadDecodeFailed,
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}

	characterID := strings.TrimSpace(payload.CharacterID.String())
	if characterID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodePayloadDecodeFailed,
			Message: "character_id is required",
		})
	}
	if err := payload.Profile.Validate(); err != nil {
		return command.Reject(command.Rejection{
			Code:    rejectionCodePayloadDecodeFailed,
			Message: err.Error(),
		})
	}

	normalized := CharacterProfileReplacePayload{
		CharacterID: ids.CharacterID(characterID),
		Profile:     payload.Profile.Normalized(),
	}
	payloadJSON, _ := json.Marshal(normalized)
	evt := command.NewEvent(cmd, EventTypeCharacterProfileReplaced, "character", characterID, payloadJSON, now().UTC())
	return command.Accept(evt)
}

func decideCharacterProfileDelete(cmd command.Command, now func() time.Time) command.Decision {
	var payload CharacterProfileDeletePayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{
			Code:    rejectionCodePayloadDecodeFailed,
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}

	characterID := strings.TrimSpace(payload.CharacterID.String())
	if characterID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodePayloadDecodeFailed,
			Message: "character_id is required",
		})
	}

	normalized := CharacterProfileDeletePayload{
		CharacterID: ids.CharacterID(characterID),
		Reason:      strings.TrimSpace(payload.Reason),
	}
	payloadJSON, _ := json.Marshal(normalized)
	evt := command.NewEvent(cmd, EventTypeCharacterProfileDeleted, "character", characterID, payloadJSON, now().UTC())
	evt.ActorType = event.ActorType(cmd.ActorType)
	return command.Accept(evt)
}
