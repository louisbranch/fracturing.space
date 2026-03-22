package adapter

import (
	"context"
	"fmt"
	"strings"

	event "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

// LevelUpApplier applies level-up progression to a character profile.
type LevelUpApplier func(*daggerheartstate.CharacterProfile, payload.LevelUpAppliedPayload)

// StatePatch carries optional field overrides for a character state update.
// Named fields replace the previous 10-parameter positional signature so that
// call sites are self-documenting and review-safe.
type StatePatch struct {
	HP                            *int
	Hope                          *int
	HopeMax                       *int
	Stress                        *int
	Armor                         *int
	LifeState                     *string
	ClassState                    *daggerheartstate.CharacterClassState
	SubclassState                 *daggerheartstate.CharacterSubclassState
	CompanionState                *daggerheartstate.CharacterCompanionState
	ImpenetrableUsedThisShortRest *bool
}

// Adapter applies Daggerheart-specific events to system projections.
type Adapter struct {
	store        projectionstore.Store
	Router       *module.AdapterRouter
	applyLevelUp LevelUpApplier
}

const (
	systemID      = "daggerheart"
	systemVersion = "1.0.0"
)

// NewAdapter creates a Daggerheart adapter with all handlers registered.
func NewAdapter(store projectionstore.Store, applyLevelUp LevelUpApplier) *Adapter {
	a := &Adapter{store: store, applyLevelUp: applyLevelUp}
	a.Router = a.buildRouter()
	return a
}

// ID returns the Daggerheart system identifier.
func (a *Adapter) ID() string {
	return systemID
}

// Version returns the Daggerheart system version.
func (a *Adapter) Version() string {
	return systemVersion
}

// HandledTypes returns the event types this adapter's Apply handles.
func (a *Adapter) HandledTypes() []event.Type {
	return a.Router.HandledTypes()
}

// Apply applies a system-specific event to Daggerheart projections.
func (a *Adapter) Apply(ctx context.Context, evt event.Event) error {
	if a == nil || a.store == nil {
		return fmt.Errorf("daggerheart store is not configured")
	}
	return a.Router.Apply(ctx, evt)
}

// Snapshot loads the Daggerheart snapshot projection.
func (a *Adapter) Snapshot(ctx context.Context, campaignID string) (any, error) {
	if a == nil || a.store == nil {
		return nil, fmt.Errorf("daggerheart store is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	return a.store.GetDaggerheartSnapshot(ctx, campaignID)
}

func (a *Adapter) buildRouter() *module.AdapterRouter {
	r := module.NewAdapterRouter()
	module.HandleAdapter(r, payload.EventTypeCharacterProfileReplaced, a.HandleCharacterProfileReplaced)
	module.HandleAdapter(r, payload.EventTypeCharacterProfileDeleted, a.HandleCharacterProfileDeleted)
	module.HandleAdapter(r, payload.EventTypeDamageApplied, a.HandleDamageApplied)
	module.HandleAdapter(r, payload.EventTypeRestTaken, a.HandleRestTaken)
	module.HandleAdapter(r, payload.EventTypeCharacterTemporaryArmorApplied, a.HandleCharacterTemporaryArmorApplied)
	module.HandleAdapter(r, payload.EventTypeDowntimeMoveApplied, a.HandleDowntimeMoveApplied)
	module.HandleAdapter(r, payload.EventTypeLoadoutSwapped, a.HandleLoadoutSwapped)
	module.HandleAdapter(r, payload.EventTypeCharacterStatePatched, a.HandleCharacterStatePatched)
	module.HandleAdapter(r, payload.EventTypeBeastformTransformed, a.HandleBeastformTransformed)
	module.HandleAdapter(r, payload.EventTypeBeastformDropped, a.HandleBeastformDropped)
	module.HandleAdapter(r, payload.EventTypeCompanionExperienceBegun, a.HandleCompanionExperienceBegun)
	module.HandleAdapter(r, payload.EventTypeCompanionReturned, a.HandleCompanionReturned)
	module.HandleAdapter(r, payload.EventTypeConditionChanged, a.HandleConditionChanged)
	module.HandleAdapter(r, payload.EventTypeAdversaryConditionChanged, a.HandleAdversaryConditionChanged)
	module.HandleAdapter(r, payload.EventTypeGMFearChanged, a.HandleGMFearChanged)
	module.HandleAdapter(r, payload.EventTypeSceneCountdownCreated, a.HandleSceneCountdownCreated)
	module.HandleAdapter(r, payload.EventTypeSceneCountdownAdvanced, a.HandleSceneCountdownAdvanced)
	module.HandleAdapter(r, payload.EventTypeSceneCountdownTriggerResolved, a.HandleSceneCountdownTriggerResolved)
	module.HandleAdapter(r, payload.EventTypeSceneCountdownDeleted, a.HandleSceneCountdownDeleted)
	module.HandleAdapter(r, payload.EventTypeCampaignCountdownCreated, a.HandleCampaignCountdownCreated)
	module.HandleAdapter(r, payload.EventTypeCampaignCountdownAdvanced, a.HandleCampaignCountdownAdvanced)
	module.HandleAdapter(r, payload.EventTypeCampaignCountdownTriggerResolved, a.HandleCampaignCountdownTriggerResolved)
	module.HandleAdapter(r, payload.EventTypeCampaignCountdownDeleted, a.HandleCampaignCountdownDeleted)
	module.HandleAdapter(r, payload.EventTypeAdversaryCreated, a.HandleAdversaryCreated)
	module.HandleAdapter(r, payload.EventTypeAdversaryDamageApplied, a.HandleAdversaryDamageApplied)
	module.HandleAdapter(r, payload.EventTypeAdversaryUpdated, a.HandleAdversaryUpdated)
	module.HandleAdapter(r, payload.EventTypeAdversaryDeleted, a.HandleAdversaryDeleted)
	module.HandleAdapter(r, payload.EventTypeEnvironmentEntityCreated, a.HandleEnvironmentEntityCreated)
	module.HandleAdapter(r, payload.EventTypeEnvironmentEntityUpdated, a.HandleEnvironmentEntityUpdated)
	module.HandleAdapter(r, payload.EventTypeEnvironmentEntityDeleted, a.HandleEnvironmentEntityDeleted)
	module.HandleAdapter(r, payload.EventTypeLevelUpApplied, a.HandleLevelUpApplied)
	module.HandleAdapter(r, payload.EventTypeGoldUpdated, a.HandleGoldUpdated)
	module.HandleAdapter(r, payload.EventTypeDomainCardAcquired, a.HandleDomainCardAcquired)
	module.HandleAdapter(r, payload.EventTypeEquipmentSwapped, a.HandleEquipmentSwapped)
	module.HandleAdapter(r, payload.EventTypeConsumableUsed, a.HandleConsumableUsed)
	module.HandleAdapter(r, payload.EventTypeConsumableAcquired, a.HandleConsumableAcquired)
	module.HandleAdapter(r, payload.EventTypeStatModifierChanged, a.HandleStatModifierChanged)
	return r
}
