package scene

import (
	"encoding/json"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

var sceneGateCommandContracts = []commandContract{
	{
		definition: command.Definition{
			Type:            CommandTypeGateOpen,
			Owner:           command.OwnerCore,
			ValidatePayload: validateGateOpenedPayload,
			Gate: command.GatePolicy{
				Scope: command.GateScopeScene,
			},
			ActiveSession: command.AllowedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeGateResolve,
			Owner:           command.OwnerCore,
			ValidatePayload: validateGateResolvedPayload,
			Gate: command.GatePolicy{
				Scope:         command.GateScopeScene,
				AllowWhenOpen: true,
			},
			ActiveSession: command.AllowedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeGateAbandon,
			Owner:           command.OwnerCore,
			ValidatePayload: validateGateAbandonedPayload,
			Gate: command.GatePolicy{
				Scope:         command.GateScopeScene,
				AllowWhenOpen: true,
			},
			ActiveSession: command.AllowedDuringActiveSession(),
		},
	},
}

var sceneGateEventContracts = []eventProjectionContract{
	{
		definition: event.Definition{
			Type:            EventTypeGateOpened,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateGateOpenedPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeGateResolved,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateGateResolvedPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeGateAbandoned,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateGateAbandonedPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
}

func validateGateOpenedPayload(raw json.RawMessage) error {
	var payload GateOpenedPayload
	return json.Unmarshal(raw, &payload)
}

func validateGateResolvedPayload(raw json.RawMessage) error {
	var payload GateResolvedPayload
	return json.Unmarshal(raw, &payload)
}

func validateGateAbandonedPayload(raw json.RawMessage) error {
	var payload GateAbandonedPayload
	return json.Unmarshal(raw, &payload)
}
