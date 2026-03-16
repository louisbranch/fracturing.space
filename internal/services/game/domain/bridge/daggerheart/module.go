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
		decider: Decider{},
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
	{Type: commandTypeGMMoveApply, Owner: command.OwnerSystem, ValidatePayload: validateGMMoveApplyPayload},
	{Type: commandTypeGMFearSet, Owner: command.OwnerSystem, ValidatePayload: validateGMFearSetPayload},
	{Type: commandTypeCharacterProfileReplace, Owner: command.OwnerSystem, ValidatePayload: validateCharacterProfileReplacePayload, ActiveSession: command.BlockedDuringActiveSession()},
	{Type: commandTypeCharacterProfileDelete, Owner: command.OwnerSystem, ValidatePayload: validateCharacterProfileDeletePayload, ActiveSession: command.BlockedDuringActiveSession()},
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
	{Type: commandTypeCharacterTemporaryArmorApply, Owner: command.OwnerSystem, ValidatePayload: validateCharacterTemporaryArmorApplyPayload},
	{Type: commandTypeAdversaryConditionChange, Owner: command.OwnerSystem, ValidatePayload: validateAdversaryConditionChangePayload},
	{Type: commandTypeAdversaryCreate, Owner: command.OwnerSystem, ValidatePayload: validateAdversaryCreatePayload},
	{Type: commandTypeAdversaryUpdate, Owner: command.OwnerSystem, ValidatePayload: validateAdversaryUpdatePayload},
	{Type: commandTypeAdversaryFeatureApply, Owner: command.OwnerSystem, ValidatePayload: validateAdversaryFeatureApplyPayload},
	{Type: commandTypeAdversaryDelete, Owner: command.OwnerSystem, ValidatePayload: validateAdversaryDeletePayload},
	{Type: commandTypeEnvironmentEntityCreate, Owner: command.OwnerSystem, ValidatePayload: validateEnvironmentEntityCreatePayload},
	{Type: commandTypeEnvironmentEntityUpdate, Owner: command.OwnerSystem, ValidatePayload: validateEnvironmentEntityUpdatePayload},
	{Type: commandTypeEnvironmentEntityDelete, Owner: command.OwnerSystem, ValidatePayload: validateEnvironmentEntityDeletePayload},
	{Type: commandTypeMultiTargetDamageApply, Owner: command.OwnerSystem, ValidatePayload: validateMultiTargetDamageApplyPayload},
	{Type: commandTypeLevelUpApply, Owner: command.OwnerSystem, ValidatePayload: validateLevelUpApplyPayload},
	{Type: commandTypeClassFeatureApply, Owner: command.OwnerSystem, ValidatePayload: validateClassFeatureApplyPayload},
	{Type: commandTypeSubclassFeatureApply, Owner: command.OwnerSystem, ValidatePayload: validateSubclassFeatureApplyPayload},
	{Type: commandTypeBeastformTransform, Owner: command.OwnerSystem, ValidatePayload: validateBeastformTransformPayload},
	{Type: commandTypeBeastformDrop, Owner: command.OwnerSystem, ValidatePayload: validateBeastformDropPayload},
	{Type: commandTypeCompanionExperienceBegin, Owner: command.OwnerSystem, ValidatePayload: validateCompanionExperienceBeginPayload},
	{Type: commandTypeCompanionReturn, Owner: command.OwnerSystem, ValidatePayload: validateCompanionReturnPayload},
	{Type: commandTypeGoldUpdate, Owner: command.OwnerSystem, ValidatePayload: validateGoldUpdatePayload},
	{Type: commandTypeDomainCardAcquire, Owner: command.OwnerSystem, ValidatePayload: validateDomainCardAcquirePayload},
	{Type: commandTypeEquipmentSwap, Owner: command.OwnerSystem, ValidatePayload: validateEquipmentSwapPayload},
	{Type: commandTypeConsumableUse, Owner: command.OwnerSystem, ValidatePayload: validateConsumableUsePayload},
	{Type: commandTypeConsumableAcquire, Owner: command.OwnerSystem, ValidatePayload: validateConsumableAcquirePayload},
}

