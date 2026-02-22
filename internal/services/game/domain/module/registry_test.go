package module

import (
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

type stubModule struct {
	id        string
	version   string
	decider   Decider
	projector Projector
	factory   StateFactory
}

func (m stubModule) ID() string {
	return m.id
}

func (m stubModule) Version() string {
	return m.version
}

func (m stubModule) RegisterCommands(*command.Registry) error {
	return nil
}

func (m stubModule) RegisterEvents(*event.Registry) error {
	return nil
}

func (m stubModule) EmittableEventTypes() []event.Type {
	return nil
}

func (m stubModule) Decider() Decider {
	return m.decider
}

func (m stubModule) Projector() Projector {
	return m.projector
}

func (m stubModule) StateFactory() StateFactory {
	return m.factory
}

type stubDecider struct {
	called   bool
	state    any
	cmd      command.Command
	decision command.Decision
}

func (d *stubDecider) Decide(state any, cmd command.Command, now func() time.Time) command.Decision {
	d.called = true
	d.state = state
	d.cmd = cmd
	return d.decision
}

type stubProjector struct {
	called bool
	state  any
	evt    event.Event
	result any
	err    error
}

func (p *stubProjector) Apply(state any, evt event.Event) (any, error) {
	p.called = true
	p.state = state
	p.evt = evt
	return p.result, p.err
}

func TestRegistryRegister_RequiresSystemID(t *testing.T) {
	registry := NewRegistry()
	err := registry.Register(stubModule{id: "", version: "v1"})
	if !errors.Is(err, ErrSystemIDRequired) {
		t.Fatalf("expected ErrSystemIDRequired, got %v", err)
	}
}

func TestRegistryGet_UsesDefaultVersion(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(stubModule{id: "daggerheart", version: "v1"}); err != nil {
		t.Fatalf("register module v1: %v", err)
	}
	if err := registry.Register(stubModule{id: "daggerheart", version: "legacy"}); err != nil {
		t.Fatalf("register module legacy: %v", err)
	}

	module := registry.Get("daggerheart", "")
	if module == nil {
		t.Fatal("expected module")
	}
	if module.Version() != "v1" {
		t.Fatalf("version = %s, want %s", module.Version(), "v1")
	}
}

func TestRouteCommand_UsesModuleDecider(t *testing.T) {
	registry := NewRegistry()
	decider := &stubDecider{decision: command.Accept(event.Event{Type: event.Type("system.event")})}
	if err := registry.Register(stubModule{id: "daggerheart", version: "v1", decider: decider}); err != nil {
		t.Fatalf("register module: %v", err)
	}

	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("system.test"),
		SystemID:      "daggerheart",
		SystemVersion: "v1",
	}
	decision, err := RouteCommand(registry, "state", cmd, nil)
	if err != nil {
		t.Fatalf("route command: %v", err)
	}
	if !decider.called {
		t.Fatal("expected decider to be called")
	}
	if decider.state != "state" {
		t.Fatalf("state = %v, want %v", decider.state, "state")
	}
	if len(decision.Events) != 1 {
		t.Fatalf("events = %d, want %d", len(decision.Events), 1)
	}
}

func TestRouteCommand_MissingSystemIDRejected(t *testing.T) {
	registry := NewRegistry()
	_, err := RouteCommand(registry, nil, command.Command{SystemVersion: "v1"}, nil)
	if !errors.Is(err, ErrSystemIDRequired) {
		t.Fatalf("expected ErrSystemIDRequired, got %v", err)
	}
}

func TestRouteCommand_MissingDeciderRejected(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(stubModule{id: "daggerheart", version: "v1"}); err != nil {
		t.Fatalf("register module: %v", err)
	}
	_, err := RouteCommand(registry, nil, command.Command{SystemID: "daggerheart", SystemVersion: "v1"}, nil)
	if !errors.Is(err, ErrDeciderRequired) {
		t.Fatalf("expected ErrDeciderRequired, got %v", err)
	}
}

func TestRouteEvent_UsesModuleProjector(t *testing.T) {
	registry := NewRegistry()
	projector := &stubProjector{result: "next"}
	if err := registry.Register(stubModule{id: "daggerheart", version: "v1", projector: projector}); err != nil {
		t.Fatalf("register module: %v", err)
	}

	evt := event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("system.event"),
		SystemID:      "daggerheart",
		SystemVersion: "v1",
	}
	state, err := RouteEvent(registry, "state", evt)
	if err != nil {
		t.Fatalf("route event: %v", err)
	}
	if !projector.called {
		t.Fatal("expected projector to be called")
	}
	if projector.state != "state" {
		t.Fatalf("state = %v, want %v", projector.state, "state")
	}
	if state != "next" {
		t.Fatalf("state = %v, want %v", state, "next")
	}
}

func TestRouteEvent_MissingProjectorRejected(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(stubModule{id: "daggerheart", version: "v1"}); err != nil {
		t.Fatalf("register module: %v", err)
	}
	_, err := RouteEvent(registry, nil, event.Event{SystemID: "daggerheart", SystemVersion: "v1"})
	if !errors.Is(err, ErrProjectorRequired) {
		t.Fatalf("expected ErrProjectorRequired, got %v", err)
	}
}
