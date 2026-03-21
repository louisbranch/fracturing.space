package daggerheart

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
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
		decider: NewDecider(commandTypesFromDefinitions()),
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
	{Type: commandTypeGMMoveApply, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateGMMoveApplyPayload},
	{Type: commandTypeGMFearSet, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateGMFearSetPayload},
	{Type: commandTypeCharacterProfileReplace, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateCharacterProfileReplacePayload, ActiveSession: command.BlockedDuringActiveSession()},
	{Type: commandTypeCharacterProfileDelete, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateCharacterProfileDeletePayload, ActiveSession: command.BlockedDuringActiveSession()},
	{Type: commandTypeCharacterStatePatch, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateCharacterStatePatchPayload},
	{Type: commandTypeConditionChange, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateConditionChangePayload},
	{Type: commandTypeHopeSpend, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateHopeSpendPayload},
	{Type: commandTypeStressSpend, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateStressSpendPayload},
	{Type: commandTypeLoadoutSwap, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateLoadoutSwapPayload},
	{Type: commandTypeRestTake, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateRestTakePayload},
	{Type: commandTypeCountdownCreate, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateCountdownCreatePayload},
	{Type: commandTypeCountdownUpdate, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateCountdownUpdatePayload},
	{Type: commandTypeCountdownDelete, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateCountdownDeletePayload},
	{Type: commandTypeDamageApply, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateDamageApplyPayload},
	{Type: commandTypeAdversaryDamageApply, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateAdversaryDamageApplyPayload},
	{Type: commandTypeCharacterTemporaryArmorApply, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateCharacterTemporaryArmorApplyPayload},
	{Type: commandTypeAdversaryConditionChange, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateAdversaryConditionChangePayload},
	{Type: commandTypeAdversaryCreate, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateAdversaryCreatePayload},
	{Type: commandTypeAdversaryUpdate, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateAdversaryUpdatePayload},
	{Type: commandTypeAdversaryFeatureApply, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateAdversaryFeatureApplyPayload},
	{Type: commandTypeAdversaryDelete, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateAdversaryDeletePayload},
	{Type: commandTypeEnvironmentEntityCreate, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateEnvironmentEntityCreatePayload},
	{Type: commandTypeEnvironmentEntityUpdate, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateEnvironmentEntityUpdatePayload},
	{Type: commandTypeEnvironmentEntityDelete, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateEnvironmentEntityDeletePayload},
	{Type: commandTypeMultiTargetDamageApply, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateMultiTargetDamageApplyPayload},
	{Type: commandTypeLevelUpApply, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateLevelUpApplyPayload},
	{Type: commandTypeClassFeatureApply, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateClassFeatureApplyPayload},
	{Type: commandTypeSubclassFeatureApply, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateSubclassFeatureApplyPayload},
	{Type: commandTypeBeastformTransform, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateBeastformTransformPayload},
	{Type: commandTypeBeastformDrop, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateBeastformDropPayload},
	{Type: commandTypeCompanionExperienceBegin, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateCompanionExperienceBeginPayload},
	{Type: commandTypeCompanionReturn, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateCompanionReturnPayload},
	{Type: commandTypeGoldUpdate, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateGoldUpdatePayload},
	{Type: commandTypeDomainCardAcquire, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateDomainCardAcquirePayload},
	{Type: commandTypeEquipmentSwap, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateEquipmentSwapPayload},
	{Type: commandTypeConsumableUse, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateConsumableUsePayload},
	{Type: commandTypeConsumableAcquire, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateConsumableAcquirePayload},
	{Type: commandTypeStatModifierChange, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateStatModifierChangePayload},
}

