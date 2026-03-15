package session

import (
	"encoding/json"
	"errors"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

type commandContract struct {
	definition command.Definition
}

type eventProjectionContract struct {
	definition event.Definition
	emittable  bool
	projection bool
}

var sessionCommandContracts = []commandContract{
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
	{
		definition: command.Definition{
			Type:            CommandTypeGateOpen,
			Owner:           command.OwnerCore,
			ValidatePayload: validateGateOpenedPayload,
			Gate: command.GatePolicy{
				Scope: command.GateScopeSession,
			},
			ActiveSession: command.AllowedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeGateRespond,
			Owner:           command.OwnerCore,
			ValidatePayload: validateGateResponseRecordedPayload,
			Gate: command.GatePolicy{
				Scope:         command.GateScopeSession,
				AllowWhenOpen: true,
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
				Scope:         command.GateScopeSession,
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
				Scope:         command.GateScopeSession,
				AllowWhenOpen: true,
			},
			ActiveSession: command.AllowedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeSpotlightSet,
			Owner:           command.OwnerCore,
			ValidatePayload: validateSpotlightSetPayload,
			Gate: command.GatePolicy{
				Scope:         command.GateScopeSession,
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
				Scope:         command.GateScopeSession,
				AllowWhenOpen: true,
			},
			ActiveSession: command.AllowedDuringActiveSession(),
		},
	},
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

var sessionEventContracts = []eventProjectionContract{
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
			Type:            EventTypeGateResponseRecorded,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateGateResponseRecordedPayload,
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

// RegisterCommands registers session commands with the shared registry.
func RegisterCommands(registry *command.Registry) error {
	if registry == nil {
		return errors.New("command registry is required")
	}
	for _, contract := range sessionCommandContracts {
		if err := registry.Register(contract.definition); err != nil {
			return err
		}
	}
	return nil
}

// EmittableEventTypes returns all event types the session decider can emit.
func EmittableEventTypes() []event.Type {
	return sessionEventTypes(func(contract eventProjectionContract) bool {
		return contract.emittable
	})
}

// DeciderHandledCommands returns all command types the session decider handles.
func DeciderHandledCommands() []command.Type {
	types := make([]command.Type, 0, len(sessionCommandContracts))
	for _, contract := range sessionCommandContracts {
		types = append(types, contract.definition.Type)
	}
	return types
}

// ProjectionHandledTypes returns the session event types that require
// projection handlers (IntentProjectionAndReplay).
func ProjectionHandledTypes() []event.Type {
	return sessionEventTypes(func(contract eventProjectionContract) bool {
		return contract.projection
	})
}

// RegisterEvents registers session events with the shared registry.
func RegisterEvents(registry *event.Registry) error {
	if registry == nil {
		return errors.New("event registry is required")
	}
	for _, contract := range sessionEventContracts {
		if err := registry.Register(contract.definition); err != nil {
			return err
		}
	}
	return nil
}

func sessionEventTypes(include func(eventProjectionContract) bool) []event.Type {
	types := make([]event.Type, 0, len(sessionEventContracts))
	for _, contract := range sessionEventContracts {
		if include(contract) {
			types = append(types, contract.definition.Type)
		}
	}
	return types
}

// validateStartPayload ensures start payloads match the session start shape.
func validateStartPayload(raw json.RawMessage) error {
	var payload StartPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	return nil
}

// validateEndPayload ensures end payloads match the session end shape.
func validateEndPayload(raw json.RawMessage) error {
	var payload EndPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	return nil
}

// validateGateOpenedPayload ensures gate opened payloads match the gate open shape.
func validateGateOpenedPayload(raw json.RawMessage) error {
	var payload GateOpenedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	gateType, err := NormalizeGateType(payload.GateType)
	if err != nil {
		return err
	}
	_, err = NormalizeGateWorkflowMetadata(gateType, payload.Metadata)
	return err
}

// validateGateResolvedPayload ensures gate resolved payloads match the gate resolve shape.
func validateGateResolvedPayload(raw json.RawMessage) error {
	var payload GateResolvedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	return nil
}

// validateGateResponseRecordedPayload ensures gate response payloads match the response shape.
func validateGateResponseRecordedPayload(raw json.RawMessage) error {
	var payload GateResponseRecordedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	return nil
}

// validateGateAbandonedPayload ensures gate abandoned payloads match the gate abandon shape.
func validateGateAbandonedPayload(raw json.RawMessage) error {
	var payload GateAbandonedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	return nil
}

func validateActiveSceneSetPayload(raw json.RawMessage) error {
	var payload ActiveSceneSetPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	return nil
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

// validateSpotlightSetPayload ensures spotlight set payloads match the set shape.
func validateSpotlightSetPayload(raw json.RawMessage) error {
	var payload SpotlightSetPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	return nil
}

// validateSpotlightClearedPayload ensures spotlight cleared payloads match the clear shape.
func validateSpotlightClearedPayload(raw json.RawMessage) error {
	var payload SpotlightClearedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	return nil
}
