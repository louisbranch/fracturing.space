package engine

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

func TestDecisionGate_BranchesWithoutRejection(t *testing.T) {
	t.Run("nil registry", func(t *testing.T) {
		decision := (DecisionGate{}).Check(session.State{GateOpen: true}, command.Command{Type: command.Type("session.start")})
		if len(decision.Rejections) != 0 {
			t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
		}
	})

	registry := command.NewRegistry()
	if err := registry.Register(command.Definition{
		Type:  command.Type("campaign.create"),
		Owner: command.OwnerCore,
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}
	if err := registry.Register(command.Definition{
		Type:  command.Type("session.start"),
		Owner: command.OwnerCore,
		Gate: command.GatePolicy{
			Scope: command.GateScopeSession,
		},
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}
	gate := DecisionGate{Registry: registry}

	t.Run("unknown command", func(t *testing.T) {
		decision := gate.Check(session.State{GateOpen: true}, command.Command{Type: command.Type("unknown")})
		if len(decision.Rejections) != 0 {
			t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
		}
	})

	t.Run("non session scope", func(t *testing.T) {
		decision := gate.Check(session.State{GateOpen: true}, command.Command{Type: command.Type("campaign.create")})
		if len(decision.Rejections) != 0 {
			t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
		}
	})

	t.Run("session gate closed", func(t *testing.T) {
		decision := gate.Check(session.State{GateOpen: false}, command.Command{Type: command.Type("session.start")})
		if len(decision.Rejections) != 0 {
			t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
		}
	})
}

func TestDecisionGate_RejectionMessageWithoutGateID(t *testing.T) {
	registry := command.NewRegistry()
	if err := registry.Register(command.Definition{
		Type:  command.Type("session.start"),
		Owner: command.OwnerCore,
		Gate: command.GatePolicy{
			Scope: command.GateScopeSession,
		},
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}

	decision := (DecisionGate{Registry: registry}).Check(
		session.State{GateOpen: true, GateID: "   "},
		command.Command{Type: command.Type("session.start")},
	)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Message != "session gate is open" {
		t.Fatalf("rejection message = %q, want %q", decision.Rejections[0].Message, "session gate is open")
	}
}

func TestDecisionGateCheckScene_BranchesWithoutRejection(t *testing.T) {
	t.Run("nil registry", func(t *testing.T) {
		decision := (DecisionGate{}).CheckScene(scene.State{GateOpen: true}, command.Command{Type: command.Type("scene.action")})
		if len(decision.Rejections) != 0 {
			t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
		}
	})

	registry := command.NewRegistry()
	if err := registry.Register(command.Definition{
		Type:  command.Type("campaign.create"),
		Owner: command.OwnerCore,
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}
	if err := registry.Register(command.Definition{
		Type:  command.Type("scene.action"),
		Owner: command.OwnerCore,
		Gate: command.GatePolicy{
			Scope: command.GateScopeScene,
		},
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}
	gate := DecisionGate{Registry: registry}

	t.Run("unknown command", func(t *testing.T) {
		decision := gate.CheckScene(scene.State{GateOpen: true}, command.Command{Type: command.Type("unknown")})
		if len(decision.Rejections) != 0 {
			t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
		}
	})

	t.Run("non scene scope", func(t *testing.T) {
		decision := gate.CheckScene(scene.State{GateOpen: true}, command.Command{Type: command.Type("campaign.create")})
		if len(decision.Rejections) != 0 {
			t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
		}
	})
}

func TestDecisionGateCheckScene_RejectionMessageWithoutGateID(t *testing.T) {
	registry := command.NewRegistry()
	if err := registry.Register(command.Definition{
		Type:  command.Type("scene.action"),
		Owner: command.OwnerCore,
		Gate: command.GatePolicy{
			Scope: command.GateScopeScene,
		},
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}

	decision := (DecisionGate{Registry: registry}).CheckScene(
		scene.State{GateOpen: true, GateID: "   "},
		command.Command{Type: command.Type("scene.action")},
	)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Message != "scene gate is open" {
		t.Fatalf("rejection message = %q, want %q", decision.Rejections[0].Message, "scene gate is open")
	}
}
