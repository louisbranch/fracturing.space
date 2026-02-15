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
	if evt.Type != eventTypeGMFearChanged {
		return state, nil
	}
	var payload GMFearChangedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("decode gm_fear_changed payload: %w", err)
	}
	if payload.After < GMFearMin || payload.After > GMFearMax {
		return state, fmt.Errorf("gm fear after must be in range %d..%d", GMFearMin, GMFearMax)
	}
	current, ok := snapshotFromState(state)
	if !ok && state != nil {
		return state, fmt.Errorf("unsupported state type %T", state)
	}
	if current.CampaignID == "" {
		current.CampaignID = evt.CampaignID
	}
	current.GMFear = payload.After
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
