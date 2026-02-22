package game

import (
	"context"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestEventStoreAdapterListEvents_NilStoreReturnsError(t *testing.T) {
	adapter := EventStoreAdapter{}

	_, err := adapter.ListEvents(context.Background(), "camp-1", 0, 10)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestJournalAdapterAppend_NilStoreReturnsError(t *testing.T) {
	adapter := JournalAdapter{}

	_, err := adapter.Append(context.Background(), event.Event{
		CampaignID: "camp-1",
		Type:       event.Type("campaign.created"),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestJournalAdapterBatchAppend_RejectsNonBatchStore(t *testing.T) {
	adapter := JournalAdapter{store: &nonBatchEventStore{}}

	events := []event.Event{
		{CampaignID: "camp-1", Type: event.Type("a")},
		{CampaignID: "camp-1", Type: event.Type("b")},
		{CampaignID: "camp-1", Type: event.Type("c")},
	}
	_, err := adapter.BatchAppend(context.Background(), events)
	if err == nil {
		t.Fatal("expected error for store without batch support")
	}
	if !strings.Contains(err.Error(), "batch append not supported") {
		t.Fatalf("expected 'batch append not supported' error, got: %v", err)
	}
}

// nonBatchEventStore implements storage.EventStore but not batchAppender.
type nonBatchEventStore struct {
	fakeEventStore
}
