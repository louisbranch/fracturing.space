package checkpoint

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/replay"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/system"
)

func TestMemoryCheckpoint_SaveAndGet(t *testing.T) {
	store := NewMemory()
	checkpoint := replay.Checkpoint{
		CampaignID: "camp-1",
		LastSeq:    42,
		UpdatedAt:  time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
	}

	if err := store.Save(context.Background(), checkpoint); err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}
	loaded, err := store.Get(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("get checkpoint: %v", err)
	}
	if loaded.LastSeq != checkpoint.LastSeq {
		t.Fatalf("last seq = %d, want %d", loaded.LastSeq, checkpoint.LastSeq)
	}
}

func TestMemoryCheckpoint_GetMissingReturnsNotFound(t *testing.T) {
	store := NewMemory()
	_, err := store.Get(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if err != replay.ErrCheckpointNotFound {
		t.Fatalf("error = %v, want %v", err, replay.ErrCheckpointNotFound)
	}
}

func TestMemoryCheckpoint_SaveAndGetState(t *testing.T) {
	store := NewMemory()
	source := aggregate.State{
		Session: session.State{
			GateOpen: true,
			GateID:   "gate-1",
		},
		Participants: map[string]participant.State{
			"p1": {Joined: true},
		},
		Characters: map[string]character.State{
			"c1": {Created: true},
		},
		Invites: map[string]invite.State{
			"i1": {Created: true},
		},
		Systems: map[system.Key]any{
			{ID: "system-1", Version: "v1"}: map[string]any{"value": 1},
		},
	}

	if err := store.SaveState(context.Background(), "camp-1", 7, source); err != nil {
		t.Fatalf("save state: %v", err)
	}
	state, seq, err := store.GetState(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if seq != 7 {
		t.Fatalf("seq = %d, want %d", seq, 7)
	}
	loaded, ok := state.(aggregate.State)
	if !ok {
		t.Fatalf("state type = %T, want aggregate.State", state)
	}
	if !loaded.Session.GateOpen || loaded.Session.GateID != "gate-1" {
		t.Fatalf("unexpected session state: %+v", loaded.Session)
	}
	if _, ok := loaded.Participants["p1"]; !ok {
		t.Fatal("expected participant p1")
	}

	loaded.Participants["p2"] = participant.State{Joined: true}
	stateAgain, _, err := store.GetState(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("get state again: %v", err)
	}
	loadedAgain, ok := stateAgain.(aggregate.State)
	if !ok {
		t.Fatalf("state type = %T, want aggregate.State", stateAgain)
	}
	if _, ok := loadedAgain.Participants["p2"]; ok {
		t.Fatal("expected stored state to be isolated from caller mutations")
	}
}

func TestMemoryCheckpoint_GetStateMissingReturnsNotFound(t *testing.T) {
	store := NewMemory()
	_, _, err := store.GetState(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if err != replay.ErrCheckpointNotFound {
		t.Fatalf("error = %v, want %v", err, replay.ErrCheckpointNotFound)
	}
}

func TestMemoryCheckpoint_SaveAndGetStatePointerInput(t *testing.T) {
	store := NewMemory()
	source := &aggregate.State{
		Session: session.State{GateID: "gate-1"},
	}

	if err := store.SaveState(context.Background(), "camp-1", 3, source); err != nil {
		t.Fatalf("save state: %v", err)
	}
	state, seq, err := store.GetState(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if seq != 3 {
		t.Fatalf("seq = %d, want %d", seq, 3)
	}
	loaded, ok := state.(aggregate.State)
	if !ok {
		t.Fatalf("state type = %T, want aggregate.State", state)
	}
	if loaded.Session.GateID != "gate-1" {
		t.Fatalf("gate id = %q, want %q", loaded.Session.GateID, "gate-1")
	}
}

func TestMemoryCheckpoint_SaveAndGetStateNonAggregate(t *testing.T) {
	store := NewMemory()
	if err := store.SaveState(context.Background(), "camp-1", 2, "plain-state"); err != nil {
		t.Fatalf("save state: %v", err)
	}
	state, seq, err := store.GetState(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if seq != 2 {
		t.Fatalf("seq = %d, want %d", seq, 2)
	}
	value, ok := state.(string)
	if !ok {
		t.Fatalf("state type = %T, want string", state)
	}
	if value != "plain-state" {
		t.Fatalf("state = %q, want %q", value, "plain-state")
	}
}
