package game

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeDomainExecutor struct {
	result engine.Result
	err    error
}

func (f fakeDomainExecutor) Execute(context.Context, command.Command) (engine.Result, error) {
	return f.result, f.err
}

type nonRetryableTestError struct {
	err error
}

func (e nonRetryableTestError) Error() string      { return e.err.Error() }
func (e nonRetryableTestError) Unwrap() error      { return e.err }
func (e nonRetryableTestError) NonRetryable() bool { return true }

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

func testWriteRuntime(t *testing.T) *domainwrite.Runtime {
	t.Helper()
	runtime := domainwrite.NewRuntime()

	registry := event.NewRegistry()
	for _, def := range []event.Definition{
		{Type: event.Type("campaign.created"), Owner: event.OwnerCore, Intent: event.IntentProjectionAndReplay},
		{Type: event.Type("story.note_added"), Owner: event.OwnerCore, Intent: event.IntentAuditOnly},
	} {
		if err := registry.Register(def); err != nil {
			t.Fatalf("register event: %v", err)
		}
	}
	runtime.SetIntentFilter(registry)
	return runtime
}

func TestExecuteAndApplyDomainCommand_AppliesEventsByDefault(t *testing.T) {
	runtime := testWriteRuntime(t)
	runtime.SetInlineApplyEnabled(true)
	domain := fakeDomainExecutor{
		result: engine.Result{
			Decision: command.Decision{Events: []event.Event{testDecisionEvent()}},
		},
	}
	stores := Stores{Write: domainwriteexec.WritePath{Executor: domain, Runtime: runtime}}
	_, err := handler.ExecuteAndApplyDomainCommand(
		context.Background(),
		stores.Write,
		projection.Applier{},
		command.Command{CampaignID: "camp-1", Type: command.Type("campaign.create")},
		domainwrite.Options{RequireEvents: true, MissingEventMsg: "missing events"},
	)
	if err == nil {
		t.Fatal("expected apply error when inline apply is enabled with unconfigured stores")
	}
}

func TestExecuteAndApplyDomainCommand_SkipsInlineApplyWhenDisabled(t *testing.T) {
	runtime := testWriteRuntime(t)
	runtime.SetInlineApplyEnabled(false)
	domain := fakeDomainExecutor{
		result: engine.Result{
			Decision: command.Decision{Events: []event.Event{testDecisionEvent()}},
		},
	}
	stores := Stores{Write: domainwriteexec.WritePath{Executor: domain, Runtime: runtime}}
	_, err := handler.ExecuteAndApplyDomainCommand(
		context.Background(),
		stores.Write,
		projection.Applier{},
		command.Command{CampaignID: "camp-1", Type: command.Type("campaign.create")},
		domainwrite.Options{RequireEvents: true, MissingEventMsg: "missing events"},
	)
	if err != nil {
		t.Fatalf("expected inline apply skip with no error, got %v", err)
	}
}

func TestExecuteAndApplyDomainCommand_SkipsJournalOnlyInlineApply(t *testing.T) {
	runtime := testWriteRuntime(t)
	runtime.SetInlineApplyEnabled(true)
	domain := fakeDomainExecutor{
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
	}
	stores := Stores{Write: domainwriteexec.WritePath{Executor: domain, Runtime: runtime}}
	_, err := handler.ExecuteAndApplyDomainCommand(
		context.Background(),
		stores.Write,
		projection.Applier{},
		command.Command{CampaignID: "camp-1", Type: command.Type("story.note.add")},
		domainwrite.Options{RequireEvents: true, MissingEventMsg: "missing events"},
	)
	if err != nil {
		t.Fatalf("expected journal-only inline apply skip with no error, got %v", err)
	}
}

func TestExecuteAndApplyDomainCommand_MapsNonRetryableExecutionError(t *testing.T) {
	runtime := testWriteRuntime(t)
	domain := fakeDomainExecutor{
		err: nonRetryableTestError{err: errors.New("post-persist checkpoint failed")},
	}
	stores := Stores{Write: domainwriteexec.WritePath{Executor: domain, Runtime: runtime}}
	_, err := handler.ExecuteAndApplyDomainCommand(
		context.Background(),
		stores.Write,
		projection.Applier{},
		command.Command{CampaignID: "camp-1", Type: command.Type("campaign.create")},
		domainwrite.Options{},
	)
	if err == nil {
		t.Fatal("expected error")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got %v", err)
	}
	if st.Code() != codes.FailedPrecondition {
		t.Fatalf("code = %s, want %s", st.Code(), codes.FailedPrecondition)
	}
}
