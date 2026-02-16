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
}
