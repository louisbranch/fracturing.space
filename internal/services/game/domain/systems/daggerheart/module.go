package daggerheart

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/system"
)

// Module wires Daggerheart system behavior into the runtime.
type Module struct {
	decider   system.Decider
	projector system.Projector
	factory   system.StateFactory
}

// NewModule creates a Daggerheart system module.
func NewModule() *Module {
	return &Module{
		decider:   Decider{},
		projector: Projector{},
		factory:   NewStateFactory(),
	}
}

// ID returns the Daggerheart system identifier.
func (m *Module) ID() string {
	return SystemID
}

// Version returns the Daggerheart system version.
func (m *Module) Version() string {
	return SystemVersion
}

var daggerheartCommandDefinitions = []command.Definition{
	{Type: commandTypeGMFearSet, Owner: command.OwnerSystem, ValidatePayload: validateGMFearSetPayload},
	{Type: commandTypeCharacterStatePatch, Owner: command.OwnerSystem, ValidatePayload: validateCharacterStatePatchPayload},
	{Type: commandTypeConditionChange, Owner: command.OwnerSystem, ValidatePayload: validateConditionChangePayload},
	{Type: commandTypeHopeSpend, Owner: command.OwnerSystem, ValidatePayload: validateHopeSpendPayload},
	{Type: commandTypeStressSpend, Owner: command.OwnerSystem, ValidatePayload: validateStressSpendPayload},
	{Type: commandTypeLoadoutSwap, Owner: command.OwnerSystem, ValidatePayload: validateLoadoutSwapPayload},
	{Type: commandTypeRestTake, Owner: command.OwnerSystem, ValidatePayload: validateRestTakePayload},
	{Type: commandTypeCountdownCreate, Owner: command.OwnerSystem, ValidatePayload: validateCountdownCreatePayload},
	{Type: commandTypeCountdownUpdate, Owner: command.OwnerSystem, ValidatePayload: validateCountdownUpdatePayload},
	{Type: commandTypeCountdownDelete, Owner: command.OwnerSystem, ValidatePayload: validateCountdownDeletePayload},
	{Type: commandTypeDamageApply, Owner: command.OwnerSystem, ValidatePayload: validateDamageApplyPayload},
	{Type: commandTypeAdversaryDamageApply, Owner: command.OwnerSystem, ValidatePayload: validateAdversaryDamageApplyPayload},
	{Type: commandTypeDowntimeMoveApply, Owner: command.OwnerSystem, ValidatePayload: validateDowntimeMoveApplyPayload},
	{Type: commandTypeCharacterTemporaryArmorApply, Owner: command.OwnerSystem, ValidatePayload: validateCharacterTemporaryArmorApplyPayload},
	{Type: commandTypeAdversaryConditionChange, Owner: command.OwnerSystem, ValidatePayload: validateAdversaryConditionChangePayload},
	{Type: commandTypeAdversaryCreate, Owner: command.OwnerSystem, ValidatePayload: validateAdversaryCreatePayload},
	{Type: commandTypeAdversaryUpdate, Owner: command.OwnerSystem, ValidatePayload: validateAdversaryUpdatePayload},
	{Type: commandTypeAdversaryDelete, Owner: command.OwnerSystem, ValidatePayload: validateAdversaryDeletePayload},
}

