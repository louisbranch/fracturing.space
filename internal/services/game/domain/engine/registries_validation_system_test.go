package engine

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

// paramModule is a test module with configurable ID and version.
type paramModule struct {
	id      string
	version string
}

func (m paramModule) ID() string                                 { return m.id }
func (m paramModule) Version() string                            { return m.version }
func (m paramModule) RegisterCommands(_ *command.Registry) error { return nil }
func (m paramModule) RegisterEvents(_ *event.Registry) error     { return nil }
func (m paramModule) EmittableEventTypes() []event.Type          { return nil }
func (m paramModule) Decider() module.Decider                    { return nil }
func (m paramModule) Folder() module.Folder                      { return nil }
func (m paramModule) StateFactory() module.StateFactory          { return nil }

func TestValidateSystemMetadataConsistency_PassesForSystemEventsWithModules(t *testing.T) {
	events := event.NewRegistry()
	if err := events.Register(event.Definition{
		Type:  "sys.alpha.action.tested",
		Owner: event.OwnerSystem,
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	modules := module.NewRegistry()
	if err := modules.Register(paramModule{id: "alpha", version: "v1"}); err != nil {
		t.Fatalf("register module: %v", err)
	}

	if err := ValidateSystemMetadataConsistency(events, modules); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSystemMetadataConsistency_FailsForOrphanedSystemEvent(t *testing.T) {
	events := event.NewRegistry()
	if err := events.Register(event.Definition{
		Type:  "sys.orphan.action.tested",
		Owner: event.OwnerSystem,
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	// No modules registered — the system event has no matching module.
	modules := module.NewRegistry()

	err := ValidateSystemMetadataConsistency(events, modules)
	if err == nil {
		t.Fatal("expected error for orphaned system event")
	}
	if !strings.Contains(err.Error(), "sys.orphan.action.tested") {
		t.Fatalf("expected error to mention event type, got: %v", err)
	}
}

func TestValidateSystemMetadataConsistency_SkipsCoreEvents(t *testing.T) {
	events := event.NewRegistry()
	// Core events should be ignored.
	if err := events.Register(event.Definition{
		Type:  "campaign.created",
		Owner: event.OwnerCore,
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	modules := module.NewRegistry()

	// Should pass even with no modules, because core events are skipped.
	if err := ValidateSystemMetadataConsistency(events, modules); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- ValidateStateFactoryFoldCompatibility tests ---

// compatTestState is a concrete state type for compatibility tests.
type compatTestState struct {
	Value int
}

// compatFactory produces *compatTestState from NewSnapshotState.
type compatFactory struct{}

func (f *compatFactory) NewSnapshotState(_ ids.CampaignID) (any, error) {
	return &compatTestState{Value: 1}, nil
}

func (f *compatFactory) NewCharacterState(_ ids.CampaignID, _ ids.CharacterID, _ string) (any, error) {
	return &compatTestState{}, nil
}

// compatFolder is a Folder backed by a FoldRouter[*compatTestState].
type compatFolder struct {
	router *module.FoldRouter[*compatTestState]
}

func newCompatFolder() *compatFolder {
	assertState := func(state any) (*compatTestState, error) {
		switch v := state.(type) {
		case nil:
			return &compatTestState{}, nil
		case *compatTestState:
			return v, nil
		default:
			return nil, fmt.Errorf("unsupported state type %T", state)
		}
	}
	router := module.NewFoldRouter(assertState)
	module.HandleFold(router, event.Type("sys.compat.tested"), func(s *compatTestState, _ struct{}) error {
		return nil
	})
	return &compatFolder{router: router}
}

func (f *compatFolder) Fold(state any, evt event.Event) (any, error) {
	return f.router.Fold(state, evt)
}

func (f *compatFolder) FoldHandledTypes() []event.Type {
	return f.router.FoldHandledTypes()
}

// incompatibleState is a different type that won't match *compatTestState.
type incompatibleState struct {
	Other string
}

// incompatibleFactory produces *incompatibleState — mismatched with compatFolder.
type incompatibleFactory struct{}

func (f *incompatibleFactory) NewSnapshotState(_ ids.CampaignID) (any, error) {
	return &incompatibleState{Other: "wrong"}, nil
}

func (f *incompatibleFactory) NewCharacterState(_ ids.CampaignID, _ ids.CharacterID, _ string) (any, error) {
	return &incompatibleState{}, nil
}

// compatModule wires a configurable factory and folder for compat tests.
type compatModule struct {
	id      string
	version string
	factory module.StateFactory
	folder  module.Folder
}

func (m *compatModule) ID() string                                 { return m.id }
func (m *compatModule) Version() string                            { return m.version }
func (m *compatModule) RegisterCommands(_ *command.Registry) error { return nil }
func (m *compatModule) RegisterEvents(_ *event.Registry) error     { return nil }
func (m *compatModule) EmittableEventTypes() []event.Type          { return nil }
func (m *compatModule) Decider() module.Decider                    { return nil }
func (m *compatModule) Folder() module.Folder                      { return m.folder }
func (m *compatModule) StateFactory() module.StateFactory          { return m.factory }

func TestValidateStateFactoryFoldCompatibility_PassesWhenTypesMatch(t *testing.T) {
	registry := module.NewRegistry()
	mod := &compatModule{
		id:      "compat-ok",
		version: "v1",
		factory: &compatFactory{},
		folder:  newCompatFolder(),
	}
	if err := registry.Register(mod); err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := ValidateStateFactoryFoldCompatibility(registry); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateStateFactoryFoldCompatibility_FailsWhenTypesMismatch(t *testing.T) {
	registry := module.NewRegistry()
	mod := &compatModule{
		id:      "compat-bad",
		version: "v1",
		factory: &incompatibleFactory{},
		folder:  newCompatFolder(),
	}
	if err := registry.Register(mod); err != nil {
		t.Fatalf("register: %v", err)
	}

	err := ValidateStateFactoryFoldCompatibility(registry)
	if err == nil {
		t.Fatal("expected error for incompatible state factory and fold router")
	}
	if !strings.Contains(err.Error(), "state factory / fold type mismatch") {
		t.Fatalf("expected type mismatch error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "compat-bad@v1") {
		t.Fatalf("expected error to mention module label, got: %v", err)
	}
}

func TestValidateStateFactoryFoldCompatibility_SkipsModulesWithoutFactory(t *testing.T) {
	registry := module.NewRegistry()
	// paramModule returns nil for both StateFactory and Folder.
	if err := registry.Register(paramModule{id: "no-factory", version: "v1"}); err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := ValidateStateFactoryFoldCompatibility(registry); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateStateFactoryFoldCompatibility_SkipsModulesWithoutFolder(t *testing.T) {
	registry := module.NewRegistry()
	mod := &compatModule{
		id:      "no-folder",
		version: "v1",
		factory: &compatFactory{},
		folder:  nil,
	}
	if err := registry.Register(mod); err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := ValidateStateFactoryFoldCompatibility(registry); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateStateFactoryFoldCompatibility_RejectsNilRegistry(t *testing.T) {
	err := ValidateStateFactoryFoldCompatibility(nil)
	if err == nil {
		t.Fatal("expected error for nil registry")
	}
	if !strings.Contains(err.Error(), "module registry is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

type readinessOnlyModule struct {
	paramModule
}

func (m readinessOnlyModule) BindCharacterReadiness(ids.CampaignID, map[module.Key]any) (module.CharacterReadinessEvaluator, error) {
	return staticReadinessEvaluator{}, nil
}

type bootstrapOnlyModule struct {
	paramModule
}

func (m bootstrapOnlyModule) BindSessionStartBootstrap(ids.CampaignID, map[module.Key]any) (module.SessionStartBootstrapEmitter, error) {
	return staticBootstrapEmitter{}, nil
}

type readinessWithFactoryModule struct {
	paramModule
	factory module.StateFactory
}

func (m readinessWithFactoryModule) StateFactory() module.StateFactory {
	return m.factory
}

func (m readinessWithFactoryModule) BindCharacterReadiness(ids.CampaignID, map[module.Key]any) (module.CharacterReadinessEvaluator, error) {
	return staticReadinessEvaluator{}, nil
}

func TestValidateOptionalSystemStateHooks_PassesWithoutHookModules(t *testing.T) {
	registry := module.NewRegistry()
	if err := registry.Register(paramModule{id: "plain", version: "v1"}); err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := ValidateOptionalSystemStateHooks(registry); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateOptionalSystemStateHooks_FailsReadinessHookWithoutFactory(t *testing.T) {
	registry := module.NewRegistry()
	if err := registry.Register(readinessOnlyModule{paramModule{id: "ready", version: "v1"}}); err != nil {
		t.Fatalf("register: %v", err)
	}

	err := ValidateOptionalSystemStateHooks(registry)
	if err == nil {
		t.Fatal("expected error for readiness hook without state factory")
	}
	if !strings.Contains(err.Error(), "CharacterReadinessProvider") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "ready@v1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateOptionalSystemStateHooks_FailsBootstrapHookWithoutFactory(t *testing.T) {
	registry := module.NewRegistry()
	if err := registry.Register(bootstrapOnlyModule{paramModule{id: "bootstrap", version: "v1"}}); err != nil {
		t.Fatalf("register: %v", err)
	}

	err := ValidateOptionalSystemStateHooks(registry)
	if err == nil {
		t.Fatal("expected error for bootstrap hook without state factory")
	}
	if !strings.Contains(err.Error(), "SessionStartBootstrapProvider") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "bootstrap@v1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateOptionalSystemStateHooks_PassesWhenHookModuleHasFactory(t *testing.T) {
	registry := module.NewRegistry()
	if err := registry.Register(readinessWithFactoryModule{
		paramModule: paramModule{id: "ready", version: "v1"},
		factory:     &compatFactory{},
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := ValidateOptionalSystemStateHooks(registry); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateOptionalSystemStateHooks_RejectsNilRegistry(t *testing.T) {
	err := ValidateOptionalSystemStateHooks(nil)
	if err == nil {
		t.Fatal("expected error for nil registry")
	}
	if !strings.Contains(err.Error(), "module registry is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

type staticReadinessEvaluator struct{}

func (staticReadinessEvaluator) CharacterReady(character.State) (bool, string) {
	return true, ""
}

type staticBootstrapEmitter struct{}

func (staticBootstrapEmitter) EmitSessionStartBootstrap(map[ids.CharacterID]character.State, command.Command, time.Time) ([]event.Event, error) {
	return nil, nil
}
