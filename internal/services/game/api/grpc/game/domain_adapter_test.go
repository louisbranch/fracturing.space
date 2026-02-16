package game

import (
	"context"
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
