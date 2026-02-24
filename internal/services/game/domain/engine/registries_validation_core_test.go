package engine

import (
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestValidateAliasFoldCoverage_PassesWhenAliasTargetHandled(t *testing.T) {
	registry := event.NewRegistry()
	if err := registry.Register(event.Definition{
		Type:   "campaign.created",
		Owner:  event.OwnerCore,
		Intent: event.IntentProjectionAndReplay,
	}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := registry.RegisterAlias("old.campaign.created", "campaign.created"); err != nil {
		t.Fatalf("register alias: %v", err)
	}

	// campaign.created is a core fold type, so the alias should be covered.
	if err := ValidateAliasFoldCoverage(registry); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAliasFoldCoverage_FailsWhenAliasTargetNotHandled(t *testing.T) {
	registry := event.NewRegistry()
	// Register a type that is NOT in CoreDomains fold handlers.
	if err := registry.Register(event.Definition{
		Type:   "exotic.never_folded",
		Owner:  event.OwnerCore,
		Intent: event.IntentProjectionAndReplay,
	}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := registry.RegisterAlias("old.exotic.type", "exotic.never_folded"); err != nil {
		t.Fatalf("register alias: %v", err)
	}

	err := ValidateAliasFoldCoverage(registry)
	if err == nil {
		t.Fatal("expected error when alias target has no fold handler")
	}
	if !strings.Contains(err.Error(), "exotic.never_folded") {
		t.Fatalf("expected error to mention unhandled type, got: %v", err)
	}
}
