package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/replay"
)

type trackingSnapshotStore struct {
	calls      int
	campaignID string
	lastSeq    uint64
	state      any
	err        error
}

func (s *trackingSnapshotStore) GetState(context.Context, string) (any, uint64, error) {
	return nil, 0, replay.ErrCheckpointNotFound
}

func (s *trackingSnapshotStore) SaveState(_ context.Context, campaignID string, lastSeq uint64, state any) error {
	s.calls++
	s.campaignID = campaignID
	s.lastSeq = lastSeq
	s.state = state
	return s.err
}

func TestHandle_RequiresGateStateLoaderForSessionScopedCommand(t *testing.T) {
	registry := command.NewRegistry()
	if err := registry.Register(command.Definition{
		Type:  command.Type("session.start"),
		Owner: command.OwnerCore,
		Gate:  command.GatePolicy{Scope: command.GateScopeSession},
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}

	handler := Handler{
		Commands: registry,
		Gate:     DecisionGate{Registry: registry},
		Decider:  fixedDecider{},
	}

	_, err := handler.Handle(context.Background(), command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("session.start"),
		ActorType:  command.ActorTypeSystem,
		SessionID:  "sess-1",
	})
	if !errors.Is(err, ErrGateStateLoaderRequired) {
		t.Fatalf("expected ErrGateStateLoaderRequired, got %v", err)
	}
}

func TestHandle_PropagatesGateLoaderError(t *testing.T) {
	registry := command.NewRegistry()
	if err := registry.Register(command.Definition{
		Type:  command.Type("session.start"),
		Owner: command.OwnerCore,
		Gate:  command.GatePolicy{Scope: command.GateScopeSession},
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}
	loaderErr := errors.New("gate loader boom")
	handler := Handler{
		Commands:        registry,
		Gate:            DecisionGate{Registry: registry},
		GateStateLoader: fakeGateLoader{err: loaderErr},
		Decider:         fixedDecider{},
	}

	_, err := handler.Handle(context.Background(), command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("session.start"),
		ActorType:  command.ActorTypeSystem,
		SessionID:  "sess-1",
	})
	if !errors.Is(err, loaderErr) {
		t.Fatalf("expected loader error, got %v", err)
	}
}

func TestExecute_SavesSnapshotAfterAppend(t *testing.T) {
	cmdRegistry := command.NewRegistry()
	if err := cmdRegistry.Register(command.Definition{Type: command.Type("action.test"), Owner: command.OwnerCore}); err != nil {
		t.Fatalf("register command: %v", err)
	}
	journal := &fakeJournal{}
	snapshots := &trackingSnapshotStore{}
	handler := Handler{
		Commands:  cmdRegistry,
		Decider:   fixedDecider{decision: command.Accept(event.Event{CampaignID: "camp-1", Type: event.Type("action.tested"), Timestamp: time.Unix(0, 0).UTC(), ActorType: event.ActorTypeSystem, PayloadJSON: []byte(`{}`)})},
		Journal:   journal,
		Snapshots: snapshots,
	}

	result, err := handler.Execute(context.Background(), command.Command{CampaignID: "camp-1", Type: command.Type("action.test"), ActorType: command.ActorTypeSystem})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if snapshots.calls != 1 {
		t.Fatalf("snapshot save calls = %d, want 1", snapshots.calls)
	}
	if snapshots.campaignID != "camp-1" {
		t.Fatalf("snapshot campaign id = %q, want camp-1", snapshots.campaignID)
	}
	if snapshots.lastSeq != 1 {
		t.Fatalf("snapshot seq = %d, want 1", snapshots.lastSeq)
	}
	if snapshots.state != result.State {
		t.Fatal("expected saved snapshot state to match execution result state")
	}
}

func TestExecute_DoesNotSaveCheckpointOrSnapshotForUnsequencedEvents(t *testing.T) {
	cmdRegistry := command.NewRegistry()
	if err := cmdRegistry.Register(command.Definition{Type: command.Type("action.test"), Owner: command.OwnerCore}); err != nil {
		t.Fatalf("register command: %v", err)
	}
	checkpoints := &fakeCheckpointStore{}
	snapshots := &trackingSnapshotStore{}
	handler := Handler{
		Commands:    cmdRegistry,
		Decider:     fixedDecider{decision: command.Accept(event.Event{CampaignID: "camp-1", Type: event.Type("action.tested"), Timestamp: time.Unix(0, 0).UTC(), ActorType: event.ActorTypeSystem, PayloadJSON: []byte(`{}`)})},
		Checkpoints: checkpoints,
		Snapshots:   snapshots,
	}

	_, err := handler.Execute(context.Background(), command.Command{CampaignID: "camp-1", Type: command.Type("action.test"), ActorType: command.ActorTypeSystem})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if checkpoints.calls != 0 {
		t.Fatalf("checkpoint calls = %d, want 0", checkpoints.calls)
	}
	if snapshots.calls != 0 {
		t.Fatalf("snapshot calls = %d, want 0", snapshots.calls)
	}
}

func TestExecute_PropagatesSnapshotSaveError(t *testing.T) {
	cmdRegistry := command.NewRegistry()
	if err := cmdRegistry.Register(command.Definition{Type: command.Type("action.test"), Owner: command.OwnerCore}); err != nil {
		t.Fatalf("register command: %v", err)
	}
	journal := &fakeJournal{}
	snapshotErr := errors.New("snapshot save boom")
	handler := Handler{
		Commands: cmdRegistry,
		Decider:  fixedDecider{decision: command.Accept(event.Event{CampaignID: "camp-1", Type: event.Type("action.tested"), Timestamp: time.Unix(0, 0).UTC(), ActorType: event.ActorTypeSystem, PayloadJSON: []byte(`{}`)})},
		Journal:  journal,
		Snapshots: &trackingSnapshotStore{
			err: snapshotErr,
		},
	}

	_, err := handler.Execute(context.Background(), command.Command{CampaignID: "camp-1", Type: command.Type("action.test"), ActorType: command.ActorTypeSystem})
	if !errors.Is(err, snapshotErr) {
		t.Fatalf("expected snapshot error, got %v", err)
	}
}
