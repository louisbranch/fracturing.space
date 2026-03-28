package workflowwrite

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowruntime"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/test/mock/gamefakes"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeEventApplier struct {
	calls int
	err   error
}

func (f *fakeEventApplier) Apply(context.Context, event.Event) error {
	f.calls++
	return f.err
}

func testSystemEvent() event.Event {
	return event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("sys.daggerheart.gm_fear_changed"),
		Timestamp:   time.Now().UTC(),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-1",
		PayloadJSON: []byte(`{"before":0,"after":1}`),
	}
}

type fakeDomainExecutor struct {
	result engine.Result
	err    error
}

func (f fakeDomainExecutor) Execute(context.Context, command.Command) (engine.Result, error) {
	return f.result, f.err
}

type captureExecutor struct {
	result engine.Result
	err    error
	cmd    command.Command
}

func (c *captureExecutor) Execute(_ context.Context, cmd command.Command) (engine.Result, error) {
	c.cmd = cmd
	return c.result, c.err
}

type nonRetryableTestError struct {
	err error
}

func (e nonRetryableTestError) Error() string      { return e.err.Error() }
func (e nonRetryableTestError) Unwrap() error      { return e.err }
func (e nonRetryableTestError) NonRetryable() bool { return true }

func testWriteRuntime(t *testing.T) *domainwrite.Runtime {
	t.Helper()
	runtime := domainwrite.NewRuntime()
	registry := event.NewRegistry()
	for _, def := range []event.Definition{
		{Type: event.Type("sys.daggerheart.gm_fear_changed"), Owner: event.OwnerSystem, Intent: event.IntentProjectionAndReplay},
		{Type: event.Type("story.note_added"), Owner: event.OwnerCore, Intent: event.IntentAuditOnly},
	} {
		if err := registry.Register(def); err != nil {
			t.Fatalf("register event: %v", err)
		}
	}
	runtime.SetIntentFilter(registry)
	return runtime
}

func TestExecuteAndApplyRequiresEvents(t *testing.T) {
	runtime := testWriteRuntime(t)
	runtime.SetInlineApplyEnabled(true)

	deps := domainwrite.WritePath{
		Executor: fakeDomainExecutor{result: engine.Result{Decision: command.Decision{}}},
		Runtime:  runtime,
	}

	_, err := ExecuteAndApply(
		context.Background(),
		deps,
		&fakeEventApplier{},
		command.Command{CampaignID: "camp-1", Type: command.Type("sys.daggerheart.gm_fear.set")},
		domainwrite.Options{RequireEvents: true, MissingEventMsg: "missing events"},
	)
	if err == nil {
		t.Fatal("expected missing-event error")
	}
	if !strings.Contains(err.Error(), "missing events") {
		t.Fatalf("error = %v, want missing events message", err)
	}
}

func TestExecuteAndApplySkipsApplyWhenInlineDisabled(t *testing.T) {
	runtime := testWriteRuntime(t)
	runtime.SetInlineApplyEnabled(false)

	applier := &fakeEventApplier{err: errors.New("should not apply")}
	deps := domainwrite.WritePath{
		Executor: fakeDomainExecutor{
			result: engine.Result{
				Decision: command.Decision{Events: []event.Event{testSystemEvent()}},
			},
		},
		Runtime: runtime,
	}

	_, err := ExecuteAndApply(
		context.Background(),
		deps,
		applier,
		command.Command{CampaignID: "camp-1", Type: command.Type("sys.daggerheart.gm_fear.set")},
		domainwrite.Options{RequireEvents: true, MissingEventMsg: "missing events"},
	)
	if err != nil {
		t.Fatalf("execute and apply with inline disabled: %v", err)
	}
	if applier.calls != 0 {
		t.Fatalf("apply calls = %d, want 0", applier.calls)
	}
}

func TestExecuteAndApplyAppliesWhenInlineEnabled(t *testing.T) {
	runtime := testWriteRuntime(t)
	runtime.SetInlineApplyEnabled(true)

	applier := &fakeEventApplier{}
	deps := domainwrite.WritePath{
		Executor: fakeDomainExecutor{
			result: engine.Result{
				Decision: command.Decision{Events: []event.Event{testSystemEvent()}},
			},
		},
		Runtime: runtime,
	}

	_, err := ExecuteAndApply(
		context.Background(),
		deps,
		applier,
		command.Command{CampaignID: "camp-1", Type: command.Type("sys.daggerheart.gm_fear.set")},
		domainwrite.Options{RequireEvents: true, MissingEventMsg: "missing events"},
	)
	if err != nil {
		t.Fatalf("execute and apply with inline enabled: %v", err)
	}
	if applier.calls != 1 {
		t.Fatalf("apply calls = %d, want 1", applier.calls)
	}
}

