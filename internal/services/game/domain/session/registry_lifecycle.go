package session

import (
	"encoding/json"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

var sessionLifecycleCommandContracts = []commandContract{
	{
		definition: command.Definition{
			Type:            CommandTypeStart,
			Owner:           command.OwnerCore,
			ValidatePayload: validateStartPayload,
			ActiveSession:   command.AllowedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeEnd,
			Owner:           command.OwnerCore,
			ValidatePayload: validateEndPayload,
			ActiveSession:   command.AllowedDuringActiveSession(),
		},
	},
}

var sessionLifecycleEventContracts = []eventProjectionContract{
	{
		definition: event.Definition{
			Type:            EventTypeStarted,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateStartPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeEnded,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateEndPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
}

func validateStartPayload(raw json.RawMessage) error {
	var payload StartPayload
	return json.Unmarshal(raw, &payload)
}

func validateEndPayload(raw json.RawMessage) error {
	var payload EndPayload
	return json.Unmarshal(raw, &payload)
}