var daggerheartEventDefinitions = []event.Definition{
	{Type: eventTypeGMFearChanged, Owner: event.OwnerSystem, ValidatePayload: validateGMFearChangedPayload},
	{Type: eventTypeCharacterStatePatched, Owner: event.OwnerSystem, ValidatePayload: validateCharacterStatePatchedPayload},
	{Type: eventTypeConditionChanged, Owner: event.OwnerSystem, ValidatePayload: validateConditionChangedPayload},
	{Type: eventTypeLoadoutSwapped, Owner: event.OwnerSystem, ValidatePayload: validateLoadoutSwappedPayload},
	{Type: eventTypeRestTaken, Owner: event.OwnerSystem, ValidatePayload: validateRestTakenPayload},
	{Type: eventTypeCountdownCreated, Owner: event.OwnerSystem, ValidatePayload: validateCountdownCreatedPayload},
	{Type: eventTypeCountdownUpdated, Owner: event.OwnerSystem, ValidatePayload: validateCountdownUpdatedPayload},
	{Type: eventTypeCountdownDeleted, Owner: event.OwnerSystem, ValidatePayload: validateCountdownDeletedPayload},
	{Type: eventTypeDamageApplied, Owner: event.OwnerSystem, ValidatePayload: validateDamageAppliedPayload},
	{Type: eventTypeAdversaryDamageApplied, Owner: event.OwnerSystem, ValidatePayload: validateAdversaryDamageAppliedPayload},
	{Type: eventTypeDowntimeMoveApplied, Owner: event.OwnerSystem, ValidatePayload: validateDowntimeMoveAppliedPayload},
	{Type: eventTypeCharacterTemporaryArmorApplied, Owner: event.OwnerSystem, ValidatePayload: validateCharacterTemporaryArmorAppliedPayload},
	{Type: eventTypeAdversaryConditionChanged, Owner: event.OwnerSystem, ValidatePayload: validateAdversaryConditionChangedPayload},
	{Type: eventTypeAdversaryCreated, Owner: event.OwnerSystem, ValidatePayload: validateAdversaryCreatedPayload},
	{Type: eventTypeAdversaryUpdated, Owner: event.OwnerSystem, ValidatePayload: validateAdversaryUpdatedPayload},
	{Type: eventTypeAdversaryDeleted, Owner: event.OwnerSystem, ValidatePayload: validateAdversaryDeletedPayload},
}

// RegisterCommands registers Daggerheart system commands.
func (m *Module) RegisterCommands(registry *command.Registry) error {
	if registry == nil {
		return errors.New("command registry is required")
	}
	for _, definition := range daggerheartCommandDefinitions {
		if err := registry.Register(definition); err != nil {
			return err
		}
	}
	return nil
}

// RegisterEvents registers Daggerheart system events.
func (m *Module) RegisterEvents(registry *event.Registry) error {
	if registry == nil {
		return errors.New("event registry is required")
	}
	for _, definition := range daggerheartEventDefinitions {
		if err := registry.Register(definition); err != nil {
			return err
		}
	}
	return nil
}

// Decider returns the system decider.
func (m *Module) Decider() system.Decider {
	return m.decider
}

// Projector returns the system projector.
func (m *Module) Projector() system.Projector {
	return m.projector
}

// StateFactory returns the state factory.
func (m *Module) StateFactory() system.StateFactory {
	return m.factory
}

func validateGMFearSetPayload(raw json.RawMessage) error {
	var payload GMFearSetPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if payload.After == nil {
		return errors.New("after is required")
	}
	if *payload.After < GMFearMin || *payload.After > GMFearMax {
		return fmt.Errorf("after must be in range %d..%d", GMFearMin, GMFearMax)
	}
	return nil
}

func validateGMFearChangedPayload(raw json.RawMessage) error {
	var payload GMFearChangedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if payload.After < GMFearMin || payload.After > GMFearMax {
		return fmt.Errorf("after must be in range %d..%d", GMFearMin, GMFearMax)
	}
	if payload.Before == payload.After {
		return errors.New("before and after must differ")
	}
	return nil
}

func validateCharacterStatePatchPayload(raw json.RawMessage) error {
	var payload CharacterStatePatchPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID) == "" {
		return errors.New("character_id is required")
	}
	if !hasCharacterStateChange(payload) {
		return errors.New("character_state patch must change at least one field")
	}
	return nil
}

func validateCharacterStatePatchedPayload(raw json.RawMessage) error {
	return validateCharacterStatePatchPayload(raw)
}

func validateHopeSpendPayload(raw json.RawMessage) error {
	var payload HopeSpendPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID) == "" {
		return errors.New("character_id is required")
	}
	if payload.Amount <= 0 {
		return errors.New("amount must be greater than zero")
	}
	if payload.Before == payload.After {
		return errors.New("before and after must differ")
	}
	if abs(payload.Before-payload.After) != payload.Amount {
		return errors.New("amount must match before and after delta")
	}
	return nil
}

