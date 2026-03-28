package daggerheart

import (
	"errors"

	daggerheartdecider "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/decider"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/validator"
)

// Module wires Daggerheart system behavior into the runtime.
type Module struct {
	decider module.Decider
	folder  module.Folder
	factory module.StateFactory
}

// NewModule creates a Daggerheart system module.
func NewModule() *Module {
	return &Module{
		decider: daggerheartdecider.NewDecider(commandTypesFromDefinitions()),
		folder:  NewFolder(),
		factory: NewStateFactory(),
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
	{Type: daggerheartdecider.CommandTypeGMMoveApply, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateGMMoveApplyPayload},
	{Type: daggerheartdecider.CommandTypeGMFearSet, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateGMFearSetPayload},
	{Type: daggerheartdecider.CommandTypeCharacterProfileReplace, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateCharacterProfileReplacePayload, ActiveSession: command.BlockedDuringActiveSession()},
	{Type: daggerheartdecider.CommandTypeCharacterProfileDelete, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateCharacterProfileDeletePayload, ActiveSession: command.BlockedDuringActiveSession()},
	{Type: daggerheartdecider.CommandTypeCharacterStatePatch, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateCharacterStatePatchPayload},
	{Type: daggerheartdecider.CommandTypeConditionChange, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateConditionChangePayload},
	{Type: daggerheartdecider.CommandTypeHopeSpend, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateHopeSpendPayload},
	{Type: daggerheartdecider.CommandTypeStressSpend, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateStressSpendPayload},
	{Type: daggerheartdecider.CommandTypeLoadoutSwap, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateLoadoutSwapPayload},
	{Type: daggerheartdecider.CommandTypeRestTake, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateRestTakePayload},
	{Type: daggerheartdecider.CommandTypeSceneCountdownCreate, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateSceneCountdownCreatePayload},
	{Type: daggerheartdecider.CommandTypeSceneCountdownAdvance, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateSceneCountdownAdvancePayload},
	{Type: daggerheartdecider.CommandTypeSceneCountdownTriggerResolve, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateSceneCountdownTriggerResolvePayload},
	{Type: daggerheartdecider.CommandTypeSceneCountdownDelete, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateSceneCountdownDeletePayload},
	{Type: daggerheartdecider.CommandTypeCampaignCountdownCreate, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateCampaignCountdownCreatePayload},
	{Type: daggerheartdecider.CommandTypeCampaignCountdownAdvance, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateCampaignCountdownAdvancePayload},
	{Type: daggerheartdecider.CommandTypeCampaignCountdownTriggerResolve, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateCampaignCountdownTriggerResolvePayload},
	{Type: daggerheartdecider.CommandTypeCampaignCountdownDelete, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateCampaignCountdownDeletePayload},
	{Type: daggerheartdecider.CommandTypeDamageApply, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateDamageApplyPayload},
	{Type: daggerheartdecider.CommandTypeAdversaryDamageApply, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateAdversaryDamageApplyPayload},
	{Type: daggerheartdecider.CommandTypeCharacterTemporaryArmorApply, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateCharacterTemporaryArmorApplyPayload},
	{Type: daggerheartdecider.CommandTypeAdversaryConditionChange, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateAdversaryConditionChangePayload},
	{Type: daggerheartdecider.CommandTypeAdversaryCreate, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateAdversaryCreatePayload},
	{Type: daggerheartdecider.CommandTypeAdversaryUpdate, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateAdversaryUpdatePayload},
	{Type: daggerheartdecider.CommandTypeAdversaryFeatureApply, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateAdversaryFeatureApplyPayload},
	{Type: daggerheartdecider.CommandTypeAdversaryDelete, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateAdversaryDeletePayload},
	{Type: daggerheartdecider.CommandTypeEnvironmentEntityCreate, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateEnvironmentEntityCreatePayload},
	{Type: daggerheartdecider.CommandTypeEnvironmentEntityUpdate, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateEnvironmentEntityUpdatePayload},
	{Type: daggerheartdecider.CommandTypeEnvironmentEntityDelete, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateEnvironmentEntityDeletePayload},
	{Type: daggerheartdecider.CommandTypeMultiTargetDamageApply, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateMultiTargetDamageApplyPayload},
	{Type: daggerheartdecider.CommandTypeLevelUpApply, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateLevelUpApplyPayload},
	{Type: daggerheartdecider.CommandTypeClassFeatureApply, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateClassFeatureApplyPayload},
	{Type: daggerheartdecider.CommandTypeSubclassFeatureApply, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateSubclassFeatureApplyPayload},
	{Type: daggerheartdecider.CommandTypeBeastformTransform, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateBeastformTransformPayload},
	{Type: daggerheartdecider.CommandTypeBeastformDrop, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateBeastformDropPayload},
	{Type: daggerheartdecider.CommandTypeCompanionExperienceBegin, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateCompanionExperienceBeginPayload},
	{Type: daggerheartdecider.CommandTypeCompanionReturn, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateCompanionReturnPayload},
	{Type: daggerheartdecider.CommandTypeGoldUpdate, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateGoldUpdatePayload},
	{Type: daggerheartdecider.CommandTypeDomainCardAcquire, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateDomainCardAcquirePayload},
	{Type: daggerheartdecider.CommandTypeEquipmentSwap, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateEquipmentSwapPayload},
	{Type: daggerheartdecider.CommandTypeConsumableUse, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateConsumableUsePayload},
	{Type: daggerheartdecider.CommandTypeConsumableAcquire, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateConsumableAcquirePayload},
	{Type: daggerheartdecider.CommandTypeStatModifierChange, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateStatModifierChangePayload},
}

var daggerheartEventDefinitions = []event.Definition{
	{Type: daggerheartpayload.EventTypeGMMoveApplied, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateGMMoveAppliedPayload, Intent: event.IntentAuditOnly},
	{Type: daggerheartpayload.EventTypeGMFearChanged, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateGMFearChangedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeCharacterProfileReplaced, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateCharacterProfileReplacedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeCharacterProfileDeleted, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateCharacterProfileDeletedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeCharacterStatePatched, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateCharacterStatePatchedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeBeastformTransformed, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateBeastformTransformedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeBeastformDropped, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateBeastformDroppedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeCompanionExperienceBegun, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateCompanionExperienceBegunPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeCompanionReturned, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateCompanionReturnedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeConditionChanged, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateConditionChangedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeLoadoutSwapped, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateLoadoutSwappedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeRestTaken, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateRestTakenPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeSceneCountdownCreated, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateSceneCountdownCreatedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeSceneCountdownAdvanced, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateSceneCountdownAdvancedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeSceneCountdownTriggerResolved, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateSceneCountdownTriggerResolvedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeSceneCountdownDeleted, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateSceneCountdownDeletedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeCampaignCountdownCreated, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateCampaignCountdownCreatedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeCampaignCountdownAdvanced, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateCampaignCountdownAdvancedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeCampaignCountdownTriggerResolved, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateCampaignCountdownTriggerResolvedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeCampaignCountdownDeleted, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateCampaignCountdownDeletedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeDamageApplied, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateDamageAppliedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeAdversaryDamageApplied, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateAdversaryDamageAppliedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeDowntimeMoveApplied, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateDowntimeMoveAppliedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeCharacterTemporaryArmorApplied, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateCharacterTemporaryArmorAppliedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeAdversaryConditionChanged, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateAdversaryConditionChangedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeAdversaryCreated, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateAdversaryCreatedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeAdversaryUpdated, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateAdversaryUpdatedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeAdversaryDeleted, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateAdversaryDeletedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeEnvironmentEntityCreated, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateEnvironmentEntityCreatedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeEnvironmentEntityUpdated, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateEnvironmentEntityUpdatedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeEnvironmentEntityDeleted, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateEnvironmentEntityDeletedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeLevelUpApplied, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateLevelUpAppliedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeGoldUpdated, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateGoldUpdatedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeDomainCardAcquired, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateDomainCardAcquiredPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeEquipmentSwapped, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateEquipmentSwappedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeConsumableUsed, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateConsumableUsedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeConsumableAcquired, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateConsumableAcquiredPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeStatModifierChanged, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateStatModifierChangedPayload, Intent: event.IntentProjectionAndReplay},
}

// commandTypesFromDefinitions returns all command types from
// daggerheartCommandDefinitions. Used by Decider.DeciderHandledCommands so the
// list stays in sync with the authoritative registration slice.
func commandTypesFromDefinitions() []command.Type {
	types := make([]command.Type, len(daggerheartCommandDefinitions))
	for i, def := range daggerheartCommandDefinitions {
		types[i] = def.Type
	}
	return types
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

// EmittableEventTypes returns all event types the Daggerheart decider can emit.
func (m *Module) EmittableEventTypes() []event.Type {
	types := make([]event.Type, len(daggerheartEventDefinitions))
	for i, def := range daggerheartEventDefinitions {
		types[i] = def.Type
	}
	return types
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
func (m *Module) Decider() module.Decider {
	return m.decider
}

// Folder returns the system folder.
func (m *Module) Folder() module.Folder {
	return m.folder
}

// StateFactory returns the state factory.
func (m *Module) StateFactory() module.StateFactory {
	return m.factory
}

// BindCharacterReadiness binds the Daggerheart session-start readiness
// evaluator against the current campaign snapshot.
func (m *Module) BindCharacterReadiness(campaignID ids.CampaignID, currentByKey map[module.Key]any) (module.CharacterReadinessEvaluator, error) {
	return bindCharacterReadiness(m.factory, campaignID, currentByKey)
}

// BindSessionStartBootstrap binds the Daggerheart first-session bootstrap
// emitter against the current campaign snapshot.
func (m *Module) BindSessionStartBootstrap(campaignID ids.CampaignID, currentByKey map[module.Key]any) (module.SessionStartBootstrapEmitter, error) {
	return bindSessionStartBootstrap(m.factory, campaignID, currentByKey)
}

var _ module.Module = (*Module)(nil)
var _ module.CharacterReadinessProvider = (*Module)(nil)
var _ module.SessionStartBootstrapProvider = (*Module)(nil)
