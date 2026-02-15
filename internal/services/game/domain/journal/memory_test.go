package journal

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestMemoryAppend_AssignsSeqAndHashes(t *testing.T) {
	registry := event.NewRegistry()
	if err := registry.Register(event.Definition{
		Type:  event.Type("session.started"),
		Owner: event.OwnerCore,
	}); err != nil {
		t.Fatalf("register event: %v", err)
	}
	store := NewMemory(registry)
	stamp := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	first, err := store.Append(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("session.started"),
		Timestamp:   stamp,
		ActorType:   event.ActorTypeSystem,
		PayloadJSON: []byte(`{"session_id":"sess-1"}`),
	})
	if err != nil {
		t.Fatalf("append first: %v", err)
	}
	if first.Seq != 1 {
		t.Fatalf("first seq = %d, want %d", first.Seq, 1)
	}
	if first.Hash == "" {
		t.Fatal("expected first hash")
	}
	if first.PrevHash != "" {
		t.Fatalf("first prev hash = %q, want empty", first.PrevHash)
	}
	if first.ChainHash == "" {
		t.Fatal("expected first chain hash")
	}

	second, err := store.Append(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("session.started"),
		Timestamp:   stamp.Add(time.Minute),
		ActorType:   event.ActorTypeSystem,
		PayloadJSON: []byte(`{"session_id":"sess-2"}`),
	})
	if err != nil {
		t.Fatalf("append second: %v", err)
	}
	if second.Seq != 2 {
		t.Fatalf("second seq = %d, want %d", second.Seq, 2)
	}
	if second.PrevHash != first.ChainHash {
		t.Fatalf("second prev hash = %q, want %q", second.PrevHash, first.ChainHash)
	}
}

func TestMemoryListEvents_RespectsAfterSeqAndLimit(t *testing.T) {
	registry := event.NewRegistry()
	if err := registry.Register(event.Definition{
		Type:  event.Type("session.started"),
		Owner: event.OwnerCore,
	}); err != nil {
		t.Fatalf("register event: %v", err)
	}
	store := NewMemory(registry)
	stamp := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	for idx := 0; idx < 3; idx++ {
		_, err := store.Append(context.Background(), event.Event{
			CampaignID:  "camp-1",
			Type:        event.Type("session.started"),
			Timestamp:   stamp.Add(time.Duration(idx) * time.Minute),
			ActorType:   event.ActorTypeSystem,
			PayloadJSON: []byte(`{"session_id":"sess-1"}`),
		})
		if err != nil {
			t.Fatalf("append %d: %v", idx, err)
		}
	}

	page, err := store.ListEvents(context.Background(), "camp-1", 1, 2)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(page) != 2 {
		t.Fatalf("page length = %d, want %d", len(page), 2)
	}
	if page[0].Seq != 2 || page[1].Seq != 3 {
		t.Fatalf("page seqs = %d,%d, want 2,3", page[0].Seq, page[1].Seq)
	}
}
