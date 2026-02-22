package session

import (
	"encoding/json"
	"errors"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// RegisterCommands registers session commands with the shared registry.
func RegisterCommands(registry *command.Registry) error {
	if registry == nil {
		return errors.New("command registry is required")
	}
	if err := registry.Register(command.Definition{
		Type:            commandTypeStart,
		Owner:           command.OwnerCore,
		ValidatePayload: validateStartPayload,
	}); err != nil {
		return err
	}
	if err := registry.Register(command.Definition{
		Type:            commandTypeEnd,
		Owner:           command.OwnerCore,
		ValidatePayload: validateEndPayload,
	}); err != nil {
		return err
	}
	if err := registry.Register(command.Definition{
		Type:            commandTypeGateOpen,
		Owner:           command.OwnerCore,
		ValidatePayload: validateGateOpenedPayload,
		Gate: command.GatePolicy{
			Scope: command.GateScopeSession,
		},
	}); err != nil {
		return err
	}
	if err := registry.Register(command.Definition{
		Type:            commandTypeGateResolve,
		Owner:           command.OwnerCore,
		ValidatePayload: validateGateResolvedPayload,
		Gate: command.GatePolicy{
			Scope:         command.GateScopeSession,
			AllowWhenOpen: true,
		},
	}); err != nil {
		return err
	}
	if err := registry.Register(command.Definition{
		Type:            commandTypeGateAbandon,
		Owner:           command.OwnerCore,
		ValidatePayload: validateGateAbandonedPayload,
		Gate: command.GatePolicy{
			Scope:         command.GateScopeSession,
			AllowWhenOpen: true,
		},
	}); err != nil {
		return err
	}
	if err := registry.Register(command.Definition{
		Type:            commandTypeSpotlightSet,
		Owner:           command.OwnerCore,
		ValidatePayload: validateSpotlightSetPayload,
		Gate: command.GatePolicy{
			Scope:         command.GateScopeSession,
			AllowWhenOpen: true,
		},
	}); err != nil {
		return err
	}
	return registry.Register(command.Definition{
		Type:            commandTypeSpotlightClear,
		Owner:           command.OwnerCore,
		ValidatePayload: validateSpotlightClearedPayload,
		Gate: command.GatePolicy{
			Scope:         command.GateScopeSession,
			AllowWhenOpen: true,
		},
	})
}

// EmittableEventTypes returns all event types the session decider can emit.
func EmittableEventTypes() []event.Type {
	return []event.Type{
		EventTypeStarted,
		EventTypeEnded,
		EventTypeGateOpened,
		EventTypeGateResolved,
		EventTypeGateAbandoned,
		EventTypeSpotlightSet,
		EventTypeSpotlightCleared,
	}
}

// RegisterEvents registers session events with the shared registry.
func RegisterEvents(registry *event.Registry) error {
	if registry == nil {
		return errors.New("event registry is required")
	}
	if err := registry.Register(event.Definition{
		Type:            EventTypeStarted,
		Owner:           event.OwnerCore,
		Addressing:      event.AddressingPolicyEntityTarget,
		ValidatePayload: validateStartPayload,
		Intent:          event.IntentProjectionAndReplay,
	}); err != nil {
		return err
	}
	if err := registry.Register(event.Definition{
		Type:            EventTypeEnded,
		Owner:           event.OwnerCore,
		Addressing:      event.AddressingPolicyEntityTarget,
		ValidatePayload: validateEndPayload,
		Intent:          event.IntentProjectionAndReplay,
	}); err != nil {
		return err
	}
	if err := registry.Register(event.Definition{
		Type:            EventTypeGateOpened,
		Owner:           event.OwnerCore,
		Addressing:      event.AddressingPolicyEntityTarget,
		ValidatePayload: validateGateOpenedPayload,
		Intent:          event.IntentProjectionAndReplay,
	}); err != nil {
		return err
	}
	if err := registry.Register(event.Definition{
		Type:            EventTypeGateResolved,
		Owner:           event.OwnerCore,
		Addressing:      event.AddressingPolicyEntityTarget,
		ValidatePayload: validateGateResolvedPayload,
		Intent:          event.IntentProjectionAndReplay,
	}); err != nil {
		return err
	}
	if err := registry.Register(event.Definition{
		Type:            EventTypeGateAbandoned,
		Owner:           event.OwnerCore,
		Addressing:      event.AddressingPolicyEntityTarget,
		ValidatePayload: validateGateAbandonedPayload,
		Intent:          event.IntentProjectionAndReplay,
	}); err != nil {
		return err
	}
	if err := registry.Register(event.Definition{
		Type:            EventTypeSpotlightSet,
		Owner:           event.OwnerCore,
		Addressing:      event.AddressingPolicyEntityTarget,
		ValidatePayload: validateSpotlightSetPayload,
		Intent:          event.IntentProjectionAndReplay,
	}); err != nil {
		return err
	}
	return registry.Register(event.Definition{
		Type:            EventTypeSpotlightCleared,
		Owner:           event.OwnerCore,
		Addressing:      event.AddressingPolicyEntityTarget,
		ValidatePayload: validateSpotlightClearedPayload,
		Intent:          event.IntentProjectionAndReplay,
	})
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
	return nil
}

// validateGateResolvedPayload ensures gate resolved payloads match the gate resolve shape.
func validateGateResolvedPayload(raw json.RawMessage) error {
	var payload GateResolvedPayload
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
