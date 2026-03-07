package engine

import (
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

func TestDecisionGateRejectsSessionScopedCommandWhenGateOpen(t *testing.T) {
	registry := command.NewRegistry()
	if err := registry.Register(command.Definition{
		Type:  command.Type("action.test"),
		Owner: command.OwnerSystem,
		Gate: command.GatePolicy{
			Scope: command.GateScopeSession,
		},
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}

	gate := DecisionGate{Registry: registry}
	cmd := command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("action.test"),
		SessionID:  "sess-1",
	}
	state := session.State{
		SessionID: "sess-1",
		GateOpen:  true,
		GateID:    "gate-123",
	}

	decision := gate.Check(state, cmd)

	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	rejection := decision.Rejections[0]
	if rejection.Code != "SESSION_GATE_OPEN" {
		t.Fatalf("rejection code = %s, want %s", rejection.Code, "SESSION_GATE_OPEN")
	}
	if !strings.Contains(rejection.Message, "gate-123") {
		t.Fatalf("expected rejection message to include gate id, got %q", rejection.Message)
	}
}

func TestDecisionGateAllowsCommandWhenGatePolicyAllowsOpenGate(t *testing.T) {
	registry := command.NewRegistry()
	if err := registry.Register(command.Definition{
		Type:  command.Type("session.gate_resolve"),
		Owner: command.OwnerCore,
		Gate: command.GatePolicy{
			Scope:         command.GateScopeSession,
			AllowWhenOpen: true,
		},
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}

	gate := DecisionGate{Registry: registry}
	cmd := command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("session.gate_resolve"),
		SessionID:  "sess-1",
	}
	state := session.State{
		SessionID: "sess-1",
		GateOpen:  true,
		GateID:    "gate-123",
	}

	decision := gate.Check(state, cmd)
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
}

func TestDecisionGateCheckSceneRejectsWhenSceneGateOpen(t *testing.T) {
	registry := command.NewRegistry()
	if err := registry.Register(command.Definition{
		Type:  command.Type("scene.gate_open"),
		Owner: command.OwnerCore,
		Gate: command.GatePolicy{
			Scope: command.GateScopeScene,
		},
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}

	gate := DecisionGate{Registry: registry}
	cmd := command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("scene.gate_open"),
		SceneID:    "scene-1",
	}
	state := scene.State{
		GateOpen: true,
		GateID:   "sg-1",
	}

	decision := gate.CheckScene(state, cmd)

	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	rejection := decision.Rejections[0]
	if rejection.Code != rejectionCodeSceneGateOpen {
		t.Fatalf("rejection code = %s, want %s", rejection.Code, rejectionCodeSceneGateOpen)
	}
	if !strings.Contains(rejection.Message, "sg-1") {
		t.Fatalf("expected rejection message to include gate id, got %q", rejection.Message)
	}
}

func TestDecisionGateCheckSceneAllowsWhenPolicyAllowsOpen(t *testing.T) {
	registry := command.NewRegistry()
	if err := registry.Register(command.Definition{
		Type:  command.Type("scene.gate_resolve"),
		Owner: command.OwnerCore,
		Gate: command.GatePolicy{
			Scope:         command.GateScopeScene,
			AllowWhenOpen: true,
		},
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}

	gate := DecisionGate{Registry: registry}
	cmd := command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("scene.gate_resolve"),
		SceneID:    "scene-1",
	}
	state := scene.State{
		GateOpen: true,
		GateID:   "sg-1",
	}

	decision := gate.CheckScene(state, cmd)
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
}

func TestDecisionGateCheckSceneNoopWhenGateClosed(t *testing.T) {
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

	gate := DecisionGate{Registry: registry}
	cmd := command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("scene.action"),
		SceneID:    "scene-1",
	}
	state := scene.State{
		GateOpen: false,
	}

	decision := gate.CheckScene(state, cmd)
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
}
