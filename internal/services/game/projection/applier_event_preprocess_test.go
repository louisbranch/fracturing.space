package projection

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestPrepareEventForProjection_NoRegistry(t *testing.T) {
	applier := Applier{}
	evt := event.Event{Type: event.Type("campaign.created")}

	got, shouldProject := applier.prepareEventForProjection(evt)
	if !shouldProject {
		t.Fatal("expected projection to continue when no registry is configured")
	}
	if got.Type != evt.Type {
		t.Fatalf("type = %q, want %q", got.Type, evt.Type)
	}
}

func TestPrepareEventForProjection_AliasResolves(t *testing.T) {
	registry := event.NewRegistry()
	if err := registry.Register(event.Definition{
		Type:   event.Type("campaign.created"),
		Owner:  event.OwnerCore,
		Intent: event.IntentProjectionAndReplay,
	}); err != nil {
		t.Fatalf("register canonical type: %v", err)
	}
	if err := registry.RegisterAlias(event.Type("campaign.created.v1"), event.Type("campaign.created")); err != nil {
		t.Fatalf("register alias: %v", err)
	}

	applier := Applier{Events: registry}
	got, shouldProject := applier.prepareEventForProjection(event.Event{Type: event.Type("campaign.created.v1")})
	if !shouldProject {
		t.Fatal("expected alias event to remain projectable")
	}
	if got.Type != event.Type("campaign.created") {
		t.Fatalf("type = %q, want %q", got.Type, event.Type("campaign.created"))
	}
}

func TestPrepareEventForProjection_SkipsAuditOnly(t *testing.T) {
	registry := event.NewRegistry()
	if err := registry.Register(event.Definition{
		Type:   event.Type("test.audit"),
		Owner:  event.OwnerCore,
		Intent: event.IntentAuditOnly,
	}); err != nil {
		t.Fatalf("register audit type: %v", err)
	}

	applier := Applier{Events: registry}
	_, shouldProject := applier.prepareEventForProjection(event.Event{Type: event.Type("test.audit")})
	if shouldProject {
		t.Fatal("expected audit-only event to be skipped")
	}
}
