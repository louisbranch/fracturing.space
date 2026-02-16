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
	{Type: commandTypeAttackResolve, Owner: command.OwnerSystem, ValidatePayload: validateAttackResolvePayload},
	{Type: commandTypeReactionResolve, Owner: command.OwnerSystem, ValidatePayload: validateReactionResolvePayload},
	{Type: commandTypeAdversaryRollResolve, Owner: command.OwnerSystem, ValidatePayload: validateAdversaryRollResolvePayload},
	{Type: commandTypeAdversaryAttackResolve, Owner: command.OwnerSystem, ValidatePayload: validateAdversaryAttackResolvePayload},
	{Type: commandTypeDamageRollResolve, Owner: command.OwnerSystem, ValidatePayload: validateDamageRollResolvePayload},
	{Type: commandTypeGroupActionResolve, Owner: command.OwnerSystem, ValidatePayload: validateGroupActionResolvePayload},
	{Type: commandTypeTagTeamResolve, Owner: command.OwnerSystem, ValidatePayload: validateTagTeamResolvePayload},
	{Type: commandTypeCountdownCreate, Owner: command.OwnerSystem, ValidatePayload: validateCountdownCreatePayload},
	{Type: commandTypeCountdownUpdate, Owner: command.OwnerSystem, ValidatePayload: validateCountdownUpdatePayload},
	{Type: commandTypeCountdownDelete, Owner: command.OwnerSystem, ValidatePayload: validateCountdownDeletePayload},
	{Type: commandTypeAdversaryActionResolve, Owner: command.OwnerSystem, ValidatePayload: validateAdversaryActionResolvePayload},
	{Type: commandTypeDamageApply, Owner: command.OwnerSystem, ValidatePayload: validateDamageApplyPayload},
	{Type: commandTypeAdversaryDamageApply, Owner: command.OwnerSystem, ValidatePayload: validateAdversaryDamageApplyPayload},
	{Type: commandTypeDowntimeMoveApply, Owner: command.OwnerSystem, ValidatePayload: validateDowntimeMoveApplyPayload},
	{Type: commandTypeDeathMoveResolve, Owner: command.OwnerSystem, ValidatePayload: validateDeathMoveResolvePayload},
	{Type: commandTypeBlazeOfGloryResolve, Owner: command.OwnerSystem, ValidatePayload: validateBlazeOfGloryResolvePayload},
	{Type: commandTypeGMMoveApply, Owner: command.OwnerSystem, ValidatePayload: validateGMMoveApplyPayload},
	{Type: commandTypeAdversaryConditionChange, Owner: command.OwnerSystem, ValidatePayload: validateAdversaryConditionChangePayload},
	{Type: commandTypeAdversaryCreate, Owner: command.OwnerSystem, ValidatePayload: validateAdversaryCreatePayload},
	{Type: commandTypeAdversaryUpdate, Owner: command.OwnerSystem, ValidatePayload: validateAdversaryUpdatePayload},
	{Type: commandTypeAdversaryDelete, Owner: command.OwnerSystem, ValidatePayload: validateAdversaryDeletePayload},
}

