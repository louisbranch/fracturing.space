package engine

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/replay"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

type fakeGateLoader struct {
	state session.State
	err   error
}

func (f fakeGateLoader) LoadSession(_ context.Context, _, _ string) (session.State, error) {
	return f.state, f.err
}

type spyDecider struct {
	called bool
}

func (s *spyDecider) Decide(_ any, _ command.Command, _ func() time.Time) command.Decision {
	s.called = true
	return command.Decision{}
}

type fixedDecider struct {
	decision command.Decision
}

func (f fixedDecider) Decide(_ any, _ command.Command, _ func() time.Time) command.Decision {
	return f.decision
}

type fakeJournal struct {
	nextSeq uint64
	last    event.Event
}

func (f *fakeJournal) Append(_ context.Context, evt event.Event) (event.Event, error) {
	f.nextSeq++
	stored := evt
	stored.Seq = f.nextSeq
	stored.Hash = fmt.Sprintf("hash-%d", f.nextSeq)
	f.last = stored
	return stored, nil
}

type fakeCheckpointStore struct {
	last  replay.Checkpoint
	calls int
}

func (f *fakeCheckpointStore) Get(_ context.Context, _ string) (replay.Checkpoint, error) {
	return replay.Checkpoint{}, replay.ErrCheckpointNotFound
}

func (f *fakeCheckpointStore) Save(_ context.Context, checkpoint replay.Checkpoint) error {
	f.calls++
	f.last = checkpoint
	return nil
}

type trackingStateLoader struct {
	campaignIDs []string
}

func (t *trackingStateLoader) Load(_ context.Context, cmd command.Command) (any, error) {
	t.campaignIDs = append(t.campaignIDs, cmd.CampaignID)
	return aggregate.State{}, nil
}

func TestHandle_RejectsWhenGateOpen(t *testing.T) {
	registry := command.NewRegistry()
	if err := registry.Register(command.Definition{
		Type:  command.Type("action.test"),
		Owner: command.OwnerCore,
		Gate: command.GatePolicy{
			Scope: command.GateScopeSession,
		},
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}

	decider := &spyDecider{}
	handler := Handler{
		Commands:        registry,
		Gate:            DecisionGate{Registry: registry},
		GateStateLoader: fakeGateLoader{state: session.State{GateOpen: true, GateID: "gate-123"}},
		Decider:         decider,
	}
	cmd := command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("action.test"),
		ActorType:  command.ActorTypeSystem,
		SessionID:  "sess-1",
	}

	decision, err := handler.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if decider.called {
		t.Fatal("expected decider not to be called")
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
}

func TestHandle_ValidatesEventsWithRegistry(t *testing.T) {
	cmdRegistry := command.NewRegistry()
	if err := cmdRegistry.Register(command.Definition{
		Type:  command.Type("action.test"),
		Owner: command.OwnerCore,
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}
	eventRegistry := event.NewRegistry()
	if err := eventRegistry.Register(event.Definition{
		Type:  event.Type("action.tested"),
		Owner: event.OwnerCore,
	}); err != nil {
		t.Fatalf("register event: %v", err)
	}

	decider := fixedDecider{decision: command.Accept(event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("action.tested"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		PayloadJSON: []byte("{"),
	})}
	handler := Handler{
		Commands: cmdRegistry,
		Events:   eventRegistry,
		Decider:  decider,
	}
	cmd := command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("action.test"),
		ActorType:  command.ActorTypeSystem,
	}

	_, err := handler.Handle(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHandle_AppendsEventsWhenJournalConfigured(t *testing.T) {
	cmdRegistry := command.NewRegistry()
	if err := cmdRegistry.Register(command.Definition{
		Type:  command.Type("action.test"),
		Owner: command.OwnerCore,
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}
	eventRegistry := event.NewRegistry()
	if err := eventRegistry.Register(event.Definition{
		Type:  event.Type("action.tested"),
		Owner: event.OwnerCore,
	}); err != nil {
		t.Fatalf("register event: %v", err)
	}

	decider := fixedDecider{decision: command.Accept(event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("action.tested"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		PayloadJSON: []byte(`{"ok":true}`),
	})}
	journal := &fakeJournal{}
	handler := Handler{
		Commands: cmdRegistry,
		Events:   eventRegistry,
		Journal:  journal,
		Decider:  decider,
	}
	cmd := command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("action.test"),
		ActorType:  command.ActorTypeSystem,
	}

	decision, err := handler.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}
	if decision.Events[0].Seq != 1 {
		t.Fatalf("event seq = %d, want %d", decision.Events[0].Seq, 1)
	}
	if journal.last.Seq != 1 {
		t.Fatalf("journal seq = %d, want %d", journal.last.Seq, 1)
	}
}

func TestExecute_AppliesEventsToState(t *testing.T) {
	cmdRegistry := command.NewRegistry()
	if err := cmdRegistry.Register(command.Definition{
		Type:  command.Type("action.test"),
		Owner: command.OwnerCore,
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}

	decider := fixedDecider{decision: command.Accept(event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("session.gate_opened"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		PayloadJSON: []byte(`{"gate_id":"gate-1","gate_type":"gm_consequence"}`),
	})}
	handler := Handler{
		Commands: cmdRegistry,
		Decider:  decider,
		Applier:  aggregate.Applier{},
	}
	cmd := command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("action.test"),
		ActorType:  command.ActorTypeSystem,
	}

	result, err := handler.Execute(context.Background(), cmd)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.Decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(result.Decision.Events))
	}
	state, ok := result.State.(aggregate.State)
	if !ok {
		t.Fatal("expected aggregate.State")
	}
	if !state.Session.GateOpen {
		t.Fatal("expected gate to be open")
	}
}

