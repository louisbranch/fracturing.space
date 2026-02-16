package checkpoint

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/replay"
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
