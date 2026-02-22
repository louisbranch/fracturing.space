package projection

import (
	"context"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestApplier_SkipsAuditOnlyEvents(t *testing.T) {
	registry := event.NewRegistry()
	if err := registry.Register(event.Definition{
		Type:   event.Type("test.audit_event"),
		Owner:  event.OwnerCore,
		Intent: event.IntentAuditOnly,
	}); err != nil {
		t.Fatalf("register event: %v", err)
	}

	applier := Applier{Events: registry}

	err := applier.Apply(context.Background(), event.Event{
		Type:       event.Type("test.audit_event"),
		CampaignID: "camp-1",
	})
	if err != nil {
		t.Fatalf("expected nil error for audit-only event, got: %v", err)
	}
}

func TestApplier_UnknownCoreEventWithoutSystemIDReturnsError(t *testing.T) {
	registry := event.NewRegistry()
	if err := registry.Register(event.Definition{
		Type:   event.Type("test.projection_event"),
		Owner:  event.OwnerCore,
		Intent: event.IntentProjectionAndReplay,
	}); err != nil {
		t.Fatalf("register event: %v", err)
	}

	applier := Applier{Events: registry}

	err := applier.Apply(context.Background(), event.Event{
		Type:       event.Type("test.projection_event"),
		CampaignID: "camp-1",
	})
	if err == nil {
		t.Fatal("expected error for unhandled projection event type")
	}
}
