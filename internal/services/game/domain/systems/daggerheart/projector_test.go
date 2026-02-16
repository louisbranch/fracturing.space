package daggerheart

import (
	"encoding/json"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestProjectorApplyGMFearChanged_UpdatesState(t *testing.T) {
	projector := Projector{}
	state := SnapshotState{CampaignID: "camp-1", GMFear: 2}

	payload, err := json.Marshal(GMFearChangedPayload{Before: 2, After: 5, Reason: "shift"})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	updated, err := projector.Apply(state, event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("sys.daggerheart.action.gm_fear_changed"),
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   payload,
	})
	if err != nil {
		t.Fatalf("apply event: %v", err)
	}
	snapshot, ok := updated.(SnapshotState)
	if !ok {
		t.Fatalf("expected SnapshotState, got %T", updated)
	}
	if snapshot.GMFear != 5 {
		t.Fatalf("gm fear = %d, want %d", snapshot.GMFear, 5)
	}
	if snapshot.CampaignID != "camp-1" {
		t.Fatalf("campaign id = %s, want %s", snapshot.CampaignID, "camp-1")
	}
}

func TestProjectorApplyGMFearChanged_UpdatesStateForSysPrefixedType(t *testing.T) {
	projector := Projector{}
	state := SnapshotState{CampaignID: "camp-1", GMFear: 1}

	payload, err := json.Marshal(GMFearChangedPayload{Before: 1, After: 4, Reason: "shift"})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	updated, err := projector.Apply(state, event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("sys." + SystemID + ".action.gm_fear_changed"),
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   payload,
	})
	if err != nil {
		t.Fatalf("apply event: %v", err)
	}
	snapshot, ok := updated.(SnapshotState)
	if !ok {
		t.Fatalf("expected SnapshotState, got %T", updated)
	}
	if snapshot.GMFear != 4 {
		t.Fatalf("gm fear = %d, want %d", snapshot.GMFear, 4)
	}
}
