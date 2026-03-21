package scene

import (
	"encoding/json"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

var sceneInteractionCommandContracts = []commandContract{
	{
		definition: command.Definition{
			Type:            CommandTypePlayerPhaseStart,
			Owner:           command.OwnerCore,
			ValidatePayload: validatePlayerPhaseStartedPayload,
			Gate: command.GatePolicy{
				Scope: command.GateScopeScene,
			},
			ActiveSession: command.AllowedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypePlayerPhasePost,
			Owner:           command.OwnerCore,
			ValidatePayload: validatePlayerPhasePostedPayload,
			Gate: command.GatePolicy{
				Scope: command.GateScopeScene,
			},
			ActiveSession: command.AllowedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypePlayerPhaseYield,
			Owner:           command.OwnerCore,
			ValidatePayload: validatePlayerPhaseYieldedPayload,
			Gate: command.GatePolicy{
				Scope: command.GateScopeScene,
			},
			ActiveSession: command.AllowedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypePlayerPhaseUnyield,
			Owner:           command.OwnerCore,
			ValidatePayload: validatePlayerPhaseUnyieldedPayload,
			Gate: command.GatePolicy{
				Scope: command.GateScopeScene,
			},
			ActiveSession: command.AllowedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypePlayerPhaseAccept,
			Owner:           command.OwnerCore,
			ValidatePayload: validatePlayerPhaseAcceptedPayload,
			Gate: command.GatePolicy{
				Scope:         command.GateScopeScene,
				AllowWhenOpen: true,
			},
			ActiveSession: command.AllowedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypePlayerPhaseRequestRevisions,
			Owner:           command.OwnerCore,
			ValidatePayload: validatePlayerPhaseRevisionsRequestedPayload,
			Gate: command.GatePolicy{
				Scope:         command.GateScopeScene,
				AllowWhenOpen: true,
			},
			ActiveSession: command.AllowedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypePlayerPhaseEnd,
			Owner:           command.OwnerCore,
			ValidatePayload: validatePlayerPhaseEndedPayload,
			Gate: command.GatePolicy{
				Scope:         command.GateScopeScene,
				AllowWhenOpen: true,
			},
			ActiveSession: command.AllowedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeGMInteractionCommit,
			Owner:           command.OwnerCore,
			ValidatePayload: validateGMInteractionCommittedPayload,
			Gate: command.GatePolicy{
				Scope:         command.GateScopeScene,
				AllowWhenOpen: true,
			},
			ActiveSession: command.AllowedDuringActiveSession(),
		},
	},
}

var sceneInteractionEventContracts = []eventProjectionContract{
	{
		definition: event.Definition{
			Type:            EventTypePlayerPhaseStarted,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validatePlayerPhaseStartedPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypePlayerPhasePosted,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validatePlayerPhasePostedPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypePlayerPhaseYielded,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validatePlayerPhaseYieldedPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypePlayerPhaseReviewStarted,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validatePlayerPhaseReviewStartedPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypePlayerPhaseUnyielded,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validatePlayerPhaseUnyieldedPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypePlayerPhaseRevisionsRequested,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validatePlayerPhaseRevisionsRequestedPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypePlayerPhaseAccepted,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validatePlayerPhaseAcceptedPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypePlayerPhaseEnded,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validatePlayerPhaseEndedPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeGMInteractionCommitted,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateGMInteractionCommittedPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
}

func validatePlayerPhaseStartedPayload(raw json.RawMessage) error {
	var payload PlayerPhaseStartedPayload
	return json.Unmarshal(raw, &payload)
}

func validatePlayerPhasePostedPayload(raw json.RawMessage) error {
	var payload PlayerPhasePostedPayload
	return json.Unmarshal(raw, &payload)
}

func validatePlayerPhaseYieldedPayload(raw json.RawMessage) error {
	var payload PlayerPhaseYieldedPayload
	return json.Unmarshal(raw, &payload)
}

func validatePlayerPhaseReviewStartedPayload(raw json.RawMessage) error {
	var payload PlayerPhaseReviewStartedPayload
	return json.Unmarshal(raw, &payload)
}

func validatePlayerPhaseUnyieldedPayload(raw json.RawMessage) error {
	var payload PlayerPhaseUnyieldedPayload
	return json.Unmarshal(raw, &payload)
}

func validatePlayerPhaseRevisionsRequestedPayload(raw json.RawMessage) error {
	var payload PlayerPhaseRevisionsRequestedPayload
	return json.Unmarshal(raw, &payload)
}

func validatePlayerPhaseAcceptedPayload(raw json.RawMessage) error {
	var payload PlayerPhaseAcceptedPayload
	return json.Unmarshal(raw, &payload)
}

func validatePlayerPhaseEndedPayload(raw json.RawMessage) error {
	var payload PlayerPhaseEndedPayload
	return json.Unmarshal(raw, &payload)
}

func validateGMInteractionCommittedPayload(raw json.RawMessage) error {
	var payload GMInteractionCommittedPayload
	return json.Unmarshal(raw, &payload)
}
