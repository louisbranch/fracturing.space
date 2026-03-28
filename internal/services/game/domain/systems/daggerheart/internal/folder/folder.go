package folder

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

// LevelUpApplier applies level-up progression to a character profile.
type LevelUpApplier func(*daggerheartstate.CharacterProfile, payload.LevelUpAppliedPayload)

// Folder folds Daggerheart system events into snapshot state.
type Folder struct {
	// Router is exported so root-package tests can verify registration
	// consistency via the type alias.
	Router       *module.FoldRouter[*daggerheartstate.SnapshotState]
	applyLevelUp LevelUpApplier
}

// NewFolder creates a Folder with all fold handlers registered.
func NewFolder(applyLevelUp LevelUpApplier) *Folder {
	f := &Folder{applyLevelUp: applyLevelUp}
	router := module.NewFoldRouter(daggerheartstate.RequireSnapshotState)
	f.registerFoldHandlers(router)
	f.Router = router
	return f
}

// FoldHandledTypes returns the event types this folder's Fold handles.
func (f *Folder) FoldHandledTypes() []event.Type {
	return f.Router.FoldHandledTypes()
}

// Fold folds a Daggerheart event into system state. It delegates to the
// FoldRouter after ensuring the snapshot CampaignID is populated from the
// event envelope.
func (f *Folder) Fold(state any, evt event.Event) (any, error) {
	s, err := daggerheartstate.RequireSnapshotState(state)
	if err != nil {
		return nil, err
	}
	if s.CampaignID == "" {
		s.CampaignID = ids.CampaignID(evt.CampaignID)
	}
	return f.Router.Fold(s, evt)
}

// registerFoldHandlers registers all Daggerheart fold handlers on the router.
func (f *Folder) registerFoldHandlers(r *module.FoldRouter[*daggerheartstate.SnapshotState]) {
	module.HandleFold(r, payload.EventTypeGMFearChanged, f.foldGMFearChanged)
	module.HandleFold(r, payload.EventTypeCharacterProfileReplaced, f.foldCharacterProfileReplaced)
	module.HandleFold(r, payload.EventTypeCharacterProfileDeleted, f.foldCharacterProfileDeleted)
	module.HandleFold(r, payload.EventTypeCharacterStatePatched, f.foldCharacterStatePatched)
	module.HandleFold(r, payload.EventTypeBeastformTransformed, f.foldBeastformTransformed)
	module.HandleFold(r, payload.EventTypeBeastformDropped, f.foldBeastformDropped)
	module.HandleFold(r, payload.EventTypeCompanionExperienceBegun, f.foldCompanionExperienceBegun)
	module.HandleFold(r, payload.EventTypeCompanionReturned, f.foldCompanionReturned)
	module.HandleFold(r, payload.EventTypeConditionChanged, f.foldConditionChanged)
	module.HandleFold(r, payload.EventTypeLoadoutSwapped, f.foldLoadoutSwapped)
	module.HandleFold(r, payload.EventTypeCharacterTemporaryArmorApplied, f.foldCharacterTemporaryArmorApplied)
	module.HandleFold(r, payload.EventTypeRestTaken, f.foldRestTaken)
	module.HandleFold(r, payload.EventTypeSceneCountdownCreated, f.foldSceneCountdownCreated)
	module.HandleFold(r, payload.EventTypeSceneCountdownAdvanced, f.foldSceneCountdownAdvanced)
	module.HandleFold(r, payload.EventTypeSceneCountdownTriggerResolved, f.foldSceneCountdownTriggerResolved)
	module.HandleFold(r, payload.EventTypeSceneCountdownDeleted, f.foldSceneCountdownDeleted)
	module.HandleFold(r, payload.EventTypeCampaignCountdownCreated, f.foldCampaignCountdownCreated)
	module.HandleFold(r, payload.EventTypeCampaignCountdownAdvanced, f.foldCampaignCountdownAdvanced)
	module.HandleFold(r, payload.EventTypeCampaignCountdownTriggerResolved, f.foldCampaignCountdownTriggerResolved)
	module.HandleFold(r, payload.EventTypeCampaignCountdownDeleted, f.foldCampaignCountdownDeleted)
	module.HandleFold(r, payload.EventTypeDamageApplied, f.foldDamageApplied)
	module.HandleFold(r, payload.EventTypeAdversaryDamageApplied, f.foldAdversaryDamageApplied)
	module.HandleFold(r, payload.EventTypeDowntimeMoveApplied, f.foldDowntimeMoveApplied)
	module.HandleFold(r, payload.EventTypeAdversaryConditionChanged, f.foldAdversaryConditionChanged)
	module.HandleFold(r, payload.EventTypeAdversaryCreated, f.foldAdversaryCreated)
	module.HandleFold(r, payload.EventTypeAdversaryUpdated, f.foldAdversaryUpdated)
	module.HandleFold(r, payload.EventTypeAdversaryDeleted, f.foldAdversaryDeleted)
	module.HandleFold(r, payload.EventTypeEnvironmentEntityCreated, f.foldEnvironmentEntityCreated)
	module.HandleFold(r, payload.EventTypeEnvironmentEntityUpdated, f.foldEnvironmentEntityUpdated)
	module.HandleFold(r, payload.EventTypeEnvironmentEntityDeleted, f.foldEnvironmentEntityDeleted)
	module.HandleFold(r, payload.EventTypeLevelUpApplied, f.foldLevelUpApplied)
	module.HandleFold(r, payload.EventTypeGoldUpdated, f.foldGoldUpdated)
	module.HandleFold(r, payload.EventTypeDomainCardAcquired, f.foldDomainCardAcquired)
	module.HandleFold(r, payload.EventTypeEquipmentSwapped, f.foldEquipmentSwapped)
	module.HandleFold(r, payload.EventTypeConsumableUsed, f.foldConsumableUsed)
	module.HandleFold(r, payload.EventTypeConsumableAcquired, f.foldConsumableAcquired)
	module.HandleFold(r, payload.EventTypeStatModifierChanged, f.foldStatModifierChanged)
}