func TestExecute_SavesCheckpointAfterAppend(t *testing.T) {
	cmdRegistry := command.NewRegistry()
	if err := cmdRegistry.Register(command.Definition{
		Type:  command.Type("action.test"),
		Owner: command.OwnerCore,
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}

	decider := fixedDecider{decision: command.Accept(event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("session.gate_opened"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		PayloadJSON: []byte(`{"gate_id":"gate-1","gate_type":"gm_consequence"}`),
	})}
	journal := &fakeJournal{}
	checkpoints := &fakeCheckpointStore{}
	handler := Handler{
		Commands:    cmdRegistry,
		Decider:     decider,
		Journal:     journal,
		Checkpoints: checkpoints,
	}
	cmd := command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("action.test"),
		ActorType:  command.ActorTypeSystem,
	}

	_, err := handler.Execute(context.Background(), cmd)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if checkpoints.calls != 1 {
		t.Fatalf("checkpoint calls = %d, want %d", checkpoints.calls, 1)
	}
	if checkpoints.last.LastSeq != 1 {
		t.Fatalf("checkpoint seq = %d, want %d", checkpoints.last.LastSeq, 1)
	}
}

func TestExecute_UsesValidatedCommandForAllStateLoads(t *testing.T) {
	cmdRegistry := command.NewRegistry()
	if err := cmdRegistry.Register(command.Definition{
		Type:  command.Type("action.test"),
		Owner: command.OwnerCore,
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}

	loader := &trackingStateLoader{}
	handler := Handler{
		Commands:    cmdRegistry,
		Decider:     fixedDecider{decision: command.Decision{}},
		StateLoader: loader,
	}

	_, err := handler.Execute(context.Background(), command.Command{
		CampaignID: "  camp-1  ",
		Type:       command.Type("action.test"),
		ActorType:  command.ActorTypeSystem,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(loader.campaignIDs) != 1 {
		t.Fatalf("state loader calls = %d, want %d", len(loader.campaignIDs), 1)
	}
	for _, id := range loader.campaignIDs {
		if id != "camp-1" {
			t.Fatalf("state loader campaign id = %q, want %q", id, "camp-1")
		}
	}
}

func TestExecute_SavesCheckpointWithValidatedCampaignID(t *testing.T) {
	cmdRegistry := command.NewRegistry()
	if err := cmdRegistry.Register(command.Definition{
		Type:  command.Type("action.test"),
		Owner: command.OwnerCore,
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}

	journal := &fakeJournal{}
	checkpoints := &fakeCheckpointStore{}
	handler := Handler{
		Commands:    cmdRegistry,
		Decider:     fixedDecider{decision: command.Accept(event.Event{CampaignID: "camp-1", Type: event.Type("action.tested"), Timestamp: time.Unix(0, 0).UTC(), ActorType: event.ActorTypeSystem, PayloadJSON: []byte(`{}`)})},
		Journal:     journal,
		Checkpoints: checkpoints,
	}

	_, err := handler.Execute(context.Background(), command.Command{
		CampaignID: "  camp-1  ",
		Type:       command.Type("action.test"),
		ActorType:  command.ActorTypeSystem,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if checkpoints.calls != 1 {
		t.Fatalf("checkpoint calls = %d, want %d", checkpoints.calls, 1)
	}
	if checkpoints.last.CampaignID != "camp-1" {
		t.Fatalf("checkpoint campaign id = %q, want %q", checkpoints.last.CampaignID, "camp-1")
	}
}
