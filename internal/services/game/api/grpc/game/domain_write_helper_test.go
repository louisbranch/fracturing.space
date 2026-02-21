package game

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
)

type fakeDomainExecutor struct {
	result engine.Result
	err    error
}

func (f fakeDomainExecutor) Execute(context.Context, command.Command) (engine.Result, error) {
	return f.result, f.err
}

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

func TestExecuteAndApplyDomainCommand_AppliesEventsByDefault(t *testing.T) {
	SetInlineProjectionApplyEnabled(true)
	domain := fakeDomainExecutor{
		result: engine.Result{
			Decision: command.Decision{Events: []event.Event{testDecisionEvent()}},
		},
	}
	_, err := executeAndApplyDomainCommand(
		context.Background(),
		domain,
		projection.Applier{},
		command.Command{CampaignID: "camp-1", Type: command.Type("campaign.create")},
		domainCommandApplyOptions{requireEvents: true, missingEventMsg: "missing events"},
	)
	if err == nil {
		t.Fatal("expected apply error when inline apply is enabled with unconfigured stores")
	}
}

func TestExecuteAndApplyDomainCommand_SkipsInlineApplyWhenDisabled(t *testing.T) {
	SetInlineProjectionApplyEnabled(false)
	t.Cleanup(func() { SetInlineProjectionApplyEnabled(true) })

	domain := fakeDomainExecutor{
		result: engine.Result{
			Decision: command.Decision{Events: []event.Event{testDecisionEvent()}},
		},
	}
	_, err := executeAndApplyDomainCommand(
		context.Background(),
		domain,
		projection.Applier{},
		command.Command{CampaignID: "camp-1", Type: command.Type("campaign.create")},
		domainCommandApplyOptions{requireEvents: true, missingEventMsg: "missing events"},
	)
	if err != nil {
		t.Fatalf("expected inline apply skip with no error, got %v", err)
	}
}

func TestExecuteAndApplyDomainCommand_SkipsJournalOnlyInlineApply(t *testing.T) {
	SetInlineProjectionApplyEnabled(true)
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
	_, err := executeAndApplyDomainCommand(
		context.Background(),
		domain,
		projection.Applier{},
		command.Command{CampaignID: "camp-1", Type: command.Type("story.note.add")},
		domainCommandApplyOptions{requireEvents: true, missingEventMsg: "missing events"},
	)
	if err != nil {
		t.Fatalf("expected journal-only inline apply skip with no error, got %v", err)
	}
}