func TestExecuteAndApplyReturnsApplyErrorWhenInlineEnabled(t *testing.T) {
	runtime := testWriteRuntime(t)
	runtime.SetInlineApplyEnabled(true)

	applier := &fakeEventApplier{err: errors.New("boom")}
	deps := domainwrite.WritePath{
		Executor: fakeDomainExecutor{
			result: engine.Result{
				Decision: command.Decision{Events: []event.Event{testSystemEvent()}},
			},
		},
		Runtime: runtime,
	}

	_, err := ExecuteAndApply(
		context.Background(),
		deps,
		applier,
		command.Command{CampaignID: "camp-1", Type: command.Type("sys.daggerheart.gm_fear.set")},
		domainwrite.Options{RequireEvents: true, MissingEventMsg: "missing events"},
	)
	if err == nil {
		t.Fatal("expected apply error")
	}
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %s, want %s", status.Code(err), codes.Internal)
	}
	if !strings.Contains(err.Error(), "apply event") {
		t.Fatalf("error = %v, want apply event prefix", err)
	}
}

func TestExecuteAndApplySkipsJournalOnlyInlineApply(t *testing.T) {
	runtime := testWriteRuntime(t)
	runtime.SetInlineApplyEnabled(true)

	applier := &fakeEventApplier{err: errors.New("should not apply")}
	deps := domainwrite.WritePath{
		Executor: fakeDomainExecutor{
			result: engine.Result{
				Decision: command.Decision{Events: []event.Event{
					{
						CampaignID:  "camp-1",
						Type:        event.Type("story.note_added"),
						Timestamp:   time.Now().UTC(),
						ActorType:   event.ActorTypeSystem,
						EntityType:  "note",
						EntityID:    "note-1",
						PayloadJSON: []byte(`{"content":"note"}`),
					},
				}},
			},
		},
		Runtime: runtime,
	}

	_, err := ExecuteAndApply(
		context.Background(),
		deps,
		applier,
		command.Command{CampaignID: "camp-1", Type: command.Type("story.note.add")},
		domainwrite.Options{RequireEvents: true, MissingEventMsg: "missing events"},
	)
	if err != nil {
		t.Fatalf("execute and apply with journal-only event: %v", err)
	}
	if applier.calls != 0 {
		t.Fatalf("apply calls = %d, want 0", applier.calls)
	}
}

func TestExecuteAndApplyMapsNonRetryableExecutionError(t *testing.T) {
	runtime := testWriteRuntime(t)
	runtime.SetInlineApplyEnabled(true)

	deps := domainwrite.WritePath{
		Executor: fakeDomainExecutor{err: nonRetryableTestError{err: errors.New("checkpoint save failed")}},
		Runtime:  runtime,
	}

	_, err := ExecuteAndApply(
		context.Background(),
		deps,
		&fakeEventApplier{},
		command.Command{CampaignID: "camp-1", Type: command.Type("sys.daggerheart.gm_fear.set")},
		domainwrite.Options{},
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %s, want %s", status.Code(err), codes.FailedPrecondition)
	}
}

