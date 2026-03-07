package daggerheart

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
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

func TestExecuteAndApplyDomainCommandRequiresEvents(t *testing.T) {
	runtime := testWriteRuntime(t)
	runtime.SetInlineApplyEnabled(true)

	svc := &DaggerheartService{
		stores: Stores{
			Domain:       fakeDomainExecutor{result: engine.Result{Decision: command.Decision{}}},
			WriteRuntime: runtime,
		},
	}

	_, err := svc.executeAndApplyDomainCommand(
		context.Background(),
		command.Command{CampaignID: "camp-1", Type: command.Type("sys.daggerheart.gm_fear.set")},
		&fakeEventApplier{},
		domainwrite.Options{RequireEvents: true, MissingEventMsg: "missing events"},
	)
	if err == nil {
		t.Fatal("expected missing-event error")
	}
	if !strings.Contains(err.Error(), "missing events") {
		t.Fatalf("error = %v, want missing events message", err)
	}
}

func TestExecuteAndApplyDomainCommandSkipsApplyWhenInlineDisabled(t *testing.T) {
	runtime := testWriteRuntime(t)
	runtime.SetInlineApplyEnabled(false)

	applier := &fakeEventApplier{err: errors.New("should not apply")}
	svc := &DaggerheartService{
		stores: Stores{
			Domain: fakeDomainExecutor{
				result: engine.Result{
					Decision: command.Decision{Events: []event.Event{testSystemEvent()}},
				},
			},
			WriteRuntime: runtime,
		},
	}

	_, err := svc.executeAndApplyDomainCommand(
		context.Background(),
		command.Command{CampaignID: "camp-1", Type: command.Type("sys.daggerheart.gm_fear.set")},
		applier,
		domainwrite.Options{RequireEvents: true, MissingEventMsg: "missing events"},
	)
	if err != nil {
		t.Fatalf("execute and apply with inline disabled: %v", err)
	}
	if applier.calls != 0 {
		t.Fatalf("apply calls = %d, want 0", applier.calls)
	}
}

func TestExecuteAndApplyDomainCommandAppliesWhenInlineEnabled(t *testing.T) {
	runtime := testWriteRuntime(t)
	runtime.SetInlineApplyEnabled(true)

	applier := &fakeEventApplier{}
	svc := &DaggerheartService{
		stores: Stores{
			Domain: fakeDomainExecutor{
				result: engine.Result{
					Decision: command.Decision{Events: []event.Event{testSystemEvent()}},
				},
			},
			WriteRuntime: runtime,
		},
	}

	_, err := svc.executeAndApplyDomainCommand(
		context.Background(),
		command.Command{CampaignID: "camp-1", Type: command.Type("sys.daggerheart.gm_fear.set")},
		applier,
		domainwrite.Options{RequireEvents: true, MissingEventMsg: "missing events"},
	)
	if err != nil {
		t.Fatalf("execute and apply with inline enabled: %v", err)
	}
	if applier.calls != 1 {
		t.Fatalf("apply calls = %d, want 1", applier.calls)
	}
}

func TestExecuteAndApplyDomainCommandReturnsApplyErrorWhenInlineEnabled(t *testing.T) {
	runtime := testWriteRuntime(t)
	runtime.SetInlineApplyEnabled(true)

	applier := &fakeEventApplier{err: errors.New("boom")}
	svc := &DaggerheartService{
		stores: Stores{
			Domain: fakeDomainExecutor{
				result: engine.Result{
					Decision: command.Decision{Events: []event.Event{testSystemEvent()}},
				},
			},
			WriteRuntime: runtime,
		},
	}

	_, err := svc.executeAndApplyDomainCommand(
		context.Background(),
		command.Command{CampaignID: "camp-1", Type: command.Type("sys.daggerheart.gm_fear.set")},
		applier,
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

func TestExecuteAndApplyDomainCommandSkipsJournalOnlyInlineApply(t *testing.T) {
	runtime := testWriteRuntime(t)
	runtime.SetInlineApplyEnabled(true)

	applier := &fakeEventApplier{err: errors.New("should not apply")}
	svc := &DaggerheartService{
		stores: Stores{
			Domain: fakeDomainExecutor{
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
			WriteRuntime: runtime,
		},
	}

	_, err := svc.executeAndApplyDomainCommand(
		context.Background(),
		command.Command{CampaignID: "camp-1", Type: command.Type("story.note.add")},
		applier,
		domainwrite.Options{RequireEvents: true, MissingEventMsg: "missing events"},
	)
	if err != nil {
		t.Fatalf("execute and apply with journal-only event: %v", err)
	}
	if applier.calls != 0 {
		t.Fatalf("apply calls = %d, want 0", applier.calls)
	}
}
