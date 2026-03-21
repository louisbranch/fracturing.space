package journalimport

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	domainwrite "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
)

func TestImport_AppendsValidatedEventsInOrder(t *testing.T) {
	store := gametest.NewFakeBatchEventStore()
	runtime := domainwrite.NewRuntime()
	runtime.SetInlineApplyEnabled(false)
	registry := event.NewRegistry()
	if err := registry.Register(event.Definition{
		Type:            event.Type("campaign.created"),
		Owner:           event.OwnerCore,
		Addressing:      event.AddressingPolicyEntityTarget,
		ValidatePayload: func(json.RawMessage) error { return nil },
	}); err != nil {
		t.Fatalf("register event: %v", err)
	}

	importer := NewService(store, projection.Applier{}, runtime, registry)
	err := importer.Import(context.Background(), []event.Event{
		{
			CampaignID:  "campaign-1",
			Type:        event.Type("campaign.created"),
			Timestamp:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			ActorType:   event.ActorTypeSystem,
			EntityType:  "campaign",
			EntityID:    "campaign-1",
			PayloadJSON: []byte(`{}`),
		},
		{
			CampaignID:  "campaign-1",
			Type:        event.Type("campaign.created"),
			Timestamp:   time.Date(2025, 1, 1, 0, 0, 1, 0, time.UTC),
			ActorType:   event.ActorTypeSystem,
			EntityType:  "campaign",
			EntityID:    "campaign-1",
			PayloadJSON: []byte(`{}`),
		},
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}

	events := store.Events["campaign-1"]
	if len(events) != 2 {
		t.Fatalf("stored events = %d, want 2", len(events))
	}
	if events[0].Seq != 1 || events[1].Seq != 2 {
		t.Fatalf("stored seqs = %d/%d, want 1/2", events[0].Seq, events[1].Seq)
	}
}