var daggerheartEventDefinitions = []event.Definition{
	{Type: eventTypeGMFearChanged, Owner: event.OwnerSystem, ValidatePayload: validateGMFearChangedPayload},
	{Type: eventTypeCharacterStatePatched, Owner: event.OwnerSystem, ValidatePayload: validateCharacterStatePatchedPayload},
	{Type: eventTypeConditionChanged, Owner: event.OwnerSystem, ValidatePayload: validateConditionChangedPayload},
	{Type: eventTypeHopeSpent, Owner: event.OwnerSystem, ValidatePayload: validateHopeSpentPayload},
	{Type: eventTypeStressSpent, Owner: event.OwnerSystem, ValidatePayload: validateStressSpentPayload},
	{Type: eventTypeLoadoutSwapped, Owner: event.OwnerSystem, ValidatePayload: validateLoadoutSwappedPayload},
	{Type: eventTypeRestTaken, Owner: event.OwnerSystem, ValidatePayload: validateRestTakenPayload},
	{Type: eventTypeAttackResolved, Owner: event.OwnerSystem, ValidatePayload: validateAttackResolvedPayload},
	{Type: eventTypeReactionResolved, Owner: event.OwnerSystem, ValidatePayload: validateReactionResolvedPayload},
	{Type: eventTypeAdversaryRollResolved, Owner: event.OwnerSystem, ValidatePayload: validateAdversaryRollResolvedPayload},
	{Type: eventTypeAdversaryAttackResolved, Owner: event.OwnerSystem, ValidatePayload: validateAdversaryAttackResolvedPayload},
	{Type: eventTypeDamageRollResolved, Owner: event.OwnerSystem, ValidatePayload: validateDamageRollResolvedPayload},
	{Type: eventTypeGroupActionResolved, Owner: event.OwnerSystem, ValidatePayload: validateGroupActionResolvedPayload},
	{Type: eventTypeTagTeamResolved, Owner: event.OwnerSystem, ValidatePayload: validateTagTeamResolvedPayload},
	{Type: eventTypeCountdownCreated, Owner: event.OwnerSystem, ValidatePayload: validateCountdownCreatedPayload},
	{Type: eventTypeCountdownUpdated, Owner: event.OwnerSystem, ValidatePayload: validateCountdownUpdatedPayload},
	{Type: eventTypeCountdownDeleted, Owner: event.OwnerSystem, ValidatePayload: validateCountdownDeletedPayload},
	{Type: eventTypeAdversaryActionResolved, Owner: event.OwnerSystem, ValidatePayload: validateAdversaryActionResolvedPayload},
	{Type: eventTypeDamageApplied, Owner: event.OwnerSystem, ValidatePayload: validateDamageAppliedPayload},
	{Type: eventTypeAdversaryDamageApplied, Owner: event.OwnerSystem, ValidatePayload: validateAdversaryDamageAppliedPayload},
	{Type: eventTypeDowntimeMoveApplied, Owner: event.OwnerSystem, ValidatePayload: validateDowntimeMoveAppliedPayload},
	{Type: eventTypeDeathMoveResolved, Owner: event.OwnerSystem, ValidatePayload: validateDeathMoveResolvedPayload},
	{Type: eventTypeBlazeOfGloryResolved, Owner: event.OwnerSystem, ValidatePayload: validateBlazeOfGloryResolvedPayload},
	{Type: eventTypeGMMoveApplied, Owner: event.OwnerSystem, ValidatePayload: validateGMMoveAppliedPayload},
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
	return nil
}

func validateCharacterStatePatchedPayload(raw json.RawMessage) error {
	return validateCharacterStatePatchPayload(raw)
}

func validateConditionChangePayload(raw json.RawMessage) error {
	var payload ConditionChangePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID) == "" {
		return errors.New("character_id is required")
	}
	return nil
}

func validateConditionChangedPayload(raw json.RawMessage) error {
	return validateConditionChangePayload(raw)
}

func validateHopeSpendPayload(raw json.RawMessage) error {
	var payload HopeSpendPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID) == "" {
		return errors.New("character_id is required")
	}
	return nil
}

func validateHopeSpentPayload(raw json.RawMessage) error {
	return validateHopeSpendPayload(raw)
}

func validateStressSpendPayload(raw json.RawMessage) error {
	var payload StressSpendPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID) == "" {
		return errors.New("character_id is required")
	}
	return nil
}

func validateStressSpentPayload(raw json.RawMessage) error {
	return validateStressSpendPayload(raw)
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
	return nil
}

func validateRestTakenPayload(raw json.RawMessage) error {
	return validateRestTakePayload(raw)
}

func validateAttackResolvePayload(raw json.RawMessage) error {
	var payload AttackResolvePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID) == "" {
		return errors.New("character_id is required")
	}
	if payload.RollSeq == 0 {
		return errors.New("roll_seq is required")
	}
	if len(payload.Targets) == 0 {
		return errors.New("targets are required")
	}
	return nil
}

func validateAttackResolvedPayload(raw json.RawMessage) error {
	return validateAttackResolvePayload(raw)
}

func validateReactionResolvePayload(raw json.RawMessage) error {
	var payload ReactionResolvePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID) == "" {
		return errors.New("character_id is required")
	}
	if payload.RollSeq == 0 {
		return errors.New("roll_seq is required")
	}
	if strings.TrimSpace(payload.Outcome) == "" {
		return errors.New("outcome is required")
	}
	return nil
}

func validateReactionResolvedPayload(raw json.RawMessage) error {
	return validateReactionResolvePayload(raw)
}

func validateAdversaryRollResolvePayload(raw json.RawMessage) error {
	var payload AdversaryRollResolvePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.AdversaryID) == "" {
		return errors.New("adversary_id is required")
	}
	if payload.RollSeq == 0 {
		return errors.New("roll_seq is required")
	}
	return nil
}

func validateAdversaryRollResolvedPayload(raw json.RawMessage) error {
	return validateAdversaryRollResolvePayload(raw)
}

