package snapshottransport

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

func TestApplyStressVulnerableCondition_AddsCondition(t *testing.T) {
	ctx := context.Background()
	eventStore := gametest.NewFakeEventStore()
	dhStore := gametest.NewFakeDaggerheartStore()
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	payload := daggerheartpayload.ConditionChangedPayload{
		CharacterID: "ch1",
		Conditions:  []rules.ConditionState{mustStandardSnapshotCondition(t, "vulnerable")},
		Added:       []rules.ConditionState{mustStandardSnapshotCondition(t, "vulnerable")},
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
	write := domainwrite.WritePath{Executor: domain, Runtime: testRuntime}

	err = applyStressVulnerableCondition(
		ctx,
		dhStore,
		write,
		testApplier(dhStore),
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
	if got := len(eventStore.Events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.Events["c1"][0].Type != event.Type("sys.daggerheart.condition_changed") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["c1"][0].Type, "sys.daggerheart.condition_changed")
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
	eventStore := gametest.NewFakeEventStore()
	dhStore := gametest.NewFakeDaggerheartStore()
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	payload := daggerheartpayload.ConditionChangedPayload{
		CharacterID: "ch1",
		Conditions:  nil,
		Removed:     []rules.ConditionState{mustStandardSnapshotCondition(t, "vulnerable")},
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
	write := domainwrite.WritePath{Executor: domain, Runtime: testRuntime}

	err = applyStressVulnerableCondition(
		ctx,
		dhStore,
		write,
		testApplier(dhStore),
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
	if got := len(eventStore.Events["c1"]); got != 1 {
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
	eventStore := gametest.NewFakeEventStore()
	dhStore := gametest.NewFakeDaggerheartStore()
	noopDomain := &fakeDomainEngine{}
	write := domainwrite.WritePath{Executor: noopDomain}

	err := applyStressVulnerableCondition(
		ctx,
		dhStore,
		write,
		testApplier(dhStore),
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
	if got := len(eventStore.Events["c1"]); got != 0 {
		t.Fatalf("expected 0 events, got %d", got)
	}
}

func TestApplyStressVulnerableCondition_NoOpWhenAlreadyVulnerable(t *testing.T) {
	ctx := context.Background()
	eventStore := gametest.NewFakeEventStore()
	dhStore := gametest.NewFakeDaggerheartStore()
	noopDomain := &fakeDomainEngine{}
	write := domainwrite.WritePath{Executor: noopDomain}

	err := applyStressVulnerableCondition(
		ctx,
		dhStore,
		write,
		testApplier(dhStore),
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
	if got := len(eventStore.Events["c1"]); got != 0 {
		t.Fatalf("expected 0 events, got %d", got)
	}
}

func containsCondition(conditions []projectionstore.DaggerheartConditionState, target string) bool {
	for _, condition := range conditions {
		if condition.Code == target || condition.Standard == target || condition.ID == target {
			return true
		}
	}
	return false
}

func mustStandardSnapshotCondition(t *testing.T, code string) rules.ConditionState {
	t.Helper()
	state, err := rules.StandardConditionState(code)
	if err != nil {
		t.Fatalf("standard condition state %q: %v", code, err)
	}
	return state
}
