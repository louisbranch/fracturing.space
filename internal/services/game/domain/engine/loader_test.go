package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/checkpoint"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/journal"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/replay"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

func TestReplayStateLoader_LoadsAggregateState(t *testing.T) {
	registry := event.NewRegistry()
	if err := registry.Register(event.Definition{
		Type:  event.Type("session.gate_opened"),
		Owner: event.OwnerCore,
	}); err != nil {
		t.Fatalf("register event: %v", err)
	}
	store := journal.NewMemory(registry)
	_, err := store.Append(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("session.gate_opened"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		PayloadJSON: []byte(`{"gate_id":"gate-1","gate_type":"gm_consequence"}`),
	})
	if err != nil {
		t.Fatalf("append event: %v", err)
	}

	loader := ReplayStateLoader{
		Events:      store,
		Checkpoints: checkpoint.NewMemory(),
		Applier:     &aggregate.Applier{},
		StateFactory: func() any {
			return aggregate.State{}
		},
	}
	state, err := loader.Load(context.Background(), command.Command{CampaignID: "camp-1"})
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	agg, ok := state.(aggregate.State)
	if !ok {
		t.Fatal("expected aggregate.State")
	}
	if !agg.Session.GateOpen {
		t.Fatal("expected gate to be open")
	}
}

func TestReplayGateStateLoader_LoadsSessionState(t *testing.T) {
	registry := event.NewRegistry()
	if err := registry.Register(event.Definition{
		Type:  event.Type("session.gate_opened"),
		Owner: event.OwnerCore,
	}); err != nil {
		t.Fatalf("register event: %v", err)
	}
	store := journal.NewMemory(registry)
	_, err := store.Append(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("session.gate_opened"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		PayloadJSON: []byte(`{"gate_id":"gate-1","gate_type":"gm_consequence"}`),
	})
	if err != nil {
		t.Fatalf("append event: %v", err)
	}

	stateLoader := ReplayStateLoader{
		Events:      store,
		Checkpoints: checkpoint.NewMemory(),
		Applier:     &aggregate.Applier{},
		StateFactory: func() any {
			return aggregate.State{}
		},
	}
	loader := ReplayGateStateLoader{StateLoader: stateLoader}
	state, err := loader.LoadSession(context.Background(), "camp-1", "sess-1")
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if !state.GateOpen {
		t.Fatal("expected gate to be open")
	}
}

type trackingReplayEventStore struct {
	events     []event.Event
	afterCalls []uint64
}

func (s *trackingReplayEventStore) ListEvents(_ context.Context, _ string, afterSeq uint64, limit int) ([]event.Event, error) {
	s.afterCalls = append(s.afterCalls, afterSeq)
	result := make([]event.Event, 0, len(s.events))
	for _, evt := range s.events {
		if evt.Seq > afterSeq {
			result = append(result, evt)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

type fakeSnapshotStore struct {
	state  any
	seq    uint64
	getErr error
}

func (s *fakeSnapshotStore) GetState(_ context.Context, _ string) (any, uint64, error) {
	if s.getErr != nil {
		return nil, 0, s.getErr
	}
	if s.state == nil {
		return nil, 0, replay.ErrCheckpointNotFound
	}
	return s.state, s.seq, nil
}

func (s *fakeSnapshotStore) SaveState(context.Context, string, uint64, any) error {
	return nil
}

func TestReplayStateLoader_SeedsReplayFromSnapshot(t *testing.T) {
	events := &trackingReplayEventStore{}
	snapshots := &fakeSnapshotStore{
		state: aggregate.State{
			Session: session.State{GateID: "gate-1"},
		},
		seq: 7,
	}
	loader := ReplayStateLoader{
		Events:      events,
		Checkpoints: checkpoint.NewMemory(),
		Snapshots:   snapshots,
		Applier:     &aggregate.Applier{},
		StateFactory: func() any {
			return aggregate.State{}
		},
	}

	state, err := loader.Load(context.Background(), command.Command{CampaignID: "camp-1"})
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if len(events.afterCalls) == 0 {
		t.Fatal("expected replay to query events")
	}
	if events.afterCalls[0] != 7 {
		t.Fatalf("first after_seq = %d, want %d", events.afterCalls[0], 7)
	}
	agg, ok := state.(aggregate.State)
	if !ok {
		t.Fatalf("state type = %T, want aggregate.State", state)
	}
	if agg.Session.GateID != "gate-1" {
		t.Fatalf("gate id = %q, want %q", agg.Session.GateID, "gate-1")
	}
}

func TestReplayStateLoader_ReturnsSnapshotLoadError(t *testing.T) {
	loader := ReplayStateLoader{
		Events:      &trackingReplayEventStore{},
		Checkpoints: checkpoint.NewMemory(),
		Snapshots: &fakeSnapshotStore{
			getErr: errors.New("snapshot boom"),
		},
		Applier: &aggregate.Applier{},
		StateFactory: func() any {
			return aggregate.State{}
		},
	}

	_, err := loader.Load(context.Background(), command.Command{CampaignID: "camp-1"})
	if err == nil {
		t.Fatal("expected error")
	}
}
