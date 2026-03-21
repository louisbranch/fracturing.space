package eventtransport

import (
	"context"
	"testing"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

type fakeDomainEngine struct {
	store         *gametest.FakeEventStore
	resultsByType map[command.Type]engine.Result
	calls         int
	commands      []command.Command
}

func (f *fakeDomainEngine) Execute(_ context.Context, cmd command.Command) (engine.Result, error) {
	f.calls++
	f.commands = append(f.commands, cmd)
	if result, ok := f.resultsByType[cmd.Type]; ok {
		return result, nil
	}
	return engine.Result{}, nil
}

func appendEventScopeContext(scope string) context.Context {
	return metadata.NewIncomingContext(
		context.Background(),
		metadata.Pairs(appendEventScopeHeader, scope),
	)
}

func TestAppendEvent_UsesDomainEngineForActionEvents(t *testing.T) {
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	ctx := appendEventScopeContext(appendEventScopeMaintenance)

	cases := []struct {
		name        string
		eventType   string
		commandType command.Type
	}{
		{
			name:        "note added",
			eventType:   "story.note_added",
			commandType: command.Type("story.note.add"),
		},
		{
			name:        "roll resolved",
			eventType:   "action.roll_resolved",
			commandType: command.Type("action.roll.resolve"),
		},
		{
			name:        "outcome applied",
			eventType:   "action.outcome_applied",
			commandType: command.Type("action.outcome.apply"),
		},
		{
			name:        "outcome rejected",
			eventType:   "action.outcome_rejected",
			commandType: command.Type("action.outcome.reject"),
		},
	}

	results := make(map[command.Type]engine.Result, len(cases))
	for _, tc := range cases {
		results[tc.commandType] = engine.Result{
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type(tc.eventType),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "campaign",
				EntityID:    "c1",
				PayloadJSON: []byte("{}"),
			}),
		}
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: results}
	svc := NewService(Deps{Event: eventStore, Write: domainwrite.WritePath{Executor: domain, Runtime: testRuntime}})

	for _, tc := range cases {
		_, err := svc.AppendEvent(ctx, &campaignv1.AppendEventRequest{
			CampaignId:  "c1",
			Type:        tc.eventType,
			ActorType:   "system",
			EntityType:  "campaign",
			EntityId:    "c1",
			PayloadJson: []byte("{}"),
		})
		if err != nil {
			t.Fatalf("AppendEvent %s returned error: %v", tc.name, err)
		}
	}

	if domain.calls != len(cases) {
		t.Fatalf("expected domain to be called %d times, got %d", len(cases), domain.calls)
	}
	if len(domain.commands) != len(cases) {
		t.Fatalf("expected %d domain commands, got %d", len(cases), len(domain.commands))
	}
	for i, tc := range cases {
		if domain.commands[i].Type != tc.commandType {
			t.Fatalf("command type = %s, want %s", domain.commands[i].Type, tc.commandType)
		}
	}
}

func TestAppendEvent_RequiresMaintenanceOrAdminScope(t *testing.T) {
	eventStore := gametest.NewFakeEventStore()
	svc := NewService(Deps{Event: eventStore})

	_, err := svc.AppendEvent(context.Background(), &campaignv1.AppendEventRequest{
		CampaignId:  "c1",
		Type:        "story.note_added",
		ActorType:   "system",
		EntityType:  "note",
		EntityId:    "note-1",
		PayloadJson: []byte(`{"content":"note"}`),
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestAppendEvent_RejectsUnmappedTypeWithDomain(t *testing.T) {
	eventStore := gametest.NewFakeEventStore()
	domain := &fakeDomainEngine{store: eventStore}
	svc := NewService(Deps{Event: eventStore, Write: domainwrite.WritePath{Executor: domain, Runtime: testRuntime}})
	ctx := appendEventScopeContext(appendEventScopeMaintenance)

	_, err := svc.AppendEvent(ctx, &campaignv1.AppendEventRequest{
		CampaignId:  "c1",
		Type:        "campaign.created",
		ActorType:   "system",
		EntityType:  "campaign",
		EntityId:    "c1",
		PayloadJson: []byte("{}"),
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
	if domain.calls != 0 {
		t.Fatalf("expected domain not to be called, got %d", domain.calls)
	}
}

func TestAppendEvent_RequiresDomainEngine(t *testing.T) {
	eventStore := gametest.NewFakeEventStore()
	svc := NewService(Deps{Event: eventStore})
	ctx := appendEventScopeContext(appendEventScopeMaintenance)

	_, err := svc.AppendEvent(ctx, &campaignv1.AppendEventRequest{
		CampaignId:  "c1",
		Type:        "story.note_added",
		ActorType:   "system",
		EntityType:  "note",
		EntityId:    "note-1",
		PayloadJson: []byte(`{"content":"note"}`),
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestAppendEvent_ReturnsRequestedMappedEventWhenDomainEmitsMultipleEvents(t *testing.T) {
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	ctx := appendEventScopeContext(appendEventScopeMaintenance)
	domain := &fakeDomainEngine{
		store: eventStore,
		resultsByType: map[command.Type]engine.Result{
			command.Type("action.outcome.apply"): {
				Decision: command.Accept(
					event.Event{
						CampaignID:    "c1",
						Type:          event.Type("sys.daggerheart.gm_fear_changed"),
						Timestamp:     now,
						ActorType:     event.ActorTypeSystem,
						EntityType:    "campaign",
						EntityID:      "c1",
						SystemID:      "daggerheart",
						SystemVersion: "1.0.0",
						PayloadJSON:   []byte(`{"before":0,"after":1}`),
					},
					event.Event{
						CampaignID:  "c1",
						Type:        event.Type("action.outcome_applied"),
						Timestamp:   now,
						ActorType:   event.ActorTypeSystem,
						EntityType:  "outcome",
						EntityID:    "req-1",
						PayloadJSON: []byte(`{"request_id":"req-1","roll_seq":1}`),
					},
				),
			},
		},
	}
	svc := NewService(Deps{Event: eventStore, Write: domainwrite.WritePath{Executor: domain, Runtime: testRuntime}})

	resp, err := svc.AppendEvent(ctx, &campaignv1.AppendEventRequest{
		CampaignId:  "c1",
		Type:        "action.outcome_applied",
		ActorType:   "system",
		EntityType:  "outcome",
		EntityId:    "req-1",
		PayloadJson: []byte(`{"request_id":"req-1","roll_seq":1}`),
	})
	if err != nil {
		t.Fatalf("AppendEvent returned error: %v", err)
	}
	if got, want := resp.GetEvent().GetType(), "action.outcome_applied"; got != want {
		t.Fatalf("event type = %s, want %s", got, want)
	}
}

func TestAppendEvent_TokenHelpersRemainListOwned(t *testing.T) {
	cursor := pagination.NewCursor(
		[]pagination.CursorValue{pagination.UintValue("seq", 10)},
		pagination.DirectionForward,
		false,
		"",
		"seq",
	)
	if _, err := pagination.Encode(cursor); err != nil {
		t.Fatalf("encode cursor: %v", err)
	}
}
