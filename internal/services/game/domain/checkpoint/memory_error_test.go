package checkpoint

import (
	"context"
	"errors"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/replay"
)

func TestMemoryGet_ErrorPaths(t *testing.T) {
	t.Run("nil store", func(t *testing.T) {
		var store *Memory
		_, err := store.Get(context.Background(), "camp-1")
		if err == nil || err.Error() != "checkpoint store is required" {
			t.Fatalf("Get() error = %v, want checkpoint store is required", err)
		}
	})

	t.Run("missing campaign id", func(t *testing.T) {
		store := NewMemory()
		_, err := store.Get(context.Background(), "   ")
		if !errors.Is(err, ErrCampaignIDRequired) {
			t.Fatalf("expected ErrCampaignIDRequired, got %v", err)
		}
	})

	t.Run("context canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		store := NewMemory()
		_, err := store.Get(ctx, "camp-1")
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	})
}

func TestMemorySave_ErrorPaths(t *testing.T) {
	t.Run("nil store", func(t *testing.T) {
		var store *Memory
		err := store.Save(context.Background(), replay.Checkpoint{CampaignID: "camp-1"})
		if err == nil || err.Error() != "checkpoint store is required" {
			t.Fatalf("Save() error = %v, want checkpoint store is required", err)
		}
	})

	t.Run("missing campaign id", func(t *testing.T) {
		store := NewMemory()
		err := store.Save(context.Background(), replay.Checkpoint{CampaignID: " "})
		if !errors.Is(err, ErrCampaignIDRequired) {
			t.Fatalf("expected ErrCampaignIDRequired, got %v", err)
		}
	})

	t.Run("context canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		store := NewMemory()
		err := store.Save(ctx, replay.Checkpoint{CampaignID: "camp-1"})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	})
}

func TestMemoryGetState_ErrorPaths(t *testing.T) {
	t.Run("nil store", func(t *testing.T) {
		var store *Memory
		_, _, err := store.GetState(context.Background(), "camp-1")
		if err == nil || err.Error() != "checkpoint store is required" {
			t.Fatalf("GetState() error = %v, want checkpoint store is required", err)
		}
	})

	t.Run("missing campaign id", func(t *testing.T) {
		store := NewMemory()
		_, _, err := store.GetState(context.Background(), " ")
		if !errors.Is(err, ErrCampaignIDRequired) {
			t.Fatalf("expected ErrCampaignIDRequired, got %v", err)
		}
	})

	t.Run("context canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		store := NewMemory()
		_, _, err := store.GetState(ctx, "camp-1")
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	})
}

func TestMemorySaveState_ErrorPaths(t *testing.T) {
	t.Run("nil store", func(t *testing.T) {
		var store *Memory
		err := store.SaveState(context.Background(), "camp-1", 1, "state")
		if err == nil || err.Error() != "checkpoint store is required" {
			t.Fatalf("SaveState() error = %v, want checkpoint store is required", err)
		}
	})

	t.Run("missing campaign id", func(t *testing.T) {
		store := NewMemory()
		err := store.SaveState(context.Background(), " ", 1, "state")
		if !errors.Is(err, ErrCampaignIDRequired) {
			t.Fatalf("expected ErrCampaignIDRequired, got %v", err)
		}
	})

	t.Run("context canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		store := NewMemory()
		err := store.SaveState(ctx, "camp-1", 1, "state")
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	})
}
