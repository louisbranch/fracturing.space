package aggregate

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/system"
)

func TestApplierApply_UpdatesSessionGateState(t *testing.T) {
	applier := Applier{}
	state := State{}

	opened, err := applier.Apply(state, event.Event{
		Type:        event.Type("session.gate_opened"),
		PayloadJSON: []byte(`{"gate_id":"gate-1","gate_type":"gm_consequence"}`),
	})
	if err != nil {
		t.Fatalf("apply gate opened: %v", err)
	}
	updated, ok := opened.(State)
	if !ok {
		t.Fatal("expected State result")
	}
	if !updated.Session.GateOpen {
		t.Fatal("expected gate to be open")
	}
	if updated.Session.GateID != "gate-1" {
		t.Fatalf("gate id = %s, want %s", updated.Session.GateID, "gate-1")
	}

	closed, err := applier.Apply(updated, event.Event{
		Type:        event.Type("session.gate_resolved"),
		PayloadJSON: []byte(`{"gate_id":"gate-1","decision":"approve"}`),
	})
	if err != nil {
		t.Fatalf("apply gate resolved: %v", err)
	}
	closedState, ok := closed.(State)
	if !ok {
		t.Fatal("expected State result")
	}
	if closedState.Session.GateOpen {
		t.Fatal("expected gate to be closed")
	}
}

type fakeSystemProjector struct{}

func (fakeSystemProjector) Apply(state any, _ event.Event) (any, error) {
	count := 0
	if existing, ok := state.(int); ok {
		count = existing
	}
	count++
	return count, nil
}

type fakeSystemModule struct{}

func (fakeSystemModule) ID() string                                 { return "system-1" }
func (fakeSystemModule) Version() string                            { return "v1" }
func (fakeSystemModule) RegisterCommands(_ *command.Registry) error { return nil }
func (fakeSystemModule) RegisterEvents(_ *event.Registry) error     { return nil }
func (fakeSystemModule) Decider() system.Decider                    { return nil }
func (fakeSystemModule) Projector() system.Projector                { return fakeSystemProjector{} }
func (fakeSystemModule) StateFactory() system.StateFactory          { return nil }

func TestApplierApply_RoutesSystemEvents(t *testing.T) {
	registry := system.NewRegistry()
	if err := registry.Register(fakeSystemModule{}); err != nil {
		t.Fatalf("register module: %v", err)
	}
	applier := Applier{SystemRegistry: registry}
	state := State{}
	key := system.Key{ID: "system-1", Version: "v1"}

	updated, err := applier.Apply(state, event.Event{
		Type:          event.Type("action.tested"),
		SystemID:      "system-1",
		SystemVersion: "v1",
	})
	if err != nil {
		t.Fatalf("apply system event: %v", err)
	}
	result, ok := updated.(State)
	if !ok {
		t.Fatal("expected State result")
	}
	systemState, ok := result.Systems[key]
	if !ok {
		t.Fatal("expected system state entry")
	}
	if systemState.(int) != 1 {
		t.Fatalf("system state = %v, want 1", systemState)
	}
}

func TestApplierApply_ReturnsErrorForUnregisteredSystemEvents(t *testing.T) {
	applier := Applier{SystemRegistry: system.NewRegistry()}
	state := State{}

	_, err := applier.Apply(state, event.Event{
		Type:          event.Type("action.unregistered_system"),
		SystemID:      "unregistered-system",
		SystemVersion: "v1",
	})
	if err == nil {
		t.Fatal("expected error for unregistered system event")
	}
}

func TestApplierApply_UpdatesInviteState(t *testing.T) {
	applier := Applier{}
	state := State{}

	updated, err := applier.Apply(state, event.Event{
		Type:        event.Type("invite.created"),
		EntityType:  "invite",
		EntityID:    "inv-1",
		PayloadJSON: []byte(`{"invite_id":"inv-1","participant_id":"p-1","status":"pending"}`),
	})
	if err != nil {
		t.Fatalf("apply invite created: %v", err)
	}
	result, ok := updated.(State)
	if !ok {
		t.Fatal("expected State result")
	}
	if result.Invites == nil {
		t.Fatal("expected invites map to be initialized")
	}
	inv, ok := result.Invites["inv-1"]
	if !ok {
		t.Fatal("expected invite state entry")
	}
	if !inv.Created {
		t.Fatal("expected invite to be marked created")
	}
	if inv.Status != "pending" {
		t.Fatalf("invite status = %s, want %s", inv.Status, "pending")
	}
}
