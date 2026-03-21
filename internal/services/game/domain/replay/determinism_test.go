package replay

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
)

// TestReplay_Determinism validates that replaying the same event sequence
// twice produces identical aggregate state. This is the foundational
// invariant for event-sourced systems: state must be a pure function of
// the ordered event history.
func TestReplay_Determinism(t *testing.T) {
	events := deterministicEventSequence(t)
	store := &fakeEventStore{events: events}
	folder := &aggregate.Folder{}
	fixedClock := func() time.Time { return time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC) }

	// First replay.
	checkpointsA := &fakeCheckpointStore{}
	resultA, err := Replay(
		context.Background(),
		store,
		checkpointsA,
		folder,
		"camp-1",
		aggregate.NewState(),
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
		aggregate.NewState(),
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
	stateA, ok := resultA.State.(aggregate.State)
	if !ok {
		t.Fatalf("expected aggregate.State, got %T", resultA.State)
	}
	if !stateA.Campaign.Created {
		t.Fatal("campaign.Created should be true after replay")
	}
	if stateA.Campaign.Name != "Determinism Test" {
		t.Fatalf("campaign name = %q, want %q", stateA.Campaign.Name, "Determinism Test")
	}
	if len(stateA.Participants) != 2 {
		t.Fatalf("participant count = %d, want 2", len(stateA.Participants))
	}
}

// deterministicEventSequence produces a fixed ordered event sequence that
// exercises multiple core domains (campaign, participant) without requiring
// system-specific modules.
func deterministicEventSequence(t *testing.T) []event.Event {
	t.Helper()

	campaignPayload, _ := json.Marshal(campaign.CreatePayload{
		Name:       "Determinism Test",
		Locale:     "en-US",
		GameSystem: "daggerheart",
		GmMode:     "human",
	})
	gmPayload, _ := json.Marshal(participant.JoinPayload{
		ParticipantID:  "p-gm",
		UserID:         "user-gm",
		Name:           "Game Master",
		Role:           "gm",
		Controller:     "human",
		CampaignAccess: "manager",
	})
	playerPayload, _ := json.Marshal(participant.JoinPayload{
		ParticipantID:  "p-player",
		UserID:         "user-player",
		Name:           "Player One",
		Role:           "player",
		Controller:     "human",
		CampaignAccess: "member",
	})
	updatePayload, _ := json.Marshal(campaign.UpdatePayload{
		Fields: map[string]string{"status": "active"},
	})

	return []event.Event{
		{CampaignID: "camp-1", Seq: 1, Type: "campaign.created", PayloadJSON: campaignPayload},
		{CampaignID: "camp-1", Seq: 2, Type: "participant.joined", EntityType: "participant", EntityID: "p-gm", PayloadJSON: gmPayload},
		{CampaignID: "camp-1", Seq: 3, Type: "participant.joined", EntityType: "participant", EntityID: "p-player", PayloadJSON: playerPayload},
		{CampaignID: "camp-1", Seq: 4, Type: "campaign.updated", PayloadJSON: updatePayload},
	}
}
