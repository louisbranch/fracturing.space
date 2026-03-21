package daggerheart

import (
	"encoding/json"
	"errors"
	"time"

	daggerheartdecider "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/decider"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"

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
	{Type: daggerheartdecider.CommandTypeCountdownCreate, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateCountdownCreatePayload},
	{Type: daggerheartdecider.CommandTypeCountdownUpdate, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateCountdownUpdatePayload},
	{Type: daggerheartdecider.CommandTypeCountdownDelete, Owner: command.OwnerSystem, ValidatePayload: validator.ValidateCountdownDeletePayload},
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
	{Type: daggerheartpayload.EventTypeCountdownCreated, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateCountdownCreatedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeCountdownUpdated, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateCountdownUpdatedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: daggerheartpayload.EventTypeCountdownDeleted, Owner: event.OwnerSystem, ValidatePayload: validator.ValidateCountdownDeletedPayload, Intent: event.IntentProjectionAndReplay},
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

// CharacterReady evaluates Daggerheart-specific character readiness gates used
// by session.start.
func (m *Module) CharacterReady(systemState any, ch character.State) (bool, string) {
	snapshot, err := daggerheartstate.AssertSnapshotState(systemState)
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
	snapshot, err := daggerheartstate.AssertSnapshotState(systemState)
	if err != nil {
		return nil, err
	}
	if snapshot.GMFear != daggerheartstate.GMFearDefault {
		return nil, nil
	}

	pcCount := 0
	for _, ch := range characters {
		if !ch.Created || ch.Deleted || ch.Kind != character.KindPC {
			continue
		}
		pcCount++
	}
	if pcCount == daggerheartstate.GMFearDefault {
		return nil, nil
	}

	payloadJSON, err := json.Marshal(daggerheartpayload.GMFearChangedPayload{
		Value:  pcCount,
		Reason: "campaign_start",
	})
	if err != nil {
		return nil, err
	}
	return []event.Event{{
		CampaignID:    cmd.CampaignID,
		Type:          daggerheartpayload.EventTypeGMFearChanged,
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
