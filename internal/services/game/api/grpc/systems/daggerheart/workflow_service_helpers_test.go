package daggerheart

import (
	"context"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowruntime"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	dhbridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
)

func TestExecuteWorkflowSystemCommandUsesServiceStores(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.gm_fear.set"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.gm_fear_changed"),
				Timestamp:     testTimestamp,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-system-1",
				EntityType:    "campaign",
				EntityID:      "camp-1",
				SystemID:      dhbridge.SystemID,
				SystemVersion: dhbridge.SystemVersion,
				PayloadJSON:   []byte(`{"after":1}`),
			}),
		},
	}}
	svc.stores.Write.Executor = domain

	err := svc.executeWorkflowSystemCommand(context.Background(), workflowruntime.SystemCommandInput{
		CampaignID:      "camp-1",
		CommandType:     command.Type("sys.daggerheart.gm_fear.set"),
		SessionID:       "sess-1",
		RequestID:       "req-system-1",
		InvocationID:    "inv-system-1",
		EntityType:      "campaign",
		EntityID:        "camp-1",
		PayloadJSON:     []byte(`{"after":1}`),
		MissingEventMsg: "missing events",
		ApplyErrMessage: "apply event",
	})
	if err != nil {
		t.Fatalf("execute workflow system command: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("domain calls = %d, want 1", domain.calls)
	}
	if domain.lastCommand.SystemID != dhbridge.SystemID {
		t.Fatalf("system id = %q, want %q", domain.lastCommand.SystemID, dhbridge.SystemID)
	}
	if domain.lastCommand.SystemVersion != dhbridge.SystemVersion {
		t.Fatalf("system version = %q, want %q", domain.lastCommand.SystemVersion, dhbridge.SystemVersion)
	}
}

func TestExecuteWorkflowCoreCommandUsesServiceApplier(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("action.roll.resolve"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.roll_resolved"),
				Timestamp:   testTimestamp,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-core-1",
				EntityType:  "roll",
				EntityID:    "req-core-1",
				PayloadJSON: []byte(`{"request_id":"req-core-1"}`),
			}),
		},
	}}
	svc.stores.Write.Executor = domain

	result, err := svc.executeWorkflowCoreCommand(context.Background(), workflowwrite.CoreCommandInput{
		CampaignID:      "camp-1",
		CommandType:     command.Type("action.roll.resolve"),
		SessionID:       "sess-1",
		RequestID:       "req-core-1",
		InvocationID:    "inv-core-1",
		CorrelationID:   "corr-core-1",
		EntityType:      "roll",
		EntityID:        "req-core-1",
		PayloadJSON:     []byte(`{"request_id":"req-core-1"}`),
		MissingEventMsg: "missing events",
		ApplyErrMessage: "apply event",
	})
	if err != nil {
		t.Fatalf("execute workflow core command: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("domain calls = %d, want 1", domain.calls)
	}
	if domain.lastCommand.SystemID != "" || domain.lastCommand.SystemVersion != "" {
		t.Fatalf("system metadata = (%q,%q), want empty", domain.lastCommand.SystemID, domain.lastCommand.SystemVersion)
	}
	if domain.lastCommand.CorrelationID != "corr-core-1" {
		t.Fatalf("correlation id = %q, want corr-core-1", domain.lastCommand.CorrelationID)
	}
	if len(result.Decision.Events) != 1 {
		t.Fatalf("result events = %d, want 1", len(result.Decision.Events))
	}
}
