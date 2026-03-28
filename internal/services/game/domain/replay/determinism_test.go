package replay

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// deterministicState is a minimal aggregate used by the determinism test.
// It avoids importing domain/aggregate (which imports replay), preventing an
// import cycle.
type deterministicState struct {
	CampaignName string
	Participants []string
	Updated      bool
}

// deterministicFolder folds a small set of core event types into
// deterministicState. The logic is intentionally self-contained so the test
// does not depend on the real aggregate.Folder.
type deterministicFolder struct{}

func (f *deterministicFolder) Fold(state any, evt event.Event) (any, error) {
	s, _ := state.(deterministicState)
	switch evt.Type {
	case "campaign.created":
		var p struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(evt.PayloadJSON, &p)
		s.CampaignName = p.Name
	case "participant.joined":
		var p struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(evt.PayloadJSON, &p)
		s.Participants = append(s.Participants, p.Name)
	case "campaign.updated":
		s.Updated = true
	}
	return s, nil
}

// TestReplay_Determinism validates that replaying the same event sequence
// twice produces identical aggregate state. This is the foundational
// invariant for event-sourced systems: state must be a pure function of
// the ordered event history.
func TestReplay_Determinism(t *testing.T) {
	events := deterministicEventSequence(t)
	store := &fakeEventStore{events: events}
	folder := &deterministicFolder{}
	fixedClock := func() time.Time { return time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC) }

	// First replay.
	checkpointsA := &fakeCheckpointStore{}
	resultA, err := Replay(
		context.Background(),
		store,
		checkpointsA,
		folder,
		"camp-1",
		deterministicState{},
		Options{Clock: fixedClock},
	)
	if err != nil {
		t.Fatalf("replay A: %v", err)
	}

	// Second replay of the exact same events.
	checkpointsB := &fakeCheckpointStore{}
	resultB, err := Replay(
		context.Background(),
		store,
		checkpointsB,
		folder,
		"camp-1",
		deterministicState{},
		Options{Clock: fixedClock},
	)
	if err != nil {
		t.Fatalf("replay B: %v", err)
	}

	// Structural equality: both replays must produce identical aggregate state.
	if resultA.Applied != resultB.Applied {
		t.Fatalf("applied mismatch: A=%d B=%d", resultA.Applied, resultB.Applied)
	}
	if resultA.LastSeq != resultB.LastSeq {
		t.Fatalf("last seq mismatch: A=%d B=%d", resultA.LastSeq, resultB.LastSeq)
	}
	if !reflect.DeepEqual(resultA.State, resultB.State) {
		t.Fatal("aggregate state differs between two replays of the same event sequence")
	}

	// Sanity: state is not zero-valued — the fold actually applied events.
	stateA, ok := resultA.State.(deterministicState)
	if !ok {
		t.Fatalf("expected deterministicState, got %T", resultA.State)
	}
	if stateA.CampaignName != "Determinism Test" {
		t.Fatalf("campaign name = %q, want %q", stateA.CampaignName, "Determinism Test")
	}
	if len(stateA.Participants) != 2 {
		t.Fatalf("participant count = %d, want 2", len(stateA.Participants))
	}
}

// deterministicEventSequence produces a fixed ordered event sequence that
// exercises multiple core event types without requiring domain-package imports
// that would create import cycles.
func deterministicEventSequence(t *testing.T) []event.Event {
	t.Helper()

	campaignPayload, _ := json.Marshal(map[string]string{
		"name":        "Determinism Test",
		"locale":      "en-US",
		"game_system": "daggerheart",
		"gm_mode":     "human",
	})
	gmPayload, _ := json.Marshal(map[string]string{
		"participant_id":  "p-gm",
		"user_id":         "user-gm",
		"name":            "Game Master",
		"role":            "gm",
		"controller":      "human",
		"campaign_access": "manager",
	})
	playerPayload, _ := json.Marshal(map[string]string{
		"participant_id":  "p-player",
		"user_id":         "user-player",
		"name":            "Player One",
		"role":            "player",
		"controller":      "human",
		"campaign_access": "member",
	})
	updatePayload, _ := json.Marshal(map[string]string{
		"status": "active",
	})

	return []event.Event{
		{CampaignID: "camp-1", Seq: 1, Type: "campaign.created", PayloadJSON: campaignPayload},
		{CampaignID: "camp-1", Seq: 2, Type: "participant.joined", EntityType: "participant", EntityID: "p-gm", PayloadJSON: gmPayload},
		{CampaignID: "camp-1", Seq: 3, Type: "participant.joined", EntityType: "participant", EntityID: "p-player", PayloadJSON: playerPayload},
		{CampaignID: "camp-1", Seq: 4, Type: "campaign.updated", PayloadJSON: updatePayload},
	}
}
