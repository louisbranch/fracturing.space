package scene

import (
	"encoding/json"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

var sceneSpotlightCommandContracts = []commandContract{
	{
		definition: command.Definition{
			Type:            CommandTypeSpotlightSet,
			Owner:           command.OwnerCore,
			ValidatePayload: validateSpotlightSetPayload,
			Gate: command.GatePolicy{
				Scope:         command.GateScopeScene,
				AllowWhenOpen: true,
			},
			ActiveSession: command.AllowedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeSpotlightClear,
			Owner:           command.OwnerCore,
			ValidatePayload: validateSpotlightClearedPayload,
			Gate: command.GatePolicy{
				Scope:         command.GateScopeScene,
				AllowWhenOpen: true,
			},
			ActiveSession: command.AllowedDuringActiveSession(),
		},
	},
}

var sceneSpotlightEventContracts = []eventProjectionContract{
	{
		definition: event.Definition{
			Type:            EventTypeSpotlightSet,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateSpotlightSetPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeSpotlightCleared,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateSpotlightClearedPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
}

func validateSpotlightSetPayload(raw json.RawMessage) error {
	var payload SpotlightSetPayload
	return json.Unmarshal(raw, &payload)
}

func validateSpotlightClearedPayload(raw json.RawMessage) error {
	var payload SpotlightClearedPayload
	return json.Unmarshal(raw, &payload)
}
