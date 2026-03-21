package workflowruntime

import (
	"context"
	"errors"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type eventStoreStub struct {
	result storage.ListEventsPageResult
	err    error
	req    storage.ListEventsPageRequest
	calls  int
}

func (s *eventStoreStub) ListEventsPage(_ context.Context, req storage.ListEventsPageRequest) (storage.ListEventsPageResult, error) {
	s.calls++
	s.req = req
	return s.result, s.err
}

type executeCapture struct {
	cmd     command.Command
	applier domainwrite.EventApplier
	options domainwrite.Options
	err     error
	calls   int
}

func (c *executeCapture) call(_ context.Context, cmd command.Command, applier domainwrite.EventApplier, options domainwrite.Options) error {
	c.calls++
	c.cmd = cmd
	c.applier = applier
	c.options = options
	return c.err
}

func TestSessionRequestEventExistsSkipsEmptyReplayInputs(t *testing.T) {
	store := &eventStoreStub{}
	runtime := New(Dependencies{Event: store})

	exists, err := runtime.SessionRequestEventExists(context.Background(), ReplayCheckInput{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
	})
	if err != nil {
		t.Fatalf("SessionRequestEventExists returned error: %v", err)
	}
	if exists {
		t.Fatal("expected false for empty replay input")
	}
	if store.calls != 0 {
		t.Fatalf("ListEventsPage calls = %d, want 0", store.calls)
	}
}

func TestSessionRequestEventExistsQueriesEventStore(t *testing.T) {
	store := &eventStoreStub{
		result: storage.ListEventsPageResult{
			Events: []event.Event{{Type: event.Type("sys.daggerheart.condition_changed")}},
		},
	}
	runtime := New(Dependencies{Event: store})

	exists, err := runtime.SessionRequestEventExists(context.Background(), ReplayCheckInput{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
		RollSeq:    7,
		RequestID:  "req-1",
		EventType:  event.Type("sys.daggerheart.condition_changed"),
		EntityID:   "char-1",
	})
	if err != nil {
		t.Fatalf("SessionRequestEventExists returned error: %v", err)
	}
	if !exists {
		t.Fatal("expected event replay check to find matching event")
	}
	if store.req.AfterSeq != 6 {
		t.Fatalf("AfterSeq = %d, want 6", store.req.AfterSeq)
	}
	if store.req.Filter.EntityID != "char-1" {
		t.Fatalf("entity filter = %q, want char-1", store.req.Filter.EntityID)
	}
}

func TestSessionRequestEventExistsRequiresEventStore(t *testing.T) {
	_, err := New(Dependencies{}).SessionRequestEventExists(context.Background(), ReplayCheckInput{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
		RollSeq:    1,
		RequestID:  "req-1",
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %s, want %s", status.Code(err), codes.Internal)
	}
}

func TestExecuteSystemCommandBuildsDaggerheartSystemCommand(t *testing.T) {
	capture := &executeCapture{}
	runtime := New(Dependencies{
		Daggerheart:          projectionStoreStub{},
		ExecuteDomainCommand: capture.call,
	})

	err := runtime.ExecuteSystemCommand(context.Background(), SystemCommandInput{
		CampaignID:      "camp-1",
		CommandType:     command.Type("sys.daggerheart.hope_spend"),
		SessionID:       "sess-1",
		SceneID:         "scene-1",
		RequestID:       "req-1",
		InvocationID:    "inv-1",
		CorrelationID:   "corr-1",
		EntityType:      "character",
		EntityID:        "char-1",
		PayloadJSON:     []byte(`{"after":1}`),
		MissingEventMsg: "missing events",
		ApplyErrMessage: "apply event",
	})
	if err != nil {
		t.Fatalf("ExecuteSystemCommand returned error: %v", err)
	}
	if capture.calls != 1 {
		t.Fatalf("execute calls = %d, want 1", capture.calls)
	}
	if capture.cmd.SystemID != "daggerheart" {
		t.Fatalf("system id = %q, want daggerheart", capture.cmd.SystemID)
	}
	if capture.cmd.CorrelationID != "corr-1" {
		t.Fatalf("correlation id = %q, want corr-1", capture.cmd.CorrelationID)
	}
	if capture.applier == nil {
		t.Fatal("expected a system adapter applier")
	}
	if !capture.options.RequireEvents || capture.options.MissingEventMsg != "missing events" {
		t.Fatal("expected require-events options to propagate")
	}
}

func TestExecuteSystemCommandPropagatesExecutorError(t *testing.T) {
	runtime := New(Dependencies{
		Daggerheart: projectionStoreStub{},
		ExecuteDomainCommand: func(context.Context, command.Command, domainwrite.EventApplier, domainwrite.Options) error {
			return errors.New("boom")
		},
	})
	err := runtime.ExecuteSystemCommand(context.Background(), SystemCommandInput{CampaignID: "camp-1"})
	if err == nil || err.Error() != "boom" {
		t.Fatalf("error = %v, want boom", err)
	}
}

func TestExecuteSystemCommandRequiresDependencies(t *testing.T) {
	err := New(Dependencies{}).ExecuteSystemCommand(context.Background(), SystemCommandInput{CampaignID: "camp-1"})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %s, want %s", status.Code(err), codes.Internal)
	}
}

type projectionStoreStub struct {
	projectionstore.Store
}
