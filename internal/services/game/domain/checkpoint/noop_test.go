package checkpoint

import (
	"context"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/replay"
)

func TestNoop_GetAndSave(t *testing.T) {
	store := NewNoop()
	if store == nil {
		t.Fatal("expected store")
	}

	_, err := store.Get(context.Background(), "camp-1")
	if err != replay.ErrCheckpointNotFound {
		t.Fatalf("get error = %v, want %v", err, replay.ErrCheckpointNotFound)
	}
	if err := store.Save(context.Background(), replay.Checkpoint{CampaignID: "camp-1", LastSeq: 1}); err != nil {
		t.Fatalf("save: %v", err)
	}
}

func TestNoop_GetStateAndSaveState(t *testing.T) {
	store := NewNoop()

	_, _, err := store.GetState(context.Background(), "camp-1")
	if err != replay.ErrCheckpointNotFound {
		t.Fatalf("get state error = %v, want %v", err, replay.ErrCheckpointNotFound)
	}
	if err := store.SaveState(context.Background(), "camp-1", 5, "some-state"); err != nil {
		t.Fatalf("save state: %v", err)
	}
	// Invariant: SaveState is a no-op, so GetState still returns not found.
	_, _, err = store.GetState(context.Background(), "camp-1")
	if err != replay.ErrCheckpointNotFound {
		t.Fatalf("get state after save error = %v, want %v", err, replay.ErrCheckpointNotFound)
	}
}

func TestNoop_RespectsContextErrors(t *testing.T) {
	store := NewNoop()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := store.Get(ctx, "camp-1"); err == nil {
		t.Fatal("expected context error")
	}
	if err := store.Save(ctx, replay.Checkpoint{CampaignID: "camp-1", LastSeq: 1}); err == nil {
		t.Fatal("expected context error")
	}
	if _, _, err := store.GetState(ctx, "camp-1"); err == nil {
		t.Fatal("expected context error from GetState")
	}
	if err := store.SaveState(ctx, "camp-1", 1, nil); err == nil {
		t.Fatal("expected context error from SaveState")
	}
}