func validateStressSpendPayload(raw json.RawMessage) error {
	var payload StressSpendPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID) == "" {
		return errors.New("character_id is required")
	}
	if payload.Amount <= 0 {
		return errors.New("amount must be greater than zero")
	}
	if payload.Before == payload.After {
		return errors.New("before and after must differ")
	}
	if abs(payload.Before-payload.After) != payload.Amount {
		return errors.New("amount must match before and after delta")
	}
	return nil
}

func validateConditionChangePayload(raw json.RawMessage) error {
	var payload ConditionChangePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID) == "" {
		return errors.New("character_id is required")
	}
	return validateConditionSetPayload(
		payload.ConditionsBefore,
		payload.ConditionsAfter,
		payload.Added,
		payload.Removed,
	)
}

func validateConditionChangedPayload(raw json.RawMessage) error {
	return validateConditionChangePayload(raw)
}

func validateLoadoutSwapPayload(raw json.RawMessage) error {
	var payload LoadoutSwapPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(payload.CardID) == "" {
		return errors.New("card_id is required")
	}
	return nil
}

func validateLoadoutSwappedPayload(raw json.RawMessage) error {
	return validateLoadoutSwapPayload(raw)
}

func validateRestTakePayload(raw json.RawMessage) error {
	var payload RestTakePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.RestType) == "" {
		return errors.New("rest_type is required")
	}
	if !hasRestTakeMutation(payload) {
		return errors.New("rest.take must change at least one field")
	}
	return nil
}

func validateRestTakenPayload(raw json.RawMessage) error {
	return validateRestTakePayload(raw)
}

func validateCountdownCreatePayload(raw json.RawMessage) error {
	var payload CountdownCreatePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CountdownID) == "" {
		return errors.New("countdown_id is required")
	}
	if strings.TrimSpace(payload.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(payload.Kind) == "" {
		return errors.New("kind is required")
	}
	if strings.TrimSpace(payload.Direction) == "" {
		return errors.New("direction is required")
	}
	if payload.Max <= 0 {
		return errors.New("max must be positive")
	}
	if payload.Current < 0 || payload.Current > payload.Max {
		return fmt.Errorf("current must be in range 0..%d", payload.Max)
	}
	return nil
}

func validateCountdownCreatedPayload(raw json.RawMessage) error {
	return validateCountdownCreatePayload(raw)
}

func validateCountdownUpdatePayload(raw json.RawMessage) error {
	var payload CountdownUpdatePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CountdownID) == "" {
		return errors.New("countdown_id is required")
	}
	if payload.Before == payload.After && payload.Delta == 0 {
		return errors.New("countdown update must change value")
	}
	return nil
}

func validateCountdownUpdatedPayload(raw json.RawMessage) error {
	return validateCountdownUpdatePayload(raw)
}

func validateCountdownDeletePayload(raw json.RawMessage) error {
	var payload CountdownDeletePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CountdownID) == "" {
		return errors.New("countdown_id is required")
	}
	return nil
}

func validateCountdownDeletedPayload(raw json.RawMessage) error {
	return validateCountdownDeletePayload(raw)
}

func validateDamageApplyPayload(raw json.RawMessage) error {
	var payload DamageApplyPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID) == "" {
		return errors.New("character_id is required")
	}
	if !hasDamagePatchMutation(payload.HpBefore, payload.HpAfter, payload.ArmorBefore, payload.ArmorAfter) {
		return errors.New("damage apply must change hp or armor")
	}
	return nil
}

func validateDamageAppliedPayload(raw json.RawMessage) error {
	return validateDamageApplyPayload(raw)
}

func validateAdversaryDamageApplyPayload(raw json.RawMessage) error {
	var payload AdversaryDamageApplyPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.AdversaryID) == "" {
		return errors.New("adversary_id is required")
	}
	if !hasDamagePatchMutation(payload.HpBefore, payload.HpAfter, payload.ArmorBefore, payload.ArmorAfter) {
		return errors.New("damage apply must change hp or armor")
	}
	return nil
}

