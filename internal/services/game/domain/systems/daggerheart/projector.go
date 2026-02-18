package daggerheart

import (
	"encoding/json"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// Projector applies Daggerheart system events to state.
type Projector struct{}

// Apply applies a Daggerheart event to state.
func (Projector) Apply(state any, evt event.Event) (any, error) {
	var fearPayload GMFearChangedPayload
	switch evt.Type {
	case eventTypeGMFearChanged:
		if err := json.Unmarshal(evt.PayloadJSON, &fearPayload); err != nil {
			return state, fmt.Errorf("decode gm_fear_changed payload: %w", err)
		}
		if fearPayload.After < GMFearMin || fearPayload.After > GMFearMax {
			return state, fmt.Errorf("gm fear after must be in range %d..%d", GMFearMin, GMFearMax)
		}
	case eventTypeCharacterStatePatched,
		eventTypeConditionChanged,
		eventTypeHopeSpent,
		eventTypeStressSpent,
		eventTypeLoadoutSwapped,
		eventTypeRestTaken,
		eventTypeAttackResolved,
		eventTypeReactionResolved,
		eventTypeAdversaryRollResolved,
		eventTypeAdversaryAttackResolved,
		eventTypeDamageRollResolved,
		eventTypeGroupActionResolved,
		eventTypeTagTeamResolved,
		eventTypeCountdownCreated,
		eventTypeCountdownUpdated,
		eventTypeCountdownDeleted,
		eventTypeAdversaryActionResolved,
		eventTypeDamageApplied,
		eventTypeAdversaryDamageApplied,
		eventTypeDowntimeMoveApplied,
		eventTypeDeathMoveResolved,
		eventTypeBlazeOfGloryResolved,
		eventTypeGMMoveApplied,
		eventTypeAdversaryConditionChanged,
		eventTypeAdversaryCreated,
		eventTypeAdversaryUpdated,
		eventTypeAdversaryDeleted:
		current, ok := snapshotFromState(state)
		if !ok && state != nil {
			return state, fmt.Errorf("unsupported state type %T", state)
		}
		return current, nil
	default:
		return nil, fmt.Errorf("unhandled daggerheart projector event type: %s", evt.Type)
	}

	current, ok := snapshotFromState(state)
	if !ok && state != nil {
		return state, fmt.Errorf("unsupported state type %T", state)
	}
	if current.CampaignID == "" {
		current.CampaignID = evt.CampaignID
	}
	current.GMFear = fearPayload.After
	return current, nil
}

func snapshotFromState(state any) (SnapshotState, bool) {
	switch typed := state.(type) {
	case SnapshotState:
		return typed, true
	case *SnapshotState:
		if typed != nil {
			return *typed, true
		}
	}
	return SnapshotState{GMFear: GMFearDefault}, false
}
