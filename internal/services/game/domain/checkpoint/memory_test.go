package checkpoint

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/replay"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
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
		Participants: map[ids.ParticipantID]participant.State{
			"p1": {Joined: true},
		},
		Characters: map[ids.CharacterID]character.State{
			"c1": {Created: true},
		},
		Invites: map[ids.InviteID]invite.State{
			"i1": {Created: true},
		},
		Systems: map[module.Key]any{
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

func TestMemoryCheckpoint_SaveAndGetState_ScenesIsolation(t *testing.T) {
	store := NewMemory()
	source := aggregate.State{
		Scenes: map[ids.SceneID]scene.State{
			"scene-1": {
				SceneID: "scene-1",
				Active:  true,
				Characters: map[ids.CharacterID]bool{
					"c1": true,
				},
				PlayerPhaseActingCharacters: []ids.CharacterID{"c1", "c2"},
				PlayerPhaseActingParticipants: map[ids.ParticipantID]bool{
					"p1": true,
				},
				PlayerPhaseSlots: map[ids.ParticipantID]scene.PlayerPhaseSlot{
					"p1": {
						ParticipantID:      "p1",
						CharacterIDs:       []ids.CharacterID{"c1"},
						Yielded:            true,
						ReviewCharacterIDs: []ids.CharacterID{"c2"},
					},
				},
			},
		},
	}

	if err := store.SaveState(context.Background(), "camp-1", 5, source); err != nil {
		t.Fatalf("save state: %v", err)
	}

	state, _, err := store.GetState(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	loaded := state.(aggregate.State)

	// Mutate the loaded scene's Characters map.
	loaded.Scenes["scene-1"].Characters["c2"] = true

	// Verify stored state is isolated from the mutation.
	stateAgain, _, err := store.GetState(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("get state again: %v", err)
	}
	loadedAgain := stateAgain.(aggregate.State)
	if _, ok := loadedAgain.Scenes["scene-1"].Characters["c2"]; ok {
		t.Fatal("expected stored scene state to be isolated from caller mutations")
	}
	if len(loadedAgain.Scenes["scene-1"].Characters) != 1 {
		t.Fatalf("scene characters count = %d, want 1", len(loadedAgain.Scenes["scene-1"].Characters))
	}

	// Verify deep clone of acting characters slice.
	sc := loadedAgain.Scenes["scene-1"]
	if len(sc.PlayerPhaseActingCharacters) != 2 {
		t.Fatalf("acting characters count = %d, want 2", len(sc.PlayerPhaseActingCharacters))
	}

	// Verify slot deep clone.
	slot := sc.PlayerPhaseSlots["p1"]
	if !slot.Yielded {
		t.Fatal("expected slot to be yielded")
	}
	if len(slot.CharacterIDs) != 1 || slot.CharacterIDs[0] != "c1" {
		t.Fatalf("slot character ids = %v, want [c1]", slot.CharacterIDs)
	}
	if len(slot.ReviewCharacterIDs) != 1 || slot.ReviewCharacterIDs[0] != "c2" {
		t.Fatalf("slot review character ids = %v, want [c2]", slot.ReviewCharacterIDs)
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

func TestMemoryCheckpoint_SaveAndGetStateNilPointerInput(t *testing.T) {
	store := NewMemory()
	if err := store.SaveState(context.Background(), "camp-1", 1, (*aggregate.State)(nil)); err != nil {
		t.Fatalf("save state: %v", err)
	}
	state, seq, err := store.GetState(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if seq != 1 {
		t.Fatalf("seq = %d, want %d", seq, 1)
	}
	loaded, ok := state.(aggregate.State)
	if !ok {
		t.Fatalf("state type = %T, want aggregate.State", state)
	}
	if loaded.Campaign.Created {
		t.Fatal("expected zero-value campaign state from nil pointer input")
	}
}

func TestMemoryCheckpoint_SaveStateNonAggregateReturnsError(t *testing.T) {
	store := NewMemory()
	err := store.SaveState(context.Background(), "camp-1", 2, "plain-state")
	if err == nil {
		t.Fatal("expected error for unhandled state type")
	}
	if !strings.Contains(err.Error(), "unhandled state type") {
		t.Fatalf("error = %q, want substring %q", err.Error(), "unhandled state type")
	}
}

func TestMemoryCheckpoint_SaveStateUsesInjectedClock(t *testing.T) {
	fixed := time.Date(2026, 3, 13, 12, 0, 0, 0, time.UTC)
	store := NewMemory()
	store.Clock = func() time.Time { return fixed }

	if err := store.SaveState(context.Background(), "camp-1", 1, aggregate.State{}); err != nil {
		t.Fatalf("save state: %v", err)
	}
	checkpoint, err := store.Get(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("get checkpoint: %v", err)
	}
	if !checkpoint.UpdatedAt.Equal(fixed) {
		t.Fatalf("updated at = %v, want %v", checkpoint.UpdatedAt, fixed)
	}
}
