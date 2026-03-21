package session

import (
	"encoding/json"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

var sessionInteractionCommandContracts = []commandContract{
	{
		definition: command.Definition{
			Type:            CommandTypeActiveSceneSet,
			Owner:           command.OwnerCore,
			ValidatePayload: validateActiveSceneSetPayload,
			Gate: command.GatePolicy{
				Scope:         command.GateScopeSession,
				AllowWhenOpen: true,
			},
			ActiveSession: command.AllowedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeGMAuthoritySet,
			Owner:           command.OwnerCore,
			ValidatePayload: validateGMAuthoritySetPayload,
			Gate: command.GatePolicy{
				Scope:         command.GateScopeSession,
				AllowWhenOpen: true,
			},
			ActiveSession: command.AllowedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeOOCPause,
			Owner:           command.OwnerCore,
			ValidatePayload: validateOOCPausedPayload,
			Gate: command.GatePolicy{
				Scope:         command.GateScopeSession,
				AllowWhenOpen: true,
			},
			ActiveSession: command.AllowedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeOOCPost,
			Owner:           command.OwnerCore,
			ValidatePayload: validateOOCPostedPayload,
			Gate: command.GatePolicy{
				Scope:         command.GateScopeSession,
				AllowWhenOpen: true,
			},
			ActiveSession: command.AllowedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeOOCReadyMark,
			Owner:           command.OwnerCore,
			ValidatePayload: validateOOCReadyMarkedPayload,
			Gate: command.GatePolicy{
				Scope:         command.GateScopeSession,
				AllowWhenOpen: true,
			},
			ActiveSession: command.AllowedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeOOCReadyClear,
			Owner:           command.OwnerCore,
			ValidatePayload: validateOOCReadyClearedPayload,
			Gate: command.GatePolicy{
				Scope:         command.GateScopeSession,
				AllowWhenOpen: true,
			},
			ActiveSession: command.AllowedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeOOCResume,
			Owner:           command.OwnerCore,
			ValidatePayload: validateOOCResumedPayload,
			Gate: command.GatePolicy{
				Scope:         command.GateScopeSession,
				AllowWhenOpen: true,
			},
			ActiveSession: command.AllowedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeAITurnQueue,
			Owner:           command.OwnerCore,
			ValidatePayload: validateAITurnQueuedPayload,
			Gate: command.GatePolicy{
				Scope:         command.GateScopeSession,
				AllowWhenOpen: true,
			},
			ActiveSession: command.AllowedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeAITurnStart,
			Owner:           command.OwnerCore,
			ValidatePayload: validateAITurnRunningPayload,
			Gate: command.GatePolicy{
				Scope:         command.GateScopeSession,
				AllowWhenOpen: true,
			},
			ActiveSession: command.AllowedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeAITurnFail,
			Owner:           command.OwnerCore,
			ValidatePayload: validateAITurnFailedPayload,
			Gate: command.GatePolicy{
				Scope:         command.GateScopeSession,
				AllowWhenOpen: true,
			},
			ActiveSession: command.AllowedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeAITurnClear,
			Owner:           command.OwnerCore,
			ValidatePayload: validateAITurnClearedPayload,
			Gate: command.GatePolicy{
				Scope:         command.GateScopeSession,
				AllowWhenOpen: true,
			},
			ActiveSession: command.AllowedDuringActiveSession(),
		},
	},
}

var sessionInteractionEventContracts = []eventProjectionContract{
	{
		definition: event.Definition{
			Type:            EventTypeActiveSceneSet,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateActiveSceneSetPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeGMAuthoritySet,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateGMAuthoritySetPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeOOCPaused,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateOOCPausedPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeOOCPosted,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateOOCPostedPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeOOCReadyMarked,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateOOCReadyMarkedPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeOOCReadyCleared,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateOOCReadyClearedPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeOOCResumed,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateOOCResumedPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeAITurnQueued,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateAITurnQueuedPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeAITurnRunning,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateAITurnRunningPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeAITurnFailed,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateAITurnFailedPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeAITurnCleared,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateAITurnClearedPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
}

func validateActiveSceneSetPayload(raw json.RawMessage) error {
	var payload ActiveSceneSetPayload
	return json.Unmarshal(raw, &payload)
}

func validateGMAuthoritySetPayload(raw json.RawMessage) error {
	var payload GMAuthoritySetPayload
	return json.Unmarshal(raw, &payload)
}

func validateOOCPausedPayload(raw json.RawMessage) error {
	var payload OOCPausedPayload
	return json.Unmarshal(raw, &payload)
}

func validateOOCPostedPayload(raw json.RawMessage) error {
	var payload OOCPostedPayload
	return json.Unmarshal(raw, &payload)
}

func validateOOCReadyMarkedPayload(raw json.RawMessage) error {
	var payload OOCReadyMarkedPayload
	return json.Unmarshal(raw, &payload)
}

func validateOOCReadyClearedPayload(raw json.RawMessage) error {
	var payload OOCReadyClearedPayload
	return json.Unmarshal(raw, &payload)
}

func validateOOCResumedPayload(raw json.RawMessage) error {
	var payload OOCResumedPayload
	return json.Unmarshal(raw, &payload)
}

func validateAITurnQueuedPayload(raw json.RawMessage) error {
	var payload AITurnQueuedPayload
	return json.Unmarshal(raw, &payload)
}

func validateAITurnRunningPayload(raw json.RawMessage) error {
	var payload AITurnRunningPayload
	return json.Unmarshal(raw, &payload)
}

func validateAITurnFailedPayload(raw json.RawMessage) error {
	var payload AITurnFailedPayload
	return json.Unmarshal(raw, &payload)
}

func validateAITurnClearedPayload(raw json.RawMessage) error {
	var payload AITurnClearedPayload
	return json.Unmarshal(raw, &payload)
}