func TestExecuteDomainCommandBuildsSystemCommand(t *testing.T) {
	runtime := testWriteRuntime(t)
	runtime.SetInlineApplyEnabled(false)

	executor := &captureExecutor{
		result: engine.Result{
			Decision: command.Decision{Events: []event.Event{testSystemEvent()}},
		},
	}
	err := ExecuteDomainCommand(
		context.Background(),
		domainwrite.WritePath{Executor: executor, Runtime: runtime},
		gamefakes.NewDaggerheartStore(),
		DomainCommandInput{
			CampaignID:      "camp-1",
			CommandType:     command.Type("sys.daggerheart.gm_fear.set"),
			SessionID:       "sess-1",
			SceneID:         "scene-1",
			RequestID:       "req-1",
			InvocationID:    "inv-1",
			EntityType:      "campaign",
			EntityID:        "camp-1",
			PayloadJSON:     []byte(`{"before":0,"after":1}`),
			MissingEventMsg: "missing events",
			ApplyErrMessage: "apply event",
		},
	)
	if err != nil {
		t.Fatalf("execute domain command: %v", err)
	}
	if executor.cmd.ActorType != command.ActorTypeSystem {
		t.Fatalf("actor type = %q, want %q", executor.cmd.ActorType, command.ActorTypeSystem)
	}
	if executor.cmd.SystemID != daggerheart.SystemID {
		t.Fatalf("system id = %q, want %q", executor.cmd.SystemID, daggerheart.SystemID)
	}
	if executor.cmd.SystemVersion != daggerheart.SystemVersion {
		t.Fatalf("system version = %q, want %q", executor.cmd.SystemVersion, daggerheart.SystemVersion)
	}
	if executor.cmd.Type != command.Type("sys.daggerheart.gm_fear.set") {
		t.Fatalf("command type = %q, want sys.daggerheart.gm_fear.set", executor.cmd.Type)
	}
	if executor.cmd.RequestID != "req-1" || executor.cmd.InvocationID != "inv-1" {
		t.Fatalf("request metadata = (%q,%q), want (req-1, inv-1)", executor.cmd.RequestID, executor.cmd.InvocationID)
	}
}

func TestExecuteCoreCommandBuildsCoreSystemCommand(t *testing.T) {
	runtime := testWriteRuntime(t)
	runtime.SetInlineApplyEnabled(false)

	executor := &captureExecutor{
		result: engine.Result{
			Decision: command.Decision{Events: []event.Event{testSystemEvent()}},
		},
	}
	_, err := ExecuteCoreCommand(
		context.Background(),
		domainwrite.WritePath{Executor: executor, Runtime: runtime},
		&fakeEventApplier{},
		CoreCommandInput{
			CampaignID:      "camp-1",
			CommandType:     command.Type("story.note.add"),
			SessionID:       "sess-1",
			SceneID:         "scene-1",
			RequestID:       "req-1",
			InvocationID:    "inv-1",
			CorrelationID:   "corr-1",
			EntityType:      "note",
			EntityID:        "note-1",
			PayloadJSON:     []byte(`{"content":"note"}`),
			MissingEventMsg: "missing events",
			ApplyErrMessage: "apply event",
		},
	)
	if err != nil {
		t.Fatalf("execute core command: %v", err)
	}
	if executor.cmd.ActorType != command.ActorTypeSystem {
		t.Fatalf("actor type = %q, want %q", executor.cmd.ActorType, command.ActorTypeSystem)
	}
	if executor.cmd.SystemID != "" || executor.cmd.SystemVersion != "" {
		t.Fatalf("system metadata = (%q,%q), want empty", executor.cmd.SystemID, executor.cmd.SystemVersion)
	}
	if executor.cmd.CorrelationID != "corr-1" {
		t.Fatalf("correlation id = %q, want corr-1", executor.cmd.CorrelationID)
	}
	if executor.cmd.EntityType != "note" || executor.cmd.EntityID != "note-1" {
		t.Fatalf("entity = (%q,%q), want (note,note-1)", executor.cmd.EntityType, executor.cmd.EntityID)
	}
}

func TestNewRuntimeUsesExecuteAndApplyPolicy(t *testing.T) {
	runtime := testWriteRuntime(t)
	eventStore := gamefakes.NewEventStore()
	daggerheartStore := gamefakes.NewDaggerheartStore()
	deps := domainwrite.WritePath{
		Executor: fakeDomainExecutor{
			result: engine.Result{
				Decision: command.Decision{Events: []event.Event{testSystemEvent()}},
			},
		},
		Runtime: runtime,
	}

	sharedRuntime := NewRuntime(deps, eventStore, daggerheartStore)
	err := sharedRuntime.ExecuteSystemCommand(context.Background(), workflowruntime.SystemCommandInput{
		CampaignID:      "camp-1",
		CommandType:     command.Type("sys.daggerheart.gm_fear.set"),
		EntityType:      "campaign",
		EntityID:        "camp-1",
		PayloadJSON:     []byte(`{"before":0,"after":1}`),
		MissingEventMsg: "missing events",
		ApplyErrMessage: "apply event",
	})
	if err != nil {
		t.Fatalf("execute system command through runtime: %v", err)
	}
}
