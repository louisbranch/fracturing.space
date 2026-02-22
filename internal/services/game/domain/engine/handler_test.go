package engine

import (
	"context"
	"errors"
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

func (f *fakeJournal) BatchAppend(_ context.Context, events []event.Event) ([]event.Event, error) {
	stored := make([]event.Event, len(events))
	for i, evt := range events {
		f.nextSeq++
		stored[i] = evt
		stored[i].Seq = f.nextSeq
		stored[i].Hash = fmt.Sprintf("hash-%d", f.nextSeq)
		f.last = stored[i]
	}
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

type systemMutationApplier struct {
	Mutate bool
}

func (a *systemMutationApplier) Apply(_ any, _ event.Event) (any, error) {
	if a.Mutate {
		return map[string]int{"mutated": 1}, nil
	}
	return nil, nil
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
		Decider:     fixedDecider{decision: command.Accept(event.Event{CampaignID: "camp-1", Type: event.Type("action.tested"), Timestamp: time.Unix(0, 0).UTC(), ActorType: event.ActorTypeSystem, PayloadJSON: []byte(`{}`)})},
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

func TestExecute_FailsWhenDeciderReturnsNoEventsWithoutRejections(t *testing.T) {
	cmdRegistry := command.NewRegistry()
	if err := cmdRegistry.Register(command.Definition{
		Type:  command.Type("action.test"),
		Owner: command.OwnerCore,
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}

	handler := Handler{
		Commands: cmdRegistry,
		Decider: fixedDecider{
			decision: command.Decision{},
		},
	}

	_, err := handler.Execute(context.Background(), command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("action.test"),
		ActorType:  command.ActorTypeSystem,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrCommandMustMutate) {
		t.Fatalf("expected ErrCommandMustMutate, got %v", err)
	}
}

func TestExecute_AllowsSystemEventWithNoMutation(t *testing.T) {
	cmdRegistry := command.NewRegistry()
	if err := cmdRegistry.Register(command.Definition{
		Type:  command.Type("sys.test.noop"),
		Owner: command.OwnerSystem,
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}
	eventRegistry := event.NewRegistry()
	if err := eventRegistry.Register(event.Definition{
		Type:  event.Type("sys.test.noop"),
		Owner: event.OwnerSystem,
	}); err != nil {
		t.Fatalf("register event: %v", err)
	}

	handler := Handler{
		Commands: cmdRegistry,
		Events:   eventRegistry,
		Decider: fixedDecider{decision: command.Accept(event.Event{
			CampaignID:    "camp-1",
			Type:          event.Type("sys.test.noop"),
			EntityType:    "character",
			EntityID:      "char-1",
			SystemID:      "test",
			SystemVersion: "1.0.0",
			Timestamp:     time.Unix(0, 0).UTC(),
			ActorType:     event.ActorTypeSystem,
			PayloadJSON:   []byte(`{}`),
		})},
		Applier: &systemMutationApplier{},
	}

	_, err := handler.Execute(context.Background(), command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.test.noop"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      "test",
		SystemVersion: "1.0.0",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
}

func TestExecute_AllowsMutatingSystemEvent(t *testing.T) {
	cmdRegistry := command.NewRegistry()
	if err := cmdRegistry.Register(command.Definition{
		Type:  command.Type("sys.test.mutates"),
		Owner: command.OwnerSystem,
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}
	eventRegistry := event.NewRegistry()
	if err := eventRegistry.Register(event.Definition{
		Type:  event.Type("sys.test.mutates"),
		Owner: event.OwnerSystem,
	}); err != nil {
		t.Fatalf("register event: %v", err)
	}

	handler := Handler{
		Commands: cmdRegistry,
		Events:   eventRegistry,
		Decider: fixedDecider{decision: command.Accept(event.Event{
			CampaignID:    "camp-1",
			Type:          event.Type("sys.test.mutates"),
			EntityType:    "character",
			EntityID:      "char-1",
			SystemID:      "test",
			SystemVersion: "1.0.0",
			Timestamp:     time.Unix(0, 0).UTC(),
			ActorType:     event.ActorTypeSystem,
			PayloadJSON:   []byte(`{}`),
		})},
		Applier: &systemMutationApplier{Mutate: true},
	}

	result, err := handler.Execute(context.Background(), command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.test.mutates"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      "test",
		SystemVersion: "1.0.0",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	mutated, ok := result.State.(map[string]int)
	if !ok {
		t.Fatalf("expected map state, got %T", result.State)
	}
	if mutated["mutated"] != 1 {
		t.Fatalf("expected mutated state marker, got %v", mutated["mutated"])
	}
}

type batchTrackingJournal struct {
	appendCalls      int
	batchAppendCalls int
	nextSeq          uint64
}

func (j *batchTrackingJournal) Append(_ context.Context, evt event.Event) (event.Event, error) {
	j.appendCalls++
	j.nextSeq++
	stored := evt
	stored.Seq = j.nextSeq
	stored.Hash = fmt.Sprintf("hash-%d", j.nextSeq)
	return stored, nil
}

func (j *batchTrackingJournal) BatchAppend(_ context.Context, events []event.Event) ([]event.Event, error) {
	j.batchAppendCalls++
	stored := make([]event.Event, len(events))
	for i, evt := range events {
		j.nextSeq++
		stored[i] = evt
		stored[i].Seq = j.nextSeq
		stored[i].Hash = fmt.Sprintf("hash-%d", j.nextSeq)
	}
	return stored, nil
}

type failingJournal struct{}

func (failingJournal) Append(_ context.Context, _ event.Event) (event.Event, error) {
	return event.Event{}, errors.New("journal unavailable")
}

func (failingJournal) BatchAppend(_ context.Context, _ []event.Event) ([]event.Event, error) {
	return nil, errors.New("journal unavailable")
}

type spyApplier struct {
	called bool
}

func (s *spyApplier) Apply(state any, _ event.Event) (any, error) {
	s.called = true
	return state, nil
}

func TestExecute_DoesNotApplyWhenJournalFails(t *testing.T) {
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

	applier := &spyApplier{}
	handler := Handler{
		Commands: cmdRegistry,
		Events:   eventRegistry,
		Decider: fixedDecider{decision: command.Accept(event.Event{
			CampaignID:  "camp-1",
			Type:        event.Type("action.tested"),
			Timestamp:   time.Unix(0, 0).UTC(),
			ActorType:   event.ActorTypeSystem,
			PayloadJSON: []byte(`{"ok":true}`),
		})},
		Journal: failingJournal{},
		Applier: applier,
	}

	_, err := handler.Execute(context.Background(), command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("action.test"),
		ActorType:  command.ActorTypeSystem,
	})
	if err == nil {
		t.Fatal("expected error from failing journal")
	}
	if applier.called {
		t.Fatal("expected applier not to be called when journal fails")
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

func TestHandle_BatchAppendsAllEventsAtOnce(t *testing.T) {
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

	journal := &batchTrackingJournal{}
	handler := Handler{
		Commands: cmdRegistry,
		Events:   eventRegistry,
		Journal:  journal,
		Decider: fixedDecider{decision: command.Decision{Events: []event.Event{
			{CampaignID: "camp-1", Type: event.Type("action.tested"), Timestamp: time.Unix(0, 0).UTC(), ActorType: event.ActorTypeSystem, PayloadJSON: []byte(`{"a":1}`)},
			{CampaignID: "camp-1", Type: event.Type("action.tested"), Timestamp: time.Unix(1, 0).UTC(), ActorType: event.ActorTypeSystem, PayloadJSON: []byte(`{"a":2}`)},
		}}},
	}

	decision, err := handler.Handle(context.Background(), command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("action.test"),
		ActorType:  command.ActorTypeSystem,
	})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if len(decision.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(decision.Events))
	}
	if journal.batchAppendCalls != 1 {
		t.Fatalf("batch append calls = %d, want 1", journal.batchAppendCalls)
	}
	if journal.appendCalls != 0 {
		t.Fatalf("individual append calls = %d, want 0", journal.appendCalls)
	}
	if decision.Events[0].Seq != 1 || decision.Events[1].Seq != 2 {
		t.Fatalf("event seqs = [%d, %d], want [1, 2]", decision.Events[0].Seq, decision.Events[1].Seq)
	}
}
