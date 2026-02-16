package action

import (
	"encoding/json"
	"errors"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// RegisterCommands registers action commands with the shared registry.
func RegisterCommands(registry *command.Registry) error {
	if registry == nil {
		return errors.New("command registry is required")
	}
	definitions := []command.Definition{
		{Type: commandTypeRollResolve, Owner: command.OwnerCore, ValidatePayload: validateRollResolvePayload},
		{Type: commandTypeOutcomeApply, Owner: command.OwnerCore, ValidatePayload: validateOutcomeApplyPayload},
		{Type: commandTypeOutcomeReject, Owner: command.OwnerCore, ValidatePayload: validateOutcomeRejectPayload},
		{Type: commandTypeNoteAdd, Owner: command.OwnerCore, ValidatePayload: validateNoteAddPayload},
	}
	for _, definition := range definitions {
		if err := registry.Register(definition); err != nil {
			return err
		}
	}
	return nil
}

// RegisterEvents registers action events with the shared registry.
func RegisterEvents(registry *event.Registry) error {
	if registry == nil {
		return errors.New("event registry is required")
	}
	definitions := []event.Definition{
		{Type: eventTypeRollResolved, Owner: event.OwnerCore, ValidatePayload: validateRollResolvePayload},
		{Type: eventTypeOutcomeApplied, Owner: event.OwnerCore, ValidatePayload: validateOutcomeApplyPayload},
		{Type: eventTypeOutcomeRejected, Owner: event.OwnerCore, ValidatePayload: validateOutcomeRejectPayload},
		{Type: eventTypeNoteAdded, Owner: event.OwnerCore, ValidatePayload: validateNoteAddPayload},
	}
	for _, definition := range definitions {
		if err := registry.Register(definition); err != nil {
			return err
		}
	}
	return nil
}

func validateRollResolvePayload(raw json.RawMessage) error {
	var payload RollResolvePayload
	return json.Unmarshal(raw, &payload)
}

func validateOutcomeApplyPayload(raw json.RawMessage) error {
	var payload OutcomeApplyPayload
	return json.Unmarshal(raw, &payload)
}

func validateOutcomeRejectPayload(raw json.RawMessage) error {
	var payload OutcomeRejectPayload
	return json.Unmarshal(raw, &payload)
}

func validateNoteAddPayload(raw json.RawMessage) error {
	var payload NoteAddPayload
	return json.Unmarshal(raw, &payload)
}
