package journal

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestMemoryAppend_ErrorPaths(t *testing.T) {
	t.Run("nil journal", func(t *testing.T) {
		var memory *Memory
		_, err := memory.Append(context.Background(), event.Event{CampaignID: "camp-1"})
		if err == nil || err.Error() != "journal is required" {
			t.Fatalf("Append() error = %v, want journal is required", err)
		}
	})

	t.Run("missing campaign id", func(t *testing.T) {
		memory := NewMemory(nil)
		_, err := memory.Append(context.Background(), event.Event{CampaignID: "   "})
		if !errors.Is(err, ErrCampaignIDRequired) {
			t.Fatalf("expected ErrCampaignIDRequired, got %v", err)
		}
	})

	t.Run("context canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		memory := NewMemory(nil)
		_, err := memory.Append(ctx, event.Event{CampaignID: "camp-1"})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	})

	t.Run("registry validation error", func(t *testing.T) {
		registry := event.NewRegistry()
		if err := registry.Register(event.Definition{
			Type:       event.Type("session.started"),
			Owner:      event.OwnerCore,
			Addressing: event.AddressingPolicyEntityTarget,
		}); err != nil {
			t.Fatalf("register event definition: %v", err)
		}

		memory := NewMemory(registry)
		_, err := memory.Append(context.Background(), event.Event{
			CampaignID:  "camp-1",
			Type:        event.Type("session.started"),
			Timestamp:   time.Unix(0, 0).UTC(),
			ActorType:   event.ActorTypeSystem,
			PayloadJSON: []byte(`{}`),
		})
		if err == nil {
			t.Fatal("expected validation error")
		}
		if !errors.Is(err, event.ErrEntityTypeRequired) {
			t.Fatalf("expected ErrEntityTypeRequired, got %v", err)
		}
	})
}

func TestMemoryListEvents_ErrorAndEdgePaths(t *testing.T) {
	t.Run("nil journal", func(t *testing.T) {
		var memory *Memory
		_, err := memory.ListEvents(context.Background(), "camp-1", 0, 10)
		if err == nil || err.Error() != "journal is required" {
			t.Fatalf("ListEvents() error = %v, want journal is required", err)
		}
	})

	t.Run("missing campaign id", func(t *testing.T) {
		memory := NewMemory(nil)
		_, err := memory.ListEvents(context.Background(), " ", 0, 10)
		if !errors.Is(err, ErrCampaignIDRequired) {
			t.Fatalf("expected ErrCampaignIDRequired, got %v", err)
		}
	})

	t.Run("context canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		memory := NewMemory(nil)
		_, err := memory.ListEvents(ctx, "camp-1", 0, 10)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	})

	t.Run("after seq past end", func(t *testing.T) {
		registry := event.NewRegistry()
		if err := registry.Register(event.Definition{Type: event.Type("session.started"), Owner: event.OwnerCore}); err != nil {
			t.Fatalf("register event: %v", err)
		}
		memory := NewMemory(registry)
		stamp := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
		if _, err := memory.Append(context.Background(), event.Event{
			CampaignID:  "camp-1",
			Type:        event.Type("session.started"),
			Timestamp:   stamp,
			ActorType:   event.ActorTypeSystem,
			PayloadJSON: []byte(`{"session_id":"sess-1"}`),
		}); err != nil {
			t.Fatalf("append event: %v", err)
		}

		page, err := memory.ListEvents(context.Background(), "camp-1", 99, 10)
		if err != nil {
			t.Fatalf("ListEvents() unexpected error: %v", err)
		}
		if page != nil {
			t.Fatalf("ListEvents() = %v, want nil", page)
		}
	})
}

func TestMemoryListEvents_ReturnsCopy(t *testing.T) {
	registry := event.NewRegistry()
	if err := registry.Register(event.Definition{Type: event.Type("session.started"), Owner: event.OwnerCore}); err != nil {
		t.Fatalf("register event: %v", err)
	}

	memory := NewMemory(registry)
	stamp := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	if _, err := memory.Append(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("session.started"),
		Timestamp:   stamp,
		ActorType:   event.ActorTypeSystem,
		PayloadJSON: []byte(`{"session_id":"sess-1"}`),
	}); err != nil {
		t.Fatalf("append event: %v", err)
	}

	page, err := memory.ListEvents(context.Background(), "camp-1", 0, 1)
	if err != nil {
		t.Fatalf("ListEvents() unexpected error: %v", err)
	}
	if len(page) != 1 {
		t.Fatalf("ListEvents() len = %d, want 1", len(page))
	}
	page[0].Type = event.Type("corrupted")

	fresh, err := memory.ListEvents(context.Background(), "camp-1", 0, 1)
	if err != nil {
		t.Fatalf("ListEvents() unexpected error: %v", err)
	}
	if len(fresh) != 1 {
		t.Fatalf("fresh ListEvents() len = %d, want 1", len(fresh))
	}
	if fresh[0].Type != event.Type("session.started") {
		t.Fatalf("fresh event type = %s, want session.started", fresh[0].Type)
	}
}
