package daggerheart

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

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

func TestExecuteAndApplyDomainCommandRequiresEvents(t *testing.T) {
	SetInlineProjectionApplyEnabled(true)

	svc := &DaggerheartService{
		stores: Stores{
			Domain: fakeDomainExecutor{
				result: engine.Result{Decision: command.Decision{}},
			},
		},
	}

	_, err := svc.executeAndApplyDomainCommand(
		context.Background(),
		command.Command{CampaignID: "camp-1", Type: command.Type("sys.daggerheart.gm_fear.set")},
		&fakeEventApplier{},
		domainCommandApplyOptions{requireEvents: true, missingEventMsg: "missing events"},
	)
	if err == nil {
		t.Fatal("expected missing-event error")
	}
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %s, want %s", status.Code(err), codes.Internal)
	}
	if !strings.Contains(err.Error(), "missing events") {
		t.Fatalf("error = %v, want missing events message", err)
	}
}

func TestExecuteAndApplyDomainCommandSkipsApplyWhenInlineDisabled(t *testing.T) {
	SetInlineProjectionApplyEnabled(false)
	t.Cleanup(func() { SetInlineProjectionApplyEnabled(true) })

	applier := &fakeEventApplier{err: errors.New("should not apply")}
	svc := &DaggerheartService{
		stores: Stores{
			Domain: fakeDomainExecutor{
				result: engine.Result{
					Decision: command.Decision{Events: []event.Event{testSystemEvent()}},
				},
			},
		},
	}

	_, err := svc.executeAndApplyDomainCommand(
		context.Background(),
		command.Command{CampaignID: "camp-1", Type: command.Type("sys.daggerheart.gm_fear.set")},
		applier,
		domainCommandApplyOptions{requireEvents: true, missingEventMsg: "missing events"},
	)
	if err != nil {
		t.Fatalf("execute and apply with inline disabled: %v", err)
	}
	if applier.calls != 0 {
		t.Fatalf("apply calls = %d, want 0", applier.calls)
	}
}

func TestExecuteAndApplyDomainCommandAppliesWhenInlineEnabled(t *testing.T) {
	SetInlineProjectionApplyEnabled(true)

	applier := &fakeEventApplier{}
	svc := &DaggerheartService{
		stores: Stores{
			Domain: fakeDomainExecutor{
				result: engine.Result{
					Decision: command.Decision{Events: []event.Event{testSystemEvent()}},
				},
			},
		},
	}

	_, err := svc.executeAndApplyDomainCommand(
		context.Background(),
		command.Command{CampaignID: "camp-1", Type: command.Type("sys.daggerheart.gm_fear.set")},
		applier,
		domainCommandApplyOptions{requireEvents: true, missingEventMsg: "missing events"},
	)
	if err != nil {
		t.Fatalf("execute and apply with inline enabled: %v", err)
	}
	if applier.calls != 1 {
		t.Fatalf("apply calls = %d, want 1", applier.calls)
	}
}

func TestExecuteAndApplyDomainCommandReturnsApplyErrorWhenInlineEnabled(t *testing.T) {
	SetInlineProjectionApplyEnabled(true)

	applier := &fakeEventApplier{err: errors.New("boom")}
	svc := &DaggerheartService{
		stores: Stores{
			Domain: fakeDomainExecutor{
				result: engine.Result{
					Decision: command.Decision{Events: []event.Event{testSystemEvent()}},
				},
			},
		},
	}

	_, err := svc.executeAndApplyDomainCommand(
		context.Background(),
		command.Command{CampaignID: "camp-1", Type: command.Type("sys.daggerheart.gm_fear.set")},
		applier,
		domainCommandApplyOptions{requireEvents: true, missingEventMsg: "missing events"},
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
	SetInlineProjectionApplyEnabled(true)
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
		},
	}

	_, err := svc.executeAndApplyDomainCommand(
		context.Background(),
		command.Command{CampaignID: "camp-1", Type: command.Type("story.note.add")},
		applier,
		domainCommandApplyOptions{requireEvents: true, missingEventMsg: "missing events"},
	)
	if err != nil {
		t.Fatalf("execute and apply with journal-only event: %v", err)
	}
	if applier.calls != 0 {
		t.Fatalf("apply calls = %d, want 0", applier.calls)
	}
}