var daggerheartEventDefinitions = []event.Definition{
	{Type: EventTypeGMMoveApplied, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateGMMoveAppliedPayload, Intent: event.IntentAuditOnly},
	{Type: EventTypeGMFearChanged, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateGMFearChangedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeCharacterProfileReplaced, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateCharacterProfileReplacedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeCharacterProfileDeleted, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateCharacterProfileDeletedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeCharacterStatePatched, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateCharacterStatePatchedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeBeastformTransformed, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateBeastformTransformedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeBeastformDropped, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateBeastformDroppedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeCompanionExperienceBegun, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateCompanionExperienceBegunPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeCompanionReturned, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateCompanionReturnedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeConditionChanged, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateConditionChangedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeLoadoutSwapped, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateLoadoutSwappedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeRestTaken, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateRestTakenPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeCountdownCreated, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateCountdownCreatedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeCountdownUpdated, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateCountdownUpdatedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeCountdownDeleted, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateCountdownDeletedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeDamageApplied, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateDamageAppliedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeAdversaryDamageApplied, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateAdversaryDamageAppliedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeDowntimeMoveApplied, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateDowntimeMoveAppliedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeCharacterTemporaryArmorApplied, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateCharacterTemporaryArmorAppliedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeAdversaryConditionChanged, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateAdversaryConditionChangedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeAdversaryCreated, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateAdversaryCreatedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeAdversaryUpdated, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateAdversaryUpdatedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeAdversaryDeleted, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateAdversaryDeletedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeEnvironmentEntityCreated, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateEnvironmentEntityCreatedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeEnvironmentEntityUpdated, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateEnvironmentEntityUpdatedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeEnvironmentEntityDeleted, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateEnvironmentEntityDeletedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeLevelUpApplied, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateLevelUpAppliedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeGoldUpdated, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateGoldUpdatedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeDomainCardAcquired, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateDomainCardAcquiredPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeEquipmentSwapped, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateEquipmentSwappedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeConsumableUsed, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateConsumableUsedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeConsumableAcquired, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateConsumableAcquiredPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeStatModifierChanged, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateStatModifierChangedPayload, Intent: event.IntentProjectionAndReplay},
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

// CharacterReady evaluates Daggerheart-specific character readiness gates used
// by session.start.
func (m *Module) CharacterReady(systemState any, ch character.State) (bool, string) {
	snapshot, err := assertSnapshotState(systemState)
	if err != nil {
		return false, "daggerheart state is invalid"
	}
	profile, ok := snapshot.CharacterProfiles[ch.CharacterID]
	if !ok {
		return false, "daggerheart profile is missing"
	}
	return EvaluateCreationReadiness(profile)
}

// SessionStartBootstrap seeds Daggerheart campaign Fear when a draft campaign
// starts its first session. The seed equals the number of created PCs in the
// campaign snapshot at activation time. Later session starts intentionally
// contribute no bootstrap events so existing Fear carries over unchanged.
func (m *Module) SessionStartBootstrap(
	systemState any,
	characters map[ids.CharacterID]character.State,
	cmd command.Command,
	now time.Time,
) ([]event.Event, error) {
	snapshot, err := assertSnapshotState(systemState)
	if err != nil {
		return nil, err
	}
	if snapshot.GMFear != GMFearDefault {
		return nil, nil
	}

	pcCount := 0
	for _, ch := range characters {
		if !ch.Created || ch.Deleted || ch.Kind != character.KindPC {
			continue
		}
		pcCount++
	}
	if pcCount == GMFearDefault {
		return nil, nil
	}

	payloadJSON, err := json.Marshal(GMFearChangedPayload{
		Value:  pcCount,
		Reason: "campaign_start",
	})
	if err != nil {
		return nil, err
	}
	return []event.Event{{
		CampaignID:    cmd.CampaignID,
		Type:          EventTypeGMFearChanged,
		Timestamp:     now.UTC(),
		ActorType:     event.ActorType(cmd.ActorType),
		ActorID:       cmd.ActorID,
		SessionID:     cmd.SessionID,
		SceneID:       cmd.SceneID,
		RequestID:     cmd.RequestID,
		InvocationID:  cmd.InvocationID,
		EntityType:    "campaign",
		EntityID:      string(cmd.CampaignID),
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		CorrelationID: cmd.CorrelationID,
		CausationID:   cmd.CausationID,
		PayloadJSON:   payloadJSON,
	}}, nil
}

var _ module.Module = (*Module)(nil)
var _ module.CharacterReadinessChecker = (*Module)(nil)
var _ module.SessionStartBootstrapper = (*Module)(nil)
