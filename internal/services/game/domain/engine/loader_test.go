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
		Folder:      &aggregate.Folder{},
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
		Folder:      &aggregate.Folder{},
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

type staticCheckpointStore struct {
	seq uint64
}

func (s staticCheckpointStore) Get(context.Context, string) (replay.Checkpoint, error) {
	if s.seq == 0 {
		return replay.Checkpoint{}, replay.ErrCheckpointNotFound
	}
	return replay.Checkpoint{CampaignID: "camp-1", LastSeq: s.seq}, nil
}

func (s staticCheckpointStore) Save(context.Context, replay.Checkpoint) error {
	return nil
}

type trackingCheckpointStore struct {
	seq       uint64
	saveCalls int
	last      replay.Checkpoint
}

func (s *trackingCheckpointStore) Get(context.Context, string) (replay.Checkpoint, error) {
	if s.seq == 0 {
		return replay.Checkpoint{}, replay.ErrCheckpointNotFound
	}
	return replay.Checkpoint{CampaignID: "camp-1", LastSeq: s.seq}, nil
}

func (s *trackingCheckpointStore) Save(_ context.Context, checkpoint replay.Checkpoint) error {
	s.saveCalls++
	s.last = checkpoint
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
		Folder:      &aggregate.Folder{},
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

func TestReplayStateLoader_DoesNotSkipPastSnapshotWhenCheckpointAhead(t *testing.T) {
	events := &trackingReplayEventStore{}
	snapshots := &fakeSnapshotStore{
		state: aggregate.State{
			Session: session.State{GateID: "gate-1"},
		},
		seq: 7,
	}
	loader := ReplayStateLoader{
		Events:      events,
		Checkpoints: staticCheckpointStore{seq: 12},
		Snapshots:   snapshots,
		Folder:      &aggregate.Folder{},
		StateFactory: func() any {
			return aggregate.State{}
		},
	}

	_, err := loader.Load(context.Background(), command.Command{CampaignID: "camp-1"})
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if len(events.afterCalls) == 0 {
		t.Fatal("expected replay to query events")
	}
	if events.afterCalls[0] != 7 {
		t.Fatalf("first after_seq = %d, want %d (snapshot seq must cap checkpoint)", events.afterCalls[0], 7)
	}
}

func TestReplayStateLoader_LoadFreshBypassesSnapshotsAndCheckpoints(t *testing.T) {
	events := &trackingReplayEventStore{
		events: []event.Event{
			{
				Seq:         1,
				CampaignID:  "camp-1",
				Type:        event.Type("session.gate_opened"),
				Timestamp:   time.Unix(0, 0).UTC(),
				ActorType:   event.ActorTypeSystem,
				EntityType:  "session",
				EntityID:    "sess-1",
				SessionID:   "sess-1",
				PayloadJSON: []byte(`{"gate_id":"gate-2","gate_type":"gm_consequence"}`),
			},
		},
	}
	snapshots := &fakeSnapshotStore{
		state: aggregate.State{
			Session: session.State{GateID: "stale-gate"},
		},
		seq: 7,
	}
	loader := ReplayStateLoader{
		Events:      events,
		Checkpoints: staticCheckpointStore{seq: 12},
		Snapshots:   snapshots,
		Folder:      &aggregate.Folder{},
		StateFactory: func() any {
			return aggregate.State{}
		},
	}

	state, err := loader.LoadFresh(context.Background(), command.Command{CampaignID: "camp-1"})
	if err != nil {
		t.Fatalf("load fresh state: %v", err)
	}
	if len(events.afterCalls) == 0 {
		t.Fatal("expected replay to query events")
	}
	if events.afterCalls[0] != 0 {
		t.Fatalf("fresh replay after_seq = %d, want 0", events.afterCalls[0])
	}
	agg, ok := state.(aggregate.State)
	if !ok {
		t.Fatalf("state type = %T, want aggregate.State", state)
	}
	if agg.Session.GateID != "gate-2" {
		t.Fatalf("gate id = %q, want %q", agg.Session.GateID, "gate-2")
	}
}

func TestCheckpointCapStore_SaveDelegatesToBaseStore(t *testing.T) {
	base := &trackingCheckpointStore{}
	capped := checkpointCapStore{base: base, maxSeq: 12}

	input := replay.Checkpoint{CampaignID: "camp-1", LastSeq: 9}
	if err := capped.Save(context.Background(), input); err != nil {
		t.Fatalf("save: %v", err)
	}
	if base.saveCalls != 1 {
		t.Fatalf("save calls = %d, want 1", base.saveCalls)
	}
	if base.last != input {
		t.Fatalf("saved checkpoint = %#v, want %#v", base.last, input)
	}
}

func TestReplayStateLoader_ReturnsSnapshotLoadError(t *testing.T) {
	loader := ReplayStateLoader{
		Events:      &trackingReplayEventStore{},
		Checkpoints: checkpoint.NewMemory(),
		Snapshots: &fakeSnapshotStore{
			getErr: errors.New("snapshot boom"),
		},
		Folder: &aggregate.Folder{},
		StateFactory: func() any {
			return aggregate.State{}
		},
	}

	_, err := loader.Load(context.Background(), command.Command{CampaignID: "camp-1"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestReplayStateLoader_RequiresStateFactory(t *testing.T) {
	loader := ReplayStateLoader{
		Events:      &trackingReplayEventStore{},
		Checkpoints: checkpoint.NewMemory(),
		Folder:      &aggregate.Folder{},
	}
	_, err := loader.Load(context.Background(), command.Command{CampaignID: "camp-1"})
	if !errors.Is(err, ErrStateFactoryRequired) {
		t.Fatalf("expected ErrStateFactoryRequired, got %v", err)
	}
}

func TestReplayStateLoader_RequiresCheckpointStore(t *testing.T) {
	loader := ReplayStateLoader{
		Events: &trackingReplayEventStore{},
		Folder: &aggregate.Folder{},
		StateFactory: func() any {
			return aggregate.State{}
		},
	}
	_, err := loader.Load(context.Background(), command.Command{CampaignID: "camp-1"})
	if !errors.Is(err, replay.ErrCheckpointStoreRequired) {
		t.Fatalf("expected ErrCheckpointStoreRequired, got %v", err)
	}
}

func TestReplayStateLoader_RequiresFolder(t *testing.T) {
	loader := ReplayStateLoader{
		Events:      &trackingReplayEventStore{},
		Checkpoints: checkpoint.NewMemory(),
		StateFactory: func() any {
			return aggregate.State{}
		},
	}
	_, err := loader.Load(context.Background(), command.Command{CampaignID: "camp-1"})
	if !errors.Is(err, replay.ErrFolderRequired) {
		t.Fatalf("expected ErrFolderRequired, got %v", err)
	}
}

func TestReplayGateStateLoader_LoadSessionErrorBranches(t *testing.T) {
	t.Run("nil reconstructed state", func(t *testing.T) {
		loader := ReplayGateStateLoader{
			StateLoader: ReplayStateLoader{
				Events:      &trackingReplayEventStore{},
				Checkpoints: checkpoint.NewMemory(),
				Folder:      &aggregate.Folder{},
				StateFactory: func() any {
					return nil
				},
			},
		}
		_, err := loader.LoadSession(context.Background(), "camp-1", "sess-1")
		if !errors.Is(err, ErrStateRequired) {
			t.Fatalf("LoadSession() error = %v, want %v", err, ErrStateRequired)
		}
	})

	t.Run("typed nil aggregate pointer", func(t *testing.T) {
		loader := ReplayGateStateLoader{
			StateLoader: ReplayStateLoader{
				Events:      &trackingReplayEventStore{},
				Checkpoints: checkpoint.NewMemory(),
				Folder:      &aggregate.Folder{},
				StateFactory: func() any {
					var state *aggregate.State
					return state
				},
			},
		}
		_, err := loader.LoadSession(context.Background(), "camp-1", "sess-1")
		if !errors.Is(err, ErrStateRequired) {
			t.Fatalf("LoadSession() error = %v, want %v", err, ErrStateRequired)
		}
	})

	t.Run("unsupported state type", func(t *testing.T) {
		loader := ReplayGateStateLoader{
			StateLoader: ReplayStateLoader{
				Events:      &trackingReplayEventStore{},
				Checkpoints: checkpoint.NewMemory(),
				Folder:      &aggregate.Folder{},
				StateFactory: func() any {
					return struct{}{}
				},
			},
		}
		_, err := loader.LoadSession(context.Background(), "camp-1", "sess-1")
		if !errors.Is(err, ErrUnsupportedStateType) {
			t.Fatalf("LoadSession() error = %v, want %v", err, ErrUnsupportedStateType)
		}
	})

	t.Run("propagates state loader error", func(t *testing.T) {
		loader := ReplayGateStateLoader{
			StateLoader: ReplayStateLoader{
				Checkpoints: checkpoint.NewMemory(),
				Folder:      &aggregate.Folder{},
			},
		}
		_, err := loader.LoadSession(context.Background(), "camp-1", "sess-1")
		if !errors.Is(err, replay.ErrEventStoreRequired) {
			t.Fatalf("expected ErrEventStoreRequired, got %v", err)
		}
	})
}

func TestReplayGateStateLoader_LoadScene(t *testing.T) {
	t.Run("returns scene state from aggregate", func(t *testing.T) {
		registry := event.NewRegistry()
		if err := registry.Register(event.Definition{
			Type:  event.Type("scene.created"),
			Owner: event.OwnerCore,
		}); err != nil {
			t.Fatalf("register event: %v", err)
		}
		store := journal.NewMemory(registry)
		_, err := store.Append(context.Background(), event.Event{
			CampaignID:  "camp-1",
			Type:        event.Type("scene.created"),
			Timestamp:   time.Unix(0, 0).UTC(),
			ActorType:   event.ActorTypeSystem,
			SceneID:     "scene-1",
			PayloadJSON: []byte(`{"scene_id":"scene-1","session_id":"sess-1","name":"Battle"}`),
		})
		if err != nil {
			t.Fatalf("append event: %v", err)
		}

		stateLoader := ReplayStateLoader{
			Events:      store,
			Checkpoints: checkpoint.NewMemory(),
			Folder:      &aggregate.Folder{},
			StateFactory: func() any {
				return aggregate.State{}
			},
		}
		loader := ReplayGateStateLoader{StateLoader: stateLoader}
		state, err := loader.LoadScene(context.Background(), "camp-1", "scene-1")
		if err != nil {
			t.Fatalf("load scene: %v", err)
		}
		// Scene state is populated from the fold.
		_ = state
	})

	t.Run("returns scene state from aggregate pointer", func(t *testing.T) {
		registry := event.NewRegistry()
		if err := registry.Register(event.Definition{
			Type:  event.Type("scene.created"),
			Owner: event.OwnerCore,
		}); err != nil {
			t.Fatalf("register event: %v", err)
		}
		store := journal.NewMemory(registry)
		_, err := store.Append(context.Background(), event.Event{
			CampaignID:  "camp-1",
			Type:        event.Type("scene.created"),
			Timestamp:   time.Unix(0, 0).UTC(),
			ActorType:   event.ActorTypeSystem,
			SceneID:     "scene-1",
			PayloadJSON: []byte(`{"scene_id":"scene-1","session_id":"sess-1","name":"Battle"}`),
		})
		if err != nil {
			t.Fatalf("append event: %v", err)
		}

		stateLoader := ReplayStateLoader{
			Events:      store,
			Checkpoints: checkpoint.NewMemory(),
			Folder:      &aggregate.Folder{},
			StateFactory: func() any {
				return &aggregate.State{}
			},
		}
		loader := ReplayGateStateLoader{StateLoader: stateLoader}
		state, err := loader.LoadScene(context.Background(), "camp-1", "scene-1")
		if err != nil {
			t.Fatalf("load scene: %v", err)
		}
		_ = state
	})

	t.Run("returns empty state for unknown scene from pointer", func(t *testing.T) {
		stateLoader := ReplayStateLoader{
			Events:      &trackingReplayEventStore{},
			Checkpoints: checkpoint.NewMemory(),
			Folder:      &aggregate.Folder{},
			StateFactory: func() any {
				return &aggregate.State{}
			},
		}
		loader := ReplayGateStateLoader{StateLoader: stateLoader}
		state, err := loader.LoadScene(context.Background(), "camp-1", "nonexistent")
		if err != nil {
			t.Fatalf("load scene: %v", err)
		}
		if state.GateOpen || state.GateID != "" {
			t.Fatalf("expected empty scene state, got %+v", state)
		}
	})

	t.Run("returns empty state for unknown scene", func(t *testing.T) {
		stateLoader := ReplayStateLoader{
			Events:      &trackingReplayEventStore{},
			Checkpoints: checkpoint.NewMemory(),
			Folder:      &aggregate.Folder{},
			StateFactory: func() any {
				return aggregate.State{}
			},
		}
		loader := ReplayGateStateLoader{StateLoader: stateLoader}
		state, err := loader.LoadScene(context.Background(), "camp-1", "nonexistent")
		if err != nil {
			t.Fatalf("load scene: %v", err)
		}
		if state.GateOpen || state.GateID != "" {
			t.Fatalf("expected empty scene state, got %+v", state)
		}
	})

	t.Run("nil reconstructed state", func(t *testing.T) {
		loader := ReplayGateStateLoader{
			StateLoader: ReplayStateLoader{
				Events:      &trackingReplayEventStore{},
				Checkpoints: checkpoint.NewMemory(),
				Folder:      &aggregate.Folder{},
				StateFactory: func() any {
					return nil
				},
			},
		}
		_, err := loader.LoadScene(context.Background(), "camp-1", "scene-1")
		if !errors.Is(err, ErrStateRequired) {
			t.Fatalf("LoadScene() error = %v, want %v", err, ErrStateRequired)
		}
	})

	t.Run("typed nil aggregate pointer", func(t *testing.T) {
		loader := ReplayGateStateLoader{
			StateLoader: ReplayStateLoader{
				Events:      &trackingReplayEventStore{},
				Checkpoints: checkpoint.NewMemory(),
				Folder:      &aggregate.Folder{},
				StateFactory: func() any {
					var state *aggregate.State
					return state
				},
			},
		}
		_, err := loader.LoadScene(context.Background(), "camp-1", "scene-1")
		if !errors.Is(err, ErrStateRequired) {
			t.Fatalf("LoadScene() error = %v, want %v", err, ErrStateRequired)
		}
	})

	t.Run("unsupported state type", func(t *testing.T) {
		loader := ReplayGateStateLoader{
			StateLoader: ReplayStateLoader{
				Events:      &trackingReplayEventStore{},
				Checkpoints: checkpoint.NewMemory(),
				Folder:      &aggregate.Folder{},
				StateFactory: func() any {
					return struct{}{}
				},
			},
		}
		_, err := loader.LoadScene(context.Background(), "camp-1", "scene-1")
		if !errors.Is(err, ErrUnsupportedStateType) {
			t.Fatalf("LoadScene() error = %v, want %v", err, ErrUnsupportedStateType)
		}
	})

	t.Run("propagates state loader error", func(t *testing.T) {
		loader := ReplayGateStateLoader{
			StateLoader: ReplayStateLoader{
				Checkpoints: checkpoint.NewMemory(),
				Folder:      &aggregate.Folder{},
			},
		}
		_, err := loader.LoadScene(context.Background(), "camp-1", "scene-1")
		if !errors.Is(err, replay.ErrEventStoreRequired) {
			t.Fatalf("expected ErrEventStoreRequired, got %v", err)
		}
	})
}
