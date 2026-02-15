package engine

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/checkpoint"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/journal"
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
		Applier:     aggregate.Applier{},
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
		Applier:     aggregate.Applier{},
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