func validateAdversaryDamageAppliedPayload(raw json.RawMessage) error {
	return validateAdversaryDamageApplyPayload(raw)
}

func validateDowntimeMoveApplyPayload(raw json.RawMessage) error {
	var payload DowntimeMoveApplyPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(payload.Move) == "" {
		return errors.New("move is required")
	}
	if !hasIntFieldChange(payload.HopeBefore, payload.HopeAfter) &&
		!hasIntFieldChange(payload.StressBefore, payload.StressAfter) &&
		!hasIntFieldChange(payload.ArmorBefore, payload.ArmorAfter) {
		return errors.New("downtime_move must change at least one state field")
	}
	return nil
}

func validateDowntimeMoveAppliedPayload(raw json.RawMessage) error {
	return validateDowntimeMoveApplyPayload(raw)
}

func validateCharacterTemporaryArmorApplyPayload(raw json.RawMessage) error {
	var payload CharacterTemporaryArmorApplyPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(payload.Source) == "" {
		return errors.New("source is required")
	}
	if !isTemporaryArmorDuration(strings.TrimSpace(payload.Duration)) {
		return errors.New("duration must be short_rest, long_rest, session, or scene")
	}
	if payload.Amount <= 0 {
		return errors.New("amount must be greater than zero")
	}
	return nil
}

func validateCharacterTemporaryArmorAppliedPayload(raw json.RawMessage) error {
	return validateCharacterTemporaryArmorApplyPayload(raw)
}

func validateAdversaryConditionChangePayload(raw json.RawMessage) error {
	var payload AdversaryConditionChangePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.AdversaryID) == "" {
		return errors.New("adversary_id is required")
	}
	return validateConditionSetPayload(
		payload.ConditionsBefore,
		payload.ConditionsAfter,
		payload.Added,
		payload.Removed,
	)
}

func validateAdversaryConditionChangedPayload(raw json.RawMessage) error {
	return validateAdversaryConditionChangePayload(raw)
}

func validateConditionSetPayload(before, after, added, removed []string) error {
	normalizedAfter, _, err := normalizeConditionListField(after, "conditions_after", true)
	if err != nil {
		return err
	}

	normalizedBefore, hasBefore, err := normalizeConditionListField(before, "conditions_before", false)
	if err != nil {
		return err
	}
	normalizedAdded, hasAdded, err := normalizeConditionListField(added, "added", false)
	if err != nil {
		return err
	}
	normalizedRemoved, hasRemoved, err := normalizeConditionListField(removed, "removed", false)
	if err != nil {
		return err
	}

	expectedAdded := normalizedAfter
	expectedRemoved := []string{}
	if hasBefore {
		expectedAdded, expectedRemoved = DiffConditions(normalizedBefore, normalizedAfter)
	}

	if !hasBefore && hasRemoved && len(normalizedRemoved) > 0 {
		return errors.New("conditions_before is required when removed are provided")
	}

	if hasAdded {
		if !ConditionsEqual(normalizedAdded, expectedAdded) {
			if hasBefore {
				return errors.New("added must match conditions_before and conditions_after diff")
			}
			return errors.New("added must match conditions_after when conditions_before is omitted")
		}
	}

	if hasRemoved && !ConditionsEqual(normalizedRemoved, expectedRemoved) {
		if hasBefore {
			return errors.New("removed must match conditions_before and conditions_after diff")
		}
		return errors.New("removed must be empty when conditions_before is omitted")
	}

	if hasBefore {
		if ConditionsEqual(normalizedBefore, normalizedAfter) &&
			len(normalizedAdded) == 0 && len(normalizedRemoved) == 0 {
			return errors.New("conditions must change")
		}
	} else if len(normalizedAfter) == 0 && len(normalizedAdded) == 0 && len(normalizedRemoved) == 0 {
		return errors.New("conditions must change")
	}

	return nil
}

