package sessiontransport

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
)

func testDecisionEvent() event.Event {
	return event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("campaign.created"),
		Timestamp:   time.Now().UTC(),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-1",
		PayloadJSON: []byte(`{"name":"C","system":"GAME_SYSTEM_DAGGERHEART","gm_mode":"HUMAN"}`),
	}
}

func TestExecuteSessionGateCommandAndLoad_Success(t *testing.T) {
	// Use a dedicated runtime with inline apply disabled: this test
	// validates the execute-then-load callback flow, not projection apply.
	runtime := gametest.SetupRuntime()
	runtime.SetInlineApplyEnabled(false)

	domain := &fakeDomainEngine{
		resultsByType: map[command.Type]engine.Result{
			command.Type("session_gate.open"): {
				Decision: command.Decision{Events: []event.Event{testDecisionEvent()}},
			},
		},
	}
	executor := newSessionGateCommandExecutor(domainwriteexec.WritePath{Executor: domain, Runtime: runtime}, projection.Applier{})

	loaded := false
	value, err := executeSessionGateCommandAndLoad(
		context.Background(),
		executor,
		command.Type("session_gate.open"),
		"camp-1",
		"session-1",
		"gate-1",
		map[string]string{"state": "open"},
		"open gate",
		func(context.Context) (string, error) {
			loaded = true
			return "loaded", nil
		},
	)
	if err != nil {
		t.Fatalf("execute and load: %v", err)
	}
	if !loaded {
		t.Fatal("expected load callback to run")
	}
	if value != "loaded" {
		t.Fatalf("value = %q, want %q", value, "loaded")
	}
}

func TestExecuteSessionGateCommandAndLoad_PropagatesExecuteError(t *testing.T) {
	executor := newSessionGateCommandExecutor(
		domainwriteexec.WritePath{Executor: &fakeDomainEngine{}, Runtime: testRuntime},
		projection.Applier{},
	)

	loaded := false
	_, err := executeSessionGateCommandAndLoad(
		context.Background(),
		executor,
		command.Type("session_gate.open"),
		"camp-1",
		"session-1",
		"gate-1",
		map[string]string{"state": "open"},
		"open gate",
		func(context.Context) (string, error) {
			loaded = true
			return "loaded", nil
		},
	)
	if err == nil {
		t.Fatal("expected execute error")
	}
	if loaded {
		t.Fatal("load callback should not run when execute fails")
	}
}