var daggerheartEventDefinitions = []event.Definition{
	{Type: EventTypeGMMoveApplied, Owner: event.OwnerSystem, ValidatePayload: validateGMMoveAppliedPayload, Intent: event.IntentAuditOnly},
	{Type: EventTypeGMFearChanged, Owner: event.OwnerSystem, ValidatePayload: validateGMFearChangedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeCharacterProfileReplaced, Owner: event.OwnerSystem, ValidatePayload: validateCharacterProfileReplacedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeCharacterProfileDeleted, Owner: event.OwnerSystem, ValidatePayload: validateCharacterProfileDeletedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeCharacterStatePatched, Owner: event.OwnerSystem, ValidatePayload: validateCharacterStatePatchedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeBeastformTransformed, Owner: event.OwnerSystem, ValidatePayload: validateBeastformTransformedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeBeastformDropped, Owner: event.OwnerSystem, ValidatePayload: validateBeastformDroppedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeCompanionExperienceBegun, Owner: event.OwnerSystem, ValidatePayload: validateCompanionExperienceBegunPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeCompanionReturned, Owner: event.OwnerSystem, ValidatePayload: validateCompanionReturnedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeConditionChanged, Owner: event.OwnerSystem, ValidatePayload: validateConditionChangedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeLoadoutSwapped, Owner: event.OwnerSystem, ValidatePayload: validateLoadoutSwappedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeRestTaken, Owner: event.OwnerSystem, ValidatePayload: validateRestTakenPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeCountdownCreated, Owner: event.OwnerSystem, ValidatePayload: validateCountdownCreatedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeCountdownUpdated, Owner: event.OwnerSystem, ValidatePayload: validateCountdownUpdatedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeCountdownDeleted, Owner: event.OwnerSystem, ValidatePayload: validateCountdownDeletedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeDamageApplied, Owner: event.OwnerSystem, ValidatePayload: validateDamageAppliedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeAdversaryDamageApplied, Owner: event.OwnerSystem, ValidatePayload: validateAdversaryDamageAppliedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeDowntimeMoveApplied, Owner: event.OwnerSystem, ValidatePayload: validateDowntimeMoveAppliedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeCharacterTemporaryArmorApplied, Owner: event.OwnerSystem, ValidatePayload: validateCharacterTemporaryArmorAppliedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeAdversaryConditionChanged, Owner: event.OwnerSystem, ValidatePayload: validateAdversaryConditionChangedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeAdversaryCreated, Owner: event.OwnerSystem, ValidatePayload: validateAdversaryCreatedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeAdversaryUpdated, Owner: event.OwnerSystem, ValidatePayload: validateAdversaryUpdatedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeAdversaryDeleted, Owner: event.OwnerSystem, ValidatePayload: validateAdversaryDeletedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeEnvironmentEntityCreated, Owner: event.OwnerSystem, ValidatePayload: validateEnvironmentEntityCreatedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeEnvironmentEntityUpdated, Owner: event.OwnerSystem, ValidatePayload: validateEnvironmentEntityUpdatedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeEnvironmentEntityDeleted, Owner: event.OwnerSystem, ValidatePayload: validateEnvironmentEntityDeletedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeLevelUpApplied, Owner: event.OwnerSystem, ValidatePayload: validateLevelUpAppliedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeGoldUpdated, Owner: event.OwnerSystem, ValidatePayload: validateGoldUpdatedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeDomainCardAcquired, Owner: event.OwnerSystem, ValidatePayload: validateDomainCardAcquiredPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeEquipmentSwapped, Owner: event.OwnerSystem, ValidatePayload: validateEquipmentSwappedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeConsumableUsed, Owner: event.OwnerSystem, ValidatePayload: validateConsumableUsedPayload, Intent: event.IntentProjectionAndReplay},
	{Type: EventTypeConsumableAcquired, Owner: event.OwnerSystem, ValidatePayload: validateConsumableAcquiredPayload, Intent: event.IntentProjectionAndReplay},
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
