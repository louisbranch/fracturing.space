package module

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

type stubModule struct {
	id      string
	version string
	decider Decider
	folder  Folder
	factory StateFactory
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

func (m stubModule) Folder() Folder {
	return m.folder
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

type stubFolder struct {
	called bool
	state  any
	evt    event.Event
	result any
	err    error
}

func (p *stubFolder) Fold(state any, evt event.Event) (any, error) {
	p.called = true
	p.state = state
	p.evt = evt
	return p.result, p.err
}

func (p *stubFolder) FoldHandledTypes() []event.Type { return nil }

type stubFactory struct {
	snapshotState any
	snapshotErr   error
}

func (f stubFactory) NewSnapshotState(ids.CampaignID) (any, error) {
	if f.snapshotErr != nil {
		return nil, f.snapshotErr
	}
	return f.snapshotState, nil
}

func (f stubFactory) NewCharacterState(ids.CampaignID, ids.CharacterID, string) (any, error) {
	return nil, nil
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
	decision, err := RouteCommand(registry, "state", cmd, time.Now)
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
	_, err := RouteCommand(registry, nil, command.Command{SystemVersion: "v1"}, time.Now)
	if !errors.Is(err, ErrSystemIDRequired) {
		t.Fatalf("expected ErrSystemIDRequired, got %v", err)
	}
}

func TestRouteCommand_MissingDeciderRejected(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(stubModule{id: "daggerheart", version: "v1"}); err != nil {
		t.Fatalf("register module: %v", err)
	}
	_, err := RouteCommand(registry, nil, command.Command{SystemID: "daggerheart", SystemVersion: "v1"}, time.Now)
	if !errors.Is(err, ErrDeciderRequired) {
		t.Fatalf("expected ErrDeciderRequired, got %v", err)
	}
}

func TestRouteEvent_UsesModuleFolder(t *testing.T) {
	registry := NewRegistry()
	folder := &stubFolder{result: "next"}
	if err := registry.Register(stubModule{id: "daggerheart", version: "v1", folder: folder}); err != nil {
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
	if !folder.called {
		t.Fatal("expected folder to be called")
	}
	if folder.state != "state" {
		t.Fatalf("state = %v, want %v", folder.state, "state")
	}
	if state != "next" {
		t.Fatalf("state = %v, want %v", state, "next")
	}
}

func TestRouteCommand_ModuleNotFoundIncludesSystemContext(t *testing.T) {
	registry := NewRegistry()
	// Register a different module so the registry is non-empty.
	if err := registry.Register(stubModule{id: "other", version: "v1"}); err != nil {
		t.Fatalf("register module: %v", err)
	}
	cmd := command.Command{SystemID: "missing-system", SystemVersion: "v2"}
	_, err := RouteCommand(registry, nil, cmd, time.Now)
	if !errors.Is(err, ErrModuleNotFound) {
		t.Fatalf("expected ErrModuleNotFound, got %v", err)
	}
	if !strings.Contains(err.Error(), "missing-system") {
		t.Fatalf("expected error to contain system ID, got %v", err)
	}
	if !strings.Contains(err.Error(), "v2") {
		t.Fatalf("expected error to contain system version, got %v", err)
	}
}

func TestRouteEvent_ModuleNotFoundIncludesSystemContext(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(stubModule{id: "other", version: "v1"}); err != nil {
		t.Fatalf("register module: %v", err)
	}
	evt := event.Event{SystemID: "missing-system", SystemVersion: "v3"}
	_, err := RouteEvent(registry, nil, evt)
	if !errors.Is(err, ErrModuleNotFound) {
		t.Fatalf("expected ErrModuleNotFound, got %v", err)
	}
	if !strings.Contains(err.Error(), "missing-system") {
		t.Fatalf("expected error to contain system ID, got %v", err)
	}
	if !strings.Contains(err.Error(), "v3") {
		t.Fatalf("expected error to contain system version, got %v", err)
	}
}

func TestRouteEvent_MissingFolderRejected(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(stubModule{id: "daggerheart", version: "v1"}); err != nil {
		t.Fatalf("register module: %v", err)
	}
	_, err := RouteEvent(registry, nil, event.Event{SystemID: "daggerheart", SystemVersion: "v1"})
	if !errors.Is(err, ErrFolderRequired) {
		t.Fatalf("expected ErrFolderRequired, got %v", err)
	}
}

func TestResolveSnapshotState_SeedsMissingStateFromFactory(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(stubModule{
		id:      "daggerheart",
		version: "v1",
		factory: stubFactory{snapshotState: "seeded"},
	}); err != nil {
		t.Fatalf("register module: %v", err)
	}

	mod, state, err := ResolveSnapshotState(registry, "camp-1", "daggerheart", "v1", nil)
	if err != nil {
		t.Fatalf("ResolveSnapshotState() error = %v", err)
	}
	if mod == nil {
		t.Fatal("expected resolved module")
	}
	if state != "seeded" {
		t.Fatalf("seeded state = %v, want %v", state, "seeded")
	}
}

func TestResolveSnapshotState_PreservesExistingState(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(stubModule{
		id:      "daggerheart",
		version: "v1",
		factory: stubFactory{snapshotState: "seeded"},
	}); err != nil {
		t.Fatalf("register module: %v", err)
	}

	_, state, err := ResolveSnapshotState(registry, "camp-1", "daggerheart", "v1", "existing")
	if err != nil {
		t.Fatalf("ResolveSnapshotState() error = %v", err)
	}
	if state != "existing" {
		t.Fatalf("state = %v, want %v", state, "existing")
	}
}

func TestResolveSnapshotState_WrapsFactoryError(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(stubModule{
		id:      "daggerheart",
		version: "v1",
		factory: stubFactory{snapshotErr: errors.New("boom")},
	}); err != nil {
		t.Fatalf("register module: %v", err)
	}

	_, _, err := ResolveSnapshotState(registry, "camp-1", "daggerheart", "v1", nil)
	if err == nil {
		t.Fatal("expected factory error")
	}
	if !strings.Contains(err.Error(), "daggerheart@v1") {
		t.Fatalf("error = %v, want module coordinates", err)
	}
	if !strings.Contains(err.Error(), "NewSnapshotState") {
		t.Fatalf("error = %v, want StateFactory context", err)
	}
}

func TestResolveCharacterReadiness_SeedsMissingStateFromFactory(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(readinessHookModule{
		stubModule: stubModule{
			id:      "daggerheart",
			version: "v1",
			factory: stubFactory{snapshotState: "seeded"},
		},
		ready:  false,
		reason: "seeded state checked",
	}); err != nil {
		t.Fatalf("register module: %v", err)
	}

	evaluator, enabled, err := ResolveCharacterReadiness(
		registry,
		"camp-1",
		"daggerheart",
		nil,
	)
	if err != nil {
		t.Fatalf("ResolveCharacterReadiness() error = %v", err)
	}
	if !enabled {
		t.Fatal("expected readiness hook to be enabled")
	}
	ready, reason := evaluator.CharacterReady(character.State{CharacterID: "char-1"})
	if ready || reason != "seeded state checked" {
		t.Fatalf("result = (%t, %q), want (false, %q)", ready, reason, "seeded state checked")
	}
}

func TestResolveCharacterReadiness_DisabledWhenHookMissing(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(stubModule{id: "daggerheart", version: "v1"}); err != nil {
		t.Fatalf("register module: %v", err)
	}

	evaluator, enabled, err := ResolveCharacterReadiness(
		registry,
		"camp-1",
		"daggerheart",
		nil,
	)
	if err != nil {
		t.Fatalf("ResolveCharacterReadiness() error = %v", err)
	}
	if enabled {
		t.Fatal("expected readiness hook to be disabled")
	}
	if evaluator != nil {
		t.Fatalf("evaluator = %v, want nil", evaluator)
	}
}

func TestResolveCharacterReadiness_PreservesExistingVersionedState(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(readinessHookModule{
		stubModule: stubModule{
			id:      "daggerheart",
			version: "v1",
			factory: stubFactory{snapshotState: "seeded"},
		},
		ready:  true,
		reason: "existing state checked",
	}); err != nil {
		t.Fatalf("register module: %v", err)
	}

	evaluator, enabled, err := ResolveCharacterReadiness(
		registry,
		"camp-1",
		"daggerheart",
		map[Key]any{{ID: "daggerheart", Version: "v1"}: "existing"},
	)
	if err != nil {
		t.Fatalf("ResolveCharacterReadiness() error = %v", err)
	}
	ready, reason := evaluator.CharacterReady(character.State{CharacterID: "char-1"})
	if !enabled || !ready || reason != "existing state checked" {
		t.Fatalf("result = (%t, %t, %q), want (true, true, %q)", enabled, ready, reason, "existing state checked")
	}
}

func TestResolveCharacterReadiness_ReportsBindError(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(readinessHookModule{
		stubModule: stubModule{
			id:      "daggerheart",
			version: "v1",
			factory: stubFactory{snapshotErr: errors.New("boom")},
		},
	}); err != nil {
		t.Fatalf("register module: %v", err)
	}

	evaluator, enabled, err := ResolveCharacterReadiness(
		registry,
		"camp-1",
		"daggerheart",
		nil,
	)
	if err == nil {
		t.Fatal("expected bind error")
	}
	if !enabled {
		t.Fatal("expected readiness hook to be enabled")
	}
	if evaluator != nil {
		t.Fatalf("evaluator = %v, want nil", evaluator)
	}
}

func TestResolveSessionStartBootstrap_SeedsMissingStateFromFactory(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(bootstrapHookModule{
		stubModule: stubModule{
			id:      "daggerheart",
			version: "v1",
			factory: stubFactory{snapshotState: "seeded"},
		},
		events: []event.Event{{Type: event.Type("sys.daggerheart.bootstrapped")}},
	}); err != nil {
		t.Fatalf("register module: %v", err)
	}

	emitter, enabled, err := ResolveSessionStartBootstrap(
		registry,
		"camp-1",
		"daggerheart",
		nil,
	)
	if err != nil {
		t.Fatalf("ResolveSessionStartBootstrap() error = %v", err)
	}
	if !enabled {
		t.Fatal("expected bootstrap hook to be enabled")
	}
	events, err := emitter.EmitSessionStartBootstrap(
		map[ids.CharacterID]character.State{"char-1": {CharacterID: "char-1"}},
		command.Command{CampaignID: "camp-1"},
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("EmitSessionStartBootstrap() error = %v", err)
	}
	if len(events) != 1 || events[0].Type != event.Type("sys.daggerheart.bootstrapped") {
		t.Fatalf("events = %v, want one sys.daggerheart.bootstrapped event", events)
	}
}

func TestResolveSessionStartBootstrap_PreservesExistingVersionedState(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(bootstrapHookModule{
		stubModule: stubModule{
			id:      "daggerheart",
			version: "v1",
			factory: stubFactory{snapshotState: "seeded"},
		},
		events: []event.Event{{Type: event.Type("sys.daggerheart.bootstrapped.existing")}},
	}); err != nil {
		t.Fatalf("register module: %v", err)
	}

	emitter, enabled, err := ResolveSessionStartBootstrap(
		registry,
		"camp-1",
		"daggerheart",
		map[Key]any{{ID: "daggerheart", Version: "v1"}: "existing"},
	)
	if err != nil {
		t.Fatalf("ResolveSessionStartBootstrap() error = %v", err)
	}
	if !enabled {
		t.Fatal("expected bootstrap hook to be enabled")
	}
	events, err := emitter.EmitSessionStartBootstrap(
		map[ids.CharacterID]character.State{"char-1": {CharacterID: "char-1"}},
		command.Command{CampaignID: "camp-1"},
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("EmitSessionStartBootstrap() error = %v", err)
	}
	if len(events) != 1 || events[0].Type != event.Type("sys.daggerheart.bootstrapped.existing") {
		t.Fatalf("events = %v, want one sys.daggerheart.bootstrapped.existing event", events)
	}
}

func TestResolveSessionStartBootstrap_DisabledWhenHookMissing(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(stubModule{id: "daggerheart", version: "v1"}); err != nil {
		t.Fatalf("register module: %v", err)
	}

	emitter, enabled, err := ResolveSessionStartBootstrap(
		registry,
		"camp-1",
		"daggerheart",
		nil,
	)
	if err != nil {
		t.Fatalf("ResolveSessionStartBootstrap() error = %v", err)
	}
	if enabled {
		t.Fatal("expected bootstrap hook to be disabled")
	}
	if emitter != nil {
		t.Fatalf("emitter = %v, want nil", emitter)
	}
}

type readinessHookModule struct {
	stubModule
	ready  bool
	reason string
}

func (m readinessHookModule) BindCharacterReadiness(campaignID ids.CampaignID, currentByKey map[Key]any) (CharacterReadinessEvaluator, error) {
	systemState := currentByKey[Key{ID: m.id, Version: m.version}]
	if systemState == nil {
		if m.factory == nil {
			return nil, errors.New("missing state factory")
		}
		seeded, err := m.factory.NewSnapshotState(campaignID)
		if err != nil {
			return nil, err
		}
		systemState = seeded
	}
	switch systemState {
	case "seeded", "existing":
		return readinessHookEvaluator{ready: m.ready, reason: m.reason}, nil
	default:
		return nil, errors.New("unexpected system state")
	}
}

type bootstrapHookModule struct {
	stubModule
	events []event.Event
	err    error
}

func (m bootstrapHookModule) BindSessionStartBootstrap(campaignID ids.CampaignID, currentByKey map[Key]any) (SessionStartBootstrapEmitter, error) {
	systemState := currentByKey[Key{ID: m.id, Version: m.version}]
	if systemState == nil {
		if m.factory == nil {
			return nil, errors.New("missing state factory")
		}
		seeded, err := m.factory.NewSnapshotState(campaignID)
		if err != nil {
			return nil, err
		}
		systemState = seeded
	}
	if systemState != "seeded" && systemState != "existing" {
		return nil, errors.New("unexpected system state")
	}
	return bootstrapHookEmitter{events: m.events, err: m.err}, nil
}

type readinessHookEvaluator struct {
	ready  bool
	reason string
}

func (e readinessHookEvaluator) CharacterReady(character.State) (bool, string) {
	return e.ready, e.reason
}

type bootstrapHookEmitter struct {
	events []event.Event
	err    error
}

func (e bootstrapHookEmitter) EmitSessionStartBootstrap(map[ids.CharacterID]character.State, command.Command, time.Time) ([]event.Event, error) {
	return e.events, e.err
}
