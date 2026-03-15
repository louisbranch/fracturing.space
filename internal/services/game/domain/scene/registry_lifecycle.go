package scene

import (
	"encoding/json"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

var sceneLifecycleCommandContracts = []commandContract{
	{
		definition: command.Definition{
			Type:            CommandTypeCreate,
			Owner:           command.OwnerCore,
			ValidatePayload: validateCreatePayload,
			ActiveSession:   command.AllowedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeUpdate,
			Owner:           command.OwnerCore,
			ValidatePayload: validateUpdatePayload,
			Gate: command.GatePolicy{
				Scope: command.GateScopeScene,
			},
			ActiveSession: command.AllowedDuringActiveSession(),
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

var sceneTransitionCommandContracts = []commandContract{
	{
		definition: command.Definition{
			Type:            CommandTypeTransition,
			Owner:           command.OwnerCore,
			ValidatePayload: validateTransitionPayload,
			ActiveSession:   command.AllowedDuringActiveSession(),
		},
	},
}

var sceneLifecycleEventContracts = []eventProjectionContract{
	{
		definition: event.Definition{
			Type:            EventTypeCreated,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateCreatePayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeUpdated,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateUpdatePayload,
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

func validateCreatePayload(raw json.RawMessage) error {
	var payload CreatePayload
	return json.Unmarshal(raw, &payload)
}

func validateUpdatePayload(raw json.RawMessage) error {
	var payload UpdatePayload
	return json.Unmarshal(raw, &payload)
}

func validateEndPayload(raw json.RawMessage) error {
	var payload EndPayload
	return json.Unmarshal(raw, &payload)
}

func validateTransitionPayload(raw json.RawMessage) error {
	var payload TransitionPayload
	return json.Unmarshal(raw, &payload)
}
