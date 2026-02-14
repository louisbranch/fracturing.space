package sqlite

import (
	"context"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestEventsNilStoreErrors(t *testing.T) {
	ctx := context.Background()
	var s *Store

	if _, err := s.AppendEvent(ctx, event.Event{}); err == nil {
		t.Fatal("expected error from nil store AppendEvent")
	}
	if _, err := s.GetEventByHash(ctx, "hash"); err == nil {
		t.Fatal("expected error from nil store GetEventByHash")
	}
	if _, err := s.GetEventBySeq(ctx, "c", 1); err == nil {
		t.Fatal("expected error from nil store GetEventBySeq")
	}
	if _, err := s.ListEvents(ctx, "c", 0, 10); err == nil {
		t.Fatal("expected error from nil store ListEvents")
	}
	if _, err := s.ListEventsBySession(ctx, "c", "s", 0, 10); err == nil {
		t.Fatal("expected error from nil store ListEventsBySession")
	}
	if _, err := s.GetLatestEventSeq(ctx, "c"); err == nil {
		t.Fatal("expected error from nil store GetLatestEventSeq")
	}
	if _, err := s.ListEventsPage(ctx, storage.ListEventsPageRequest{CampaignID: "c", PageSize: 10}); err == nil {
		t.Fatal("expected error from nil store ListEventsPage")
	}
	if err := s.VerifyEventIntegrity(ctx); err == nil {
		t.Fatal("expected error from nil store VerifyEventIntegrity")
	}
}

func TestEventsCancelledContextErrors(t *testing.T) {
	store := openTestEventsStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := store.AppendEvent(ctx, event.Event{}); err == nil {
		t.Fatal("expected context error from AppendEvent")
	}
	if _, err := store.GetEventByHash(ctx, "hash"); err == nil {
		t.Fatal("expected context error from GetEventByHash")
	}
	if _, err := store.GetEventBySeq(ctx, "c", 1); err == nil {
		t.Fatal("expected context error from GetEventBySeq")
	}
	if _, err := store.ListEvents(ctx, "c", 0, 10); err == nil {
		t.Fatal("expected context error from ListEvents")
	}
	if _, err := store.ListEventsBySession(ctx, "c", "s", 0, 10); err == nil {
		t.Fatal("expected context error from ListEventsBySession")
	}
	if _, err := store.GetLatestEventSeq(ctx, "c"); err == nil {
		t.Fatal("expected context error from GetLatestEventSeq")
	}
	if _, err := store.ListEventsPage(ctx, storage.ListEventsPageRequest{CampaignID: "c", PageSize: 10}); err == nil {
		t.Fatal("expected context error from ListEventsPage")
	}
	if err := store.VerifyEventIntegrity(ctx); err == nil {
		t.Fatal("expected context error from VerifyEventIntegrity")
	}
}

func TestEventsValidationGuards(t *testing.T) {
	store := openTestEventsStore(t)
	ctx := context.Background()

	// GetEventByHash: empty hash
	if _, err := store.GetEventByHash(ctx, ""); err == nil {
		t.Fatal("expected error for empty hash")
	}
	if _, err := store.GetEventByHash(ctx, "  "); err == nil {
		t.Fatal("expected error for whitespace hash")
	}

	// GetEventBySeq: empty campaign ID
	if _, err := store.GetEventBySeq(ctx, "", 1); err == nil {
		t.Fatal("expected error for empty campaign ID in GetEventBySeq")
	}

	// ListEvents: empty campaign ID
	if _, err := store.ListEvents(ctx, "", 0, 10); err == nil {
		t.Fatal("expected error for empty campaign ID in ListEvents")
	}
	// ListEvents: zero limit
	if _, err := store.ListEvents(ctx, "c", 0, 0); err == nil {
		t.Fatal("expected error for zero limit in ListEvents")
	}

	// ListEventsBySession: empty campaign ID
	if _, err := store.ListEventsBySession(ctx, "", "s", 0, 10); err == nil {
		t.Fatal("expected error for empty campaign ID in ListEventsBySession")
	}
	// ListEventsBySession: empty session ID
	if _, err := store.ListEventsBySession(ctx, "c", "", 0, 10); err == nil {
		t.Fatal("expected error for empty session ID in ListEventsBySession")
	}
	// ListEventsBySession: zero limit
	if _, err := store.ListEventsBySession(ctx, "c", "s", 0, 0); err == nil {
		t.Fatal("expected error for zero limit in ListEventsBySession")
	}

	// GetLatestEventSeq: empty campaign ID
	if _, err := store.GetLatestEventSeq(ctx, ""); err == nil {
		t.Fatal("expected error for empty campaign ID in GetLatestEventSeq")
	}

	// ListEventsPage: empty campaign ID
	if _, err := store.ListEventsPage(ctx, storage.ListEventsPageRequest{PageSize: 10}); err == nil {
		t.Fatal("expected error for empty campaign ID in ListEventsPage")
	}
}