func normalizeConditionListField(values []string, field string, required bool) ([]string, bool, error) {
	if values == nil {
		if required {
			return nil, false, fmt.Errorf("%s is required", field)
		}
		return nil, false, nil
	}

	normalized, err := NormalizeConditions(values)
	if err != nil {
		return nil, true, fmt.Errorf("%s: %w", field, err)
	}
	return normalized, true, nil
}
func validateAdversaryCreatePayload(raw json.RawMessage) error {
	var payload AdversaryCreatePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.AdversaryID) == "" {
		return errors.New("adversary_id is required")
	}
	if strings.TrimSpace(payload.Name) == "" {
		return errors.New("name is required")
	}
	return nil
}

func validateAdversaryCreatedPayload(raw json.RawMessage) error {
	return validateAdversaryCreatePayload(raw)
}

func validateAdversaryUpdatePayload(raw json.RawMessage) error {
	var payload AdversaryUpdatePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.AdversaryID) == "" {
		return errors.New("adversary_id is required")
	}
	if strings.TrimSpace(payload.Name) == "" {
		return errors.New("name is required")
	}
	return nil
}

func validateAdversaryUpdatedPayload(raw json.RawMessage) error {
	return validateAdversaryUpdatePayload(raw)
}

func validateAdversaryDeletePayload(raw json.RawMessage) error {
	var payload AdversaryDeletePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.AdversaryID) == "" {
		return errors.New("adversary_id is required")
	}
	return nil
}

func validateAdversaryDeletedPayload(raw json.RawMessage) error {
	return validateAdversaryDeletePayload(raw)
}

func hasCharacterStateChange(payload CharacterStatePatchPayload) bool {
	return hasIntFieldChange(payload.HPBefore, payload.HPAfter) ||
		hasIntFieldChange(payload.HopeBefore, payload.HopeAfter) ||
		hasIntFieldChange(payload.HopeMaxBefore, payload.HopeMaxAfter) ||
		hasIntFieldChange(payload.StressBefore, payload.StressAfter) ||
		hasIntFieldChange(payload.ArmorBefore, payload.ArmorAfter) ||
		hasStringFieldChange(payload.LifeStateBefore, payload.LifeStateAfter)
}

func hasConditionListMutation(before, after []string) bool {
	beforeNormalized, err := NormalizeConditions(before)
	if err != nil {
		return true
	}
	afterNormalized, err := NormalizeConditions(after)
	if err != nil {
		return true
	}
	return !ConditionsEqual(beforeNormalized, afterNormalized)
}

func hasRestCharacterStateMutation(payload RestCharacterStatePatch) bool {
	return hasIntFieldChange(payload.HopeBefore, payload.HopeAfter) ||
		hasIntFieldChange(payload.StressBefore, payload.StressAfter) ||
		hasIntFieldChange(payload.ArmorBefore, payload.ArmorAfter)
}

func hasRestTakeMutation(payload RestTakePayload) bool {
	if payload.GMFearBefore != payload.GMFearAfter ||
		payload.ShortRestsBefore != payload.ShortRestsAfter ||
		payload.RefreshRest ||
		payload.RefreshLongRest {
		return true
	}
	for _, patch := range payload.CharacterStates {
		if hasRestCharacterStateMutation(patch) {
			return true
		}
	}
	return false
}

func hasDamagePatchMutation(hpBefore, hpAfter, armorBefore, armorAfter *int) bool {
	return hasIntFieldChange(hpBefore, hpAfter) || hasIntFieldChange(armorBefore, armorAfter)
}

func isTemporaryArmorDuration(duration string) bool {
	switch duration {
	case "short_rest", "long_rest", "session", "scene":
		return true
	default:
		return false
	}
}

func hasIntFieldChange(before, after *int) bool {
	if after == nil {
		return false
	}
	if before == nil {
		return true
	}
	return *before != *after
}

func hasStringFieldChange(before, after *string) bool {
	if after == nil {
		return false
	}
	if before == nil {
		return true
	}
	return *before != *after
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

var _ system.Module = (*Module)(nil)