func validateAdversaryAttackResolvePayload(raw json.RawMessage) error {
	var payload AdversaryAttackResolvePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.AdversaryID) == "" {
		return errors.New("adversary_id is required")
	}
	if payload.RollSeq == 0 {
		return errors.New("roll_seq is required")
	}
	if len(payload.Targets) == 0 {
		return errors.New("targets are required")
	}
	return nil
}

func validateAdversaryAttackResolvedPayload(raw json.RawMessage) error {
	return validateAdversaryAttackResolvePayload(raw)
}

func validateDamageRollResolvePayload(raw json.RawMessage) error {
	var payload DamageRollResolvePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID) == "" {
		return errors.New("character_id is required")
	}
	if payload.RollSeq == 0 {
		return errors.New("roll_seq is required")
	}
	return nil
}

func validateDamageRollResolvedPayload(raw json.RawMessage) error {
	return validateDamageRollResolvePayload(raw)
}

func validateGroupActionResolvePayload(raw json.RawMessage) error {
	var payload GroupActionResolvePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.LeaderCharacterID) == "" {
		return errors.New("leader_character_id is required")
	}
	if payload.LeaderRollSeq == 0 {
		return errors.New("leader_roll_seq is required")
	}
	return nil
}

func validateGroupActionResolvedPayload(raw json.RawMessage) error {
	return validateGroupActionResolvePayload(raw)
}

func validateTagTeamResolvePayload(raw json.RawMessage) error {
	var payload TagTeamResolvePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.FirstCharacterID) == "" {
		return errors.New("first_character_id is required")
	}
	if strings.TrimSpace(payload.SecondCharacterID) == "" {
		return errors.New("second_character_id is required")
	}
	if strings.TrimSpace(payload.SelectedCharacterID) == "" {
		return errors.New("selected_character_id is required")
	}
	if payload.FirstRollSeq == 0 {
		return errors.New("first_roll_seq is required")
	}
	if payload.SecondRollSeq == 0 {
		return errors.New("second_roll_seq is required")
	}
	if payload.SelectedRollSeq == 0 {
		return errors.New("selected_roll_seq is required")
	}
	return nil
}

func validateTagTeamResolvedPayload(raw json.RawMessage) error {
	return validateTagTeamResolvePayload(raw)
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

func validateAdversaryActionResolvePayload(raw json.RawMessage) error {
	var payload AdversaryActionResolvePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.AdversaryID) == "" {
		return errors.New("adversary_id is required")
	}
	if payload.RollSeq == 0 {
		return errors.New("roll_seq is required")
	}
	return nil
}

func validateAdversaryActionResolvedPayload(raw json.RawMessage) error {
	return validateAdversaryActionResolvePayload(raw)
}

func validateDamageApplyPayload(raw json.RawMessage) error {
	var payload DamageApplyPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID) == "" {
		return errors.New("character_id is required")
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
	return nil
}

func validateDowntimeMoveAppliedPayload(raw json.RawMessage) error {
	return validateDowntimeMoveApplyPayload(raw)
}

func validateDeathMoveResolvePayload(raw json.RawMessage) error {
	var payload DeathMoveResolvePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(payload.Move) == "" {
		return errors.New("move is required")
	}
	if strings.TrimSpace(payload.LifeStateAfter) == "" {
		return errors.New("life_state_after is required")
	}
	return nil
}

func validateDeathMoveResolvedPayload(raw json.RawMessage) error {
	return validateDeathMoveResolvePayload(raw)
}

func validateBlazeOfGloryResolvePayload(raw json.RawMessage) error {
	var payload BlazeOfGloryResolvePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(payload.LifeStateAfter) == "" {
		return errors.New("life_state_after is required")
	}
	return nil
}

func validateBlazeOfGloryResolvedPayload(raw json.RawMessage) error {
	return validateBlazeOfGloryResolvePayload(raw)
}

func validateGMMoveApplyPayload(raw json.RawMessage) error {
	var payload GMMoveApplyPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.Move) == "" {
		return errors.New("move is required")
	}
	return nil
}

func validateGMMoveAppliedPayload(raw json.RawMessage) error {
	return validateGMMoveApplyPayload(raw)
}

func validateAdversaryConditionChangePayload(raw json.RawMessage) error {
	var payload AdversaryConditionChangePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.AdversaryID) == "" {
		return errors.New("adversary_id is required")
	}
	return nil
}

func validateAdversaryConditionChangedPayload(raw json.RawMessage) error {
	return validateAdversaryConditionChangePayload(raw)
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

var _ system.Module = (*Module)(nil)
