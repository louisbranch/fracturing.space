package action

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

var (
	errActionRequestIDRequired = errors.New("request_id is required")
	errActionRollSeqRequired   = errors.New("roll_seq must be greater than zero")
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
		{
			Type:            eventTypeRollResolved,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateRollResolvePayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		{
			Type:            eventTypeOutcomeApplied,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateOutcomeApplyPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		{
			Type:            eventTypeOutcomeRejected,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateOutcomeRejectPayload,
			Intent:          event.IntentAuditOnly,
		},
		{
			Type:            eventTypeNoteAdded,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateNoteAddPayload,
			Intent:          event.IntentAuditOnly,
		},
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
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	return validateActionRequestAndRoll(payload.RequestID, payload.RollSeq)
}

func validateOutcomeApplyPayload(raw json.RawMessage) error {
	var payload OutcomeApplyPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	return validateActionRequestAndRoll(payload.RequestID, payload.RollSeq)
}

func validateOutcomeRejectPayload(raw json.RawMessage) error {
	var payload OutcomeRejectPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	return validateActionRequestAndRoll(payload.RequestID, payload.RollSeq)
}

func validateNoteAddPayload(raw json.RawMessage) error {
	var payload NoteAddPayload
	return json.Unmarshal(raw, &payload)
}

func validateActionRequestAndRoll(requestID string, rollSeq uint64) error {
	if strings.TrimSpace(requestID) == "" {
		return errActionRequestIDRequired
	}
	if rollSeq == 0 {
		return errActionRollSeqRequired
	}
	return nil
}
