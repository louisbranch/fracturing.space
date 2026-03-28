package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
)

type fakeSceneGateLoader struct {
	state scene.State
	err   error
}

func (f fakeSceneGateLoader) LoadScene(_ context.Context, _, _ string) (scene.State, error) {
	return f.state, f.err
}

func TestExecute_RejectsWhenSceneGateOpen(t *testing.T) {
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

	decider := &spyDecider{}
	handler := Handler{
		Commands: registry,
		// Intentionally leave Gate.Registry empty to verify handler binds from Commands.
		Gate:                 DecisionGate{},
		SceneGateStateLoader: fakeSceneGateLoader{state: scene.State{GateOpen: true, GateID: "sg-1"}},
		Decider:              decider,
	}
	cmd := command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("scene.gate_open"),
		ActorType:  command.ActorTypeSystem,
		SceneID:    "scene-1",
	}

	result, err := handler.Execute(context.Background(), cmd)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if decider.called {
		t.Fatal("expected decider not to be called")
	}
	if len(result.Decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(result.Decision.Rejections))
	}
	if result.Decision.Rejections[0].Code != RejectionCodeSceneGateOpen {
		t.Errorf("code = %q, want %q", result.Decision.Rejections[0].Code, RejectionCodeSceneGateOpen)
	}
}

func TestExecute_SceneGateRequiresSceneID(t *testing.T) {
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

	handler := Handler{
		Commands:             registry,
		Gate:                 DecisionGate{Registry: registry},
		SceneGateStateLoader: fakeSceneGateLoader{state: scene.State{GateOpen: true, GateID: "sg-1"}},
		Decider:              &spyDecider{},
	}
	cmd := command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("scene.action"),
		ActorType:  command.ActorTypeSystem,
		// No SceneID — scene-scoped command must fail closed.
	}

	_, err := handler.Execute(context.Background(), cmd)
	if !errors.Is(err, ErrSceneIDRequired) {
		t.Fatalf("expected ErrSceneIDRequired, got %v", err)
	}
}

func TestExecute_SceneGateLoaderError(t *testing.T) {
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

	handler := Handler{
		Commands:             registry,
		Gate:                 DecisionGate{Registry: registry},
		SceneGateStateLoader: fakeSceneGateLoader{err: errors.New("load failed")},
		Decider:              &spyDecider{},
	}
	cmd := command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("scene.action"),
		ActorType:  command.ActorTypeSystem,
		SceneID:    "scene-1",
	}

	_, err := handler.Execute(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error from scene gate loader")
	}
}

func TestExecute_SceneGateRequiresLoader(t *testing.T) {
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

	handler := Handler{
		Commands: registry,
		Gate:     DecisionGate{Registry: registry},
		Decider:  &spyDecider{},
	}
	cmd := command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("scene.action"),
		ActorType:  command.ActorTypeSystem,
		SceneID:    "scene-1",
	}

	_, err := handler.Execute(context.Background(), cmd)
	if !errors.Is(err, ErrSceneGateStateLoaderRequired) {
		t.Fatalf("expected ErrSceneGateStateLoaderRequired, got %v", err)
	}
}
