package game

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/manifest"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestApplyStressVulnerableCondition_AddsCondition(t *testing.T) {
	ctx := context.Background()
	eventStore := newFakeEventStore()
	dhStore := newFakeDaggerheartStore()
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	payload := daggerheart.ConditionChangedPayload{
		CharacterID:      "ch1",
		ConditionsBefore: []string{},
		ConditionsAfter:  []string{"vulnerable"},
		Added:            []string{"vulnerable"},
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode condition payload: %v", err)
	}
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.condition.change"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "c1",
				Type:          event.Type("sys.daggerheart.condition_changed"),
				Timestamp:     now,
				ActorType:     event.ActorTypeGM,
				SessionID:     "s1",
				EntityType:    "character",
				EntityID:      "ch1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}

	err = applyStressVulnerableCondition(
		ctx,
		Stores{Event: eventStore, SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore}, Domain: domain},
		"c1",
		"s1",
		"ch1",
		nil,
		2,
		6,
		6,
		event.ActorTypeGM,
		"gm-1",
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.Type("sys.daggerheart.condition_changed") {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, "sys.daggerheart.condition_changed")
	}
	state, err := dhStore.GetDaggerheartCharacterState(ctx, "c1", "ch1")
	if err != nil {
		t.Fatalf("expected daggerheart state, got %v", err)
	}
	if !containsCondition(state.Conditions, "vulnerable") {
		t.Fatalf("expected vulnerable condition, got %v", state.Conditions)
	}
}

func TestApplyStressVulnerableCondition_RemovesCondition(t *testing.T) {
	ctx := context.Background()
	eventStore := newFakeEventStore()
	dhStore := newFakeDaggerheartStore()
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	payload := daggerheart.ConditionChangedPayload{
		CharacterID:      "ch1",
		ConditionsBefore: []string{"vulnerable"},
		ConditionsAfter:  []string{},
		Removed:          []string{"vulnerable"},
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode condition payload: %v", err)
	}
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.condition.change"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "c1",
				Type:          event.Type("sys.daggerheart.condition_changed"),
				Timestamp:     now,
				ActorType:     event.ActorTypeGM,
				SessionID:     "s1",
				EntityType:    "character",
				EntityID:      "ch1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}

	err = applyStressVulnerableCondition(
		ctx,
		Stores{Event: eventStore, SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore}, Domain: domain},
		"c1",
		"s1",
		"ch1",
		[]string{"vulnerable"},
		6,
		5,
		6,
		event.ActorTypeGM,
		"gm-1",
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	state, err := dhStore.GetDaggerheartCharacterState(ctx, "c1", "ch1")
	if err != nil {
		t.Fatalf("expected daggerheart state, got %v", err)
	}
	if containsCondition(state.Conditions, "vulnerable") {
		t.Fatalf("expected vulnerable condition removed, got %v", state.Conditions)
	}
}

func TestApplyStressVulnerableCondition_NoOpWhenUnchanged(t *testing.T) {
	ctx := context.Background()
	eventStore := newFakeEventStore()
	dhStore := newFakeDaggerheartStore()
	noopDomain := &fakeDomainEngine{}

	err := applyStressVulnerableCondition(
		ctx,
		Stores{Event: eventStore, SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore}, Domain: noopDomain},
		"c1",
		"s1",
		"ch1",
		nil,
		3,
		3,
		6,
		event.ActorTypeGM,
		"gm-1",
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got := len(eventStore.events["c1"]); got != 0 {
		t.Fatalf("expected 0 events, got %d", got)
	}
}

func TestApplyStressVulnerableCondition_NoOpWhenAlreadyVulnerable(t *testing.T) {
	ctx := context.Background()
	eventStore := newFakeEventStore()
	dhStore := newFakeDaggerheartStore()
	noopDomain := &fakeDomainEngine{}

	err := applyStressVulnerableCondition(
		ctx,
		Stores{Event: eventStore, SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore}, Domain: noopDomain},
		"c1",
		"s1",
		"ch1",
		[]string{"vulnerable"},
		5,
		6,
		6,
		event.ActorTypeGM,
		"gm-1",
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got := len(eventStore.events["c1"]); got != 0 {
		t.Fatalf("expected 0 events, got %d", got)
	}
}

func containsCondition(conditions []string, target string) bool {
	for _, condition := range conditions {
		if condition == target {
			return true
		}
	}
	return false
}
