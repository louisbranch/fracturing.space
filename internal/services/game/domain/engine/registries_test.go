package engine

import (
	"context"
	"strings"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
)

type fakeModule struct{}

func (fakeModule) ID() string      { return "system-1" }
func (fakeModule) Version() string { return "v1" }
func (fakeModule) RegisterCommands(registry *command.Registry) error {
	return registry.Register(command.Definition{
		Type:  command.Type("sys.system_1.action.test"),
		Owner: command.OwnerSystem,
	})
}
func (fakeModule) RegisterEvents(registry *event.Registry) error {
	return registry.Register(event.Definition{
		Type:  event.Type("sys.system_1.action.tested"),
		Owner: event.OwnerSystem,
	})
}
func (fakeModule) EmittableEventTypes() []event.Type {
	return []event.Type{event.Type("sys.system_1.action.tested")}
}
func (fakeModule) Decider() module.Decider           { return nil }
func (fakeModule) Projector() module.Projector       { return nil }
func (fakeModule) StateFactory() module.StateFactory { return nil }

type syntheticModule struct {
	id          string
	version     string
	commandType command.Type
	eventType   event.Type
}

func (m syntheticModule) ID() string      { return m.id }
func (m syntheticModule) Version() string { return m.version }
func (m syntheticModule) RegisterCommands(registry *command.Registry) error {
	return registry.Register(command.Definition{
		Type:  m.commandType,
		Owner: command.OwnerSystem,
	})
}
func (m syntheticModule) RegisterEvents(registry *event.Registry) error {
	return registry.Register(event.Definition{
		Type:  m.eventType,
		Owner: event.OwnerSystem,
	})
}
func (m syntheticModule) EmittableEventTypes() []event.Type {
	return []event.Type{m.eventType}
}
func (m syntheticModule) Decider() module.Decider           { return nil }
func (m syntheticModule) Projector() module.Projector       { return nil }
func (m syntheticModule) StateFactory() module.StateFactory { return nil }

func TestBuildRegistries_RegistersCoreAndSystem(t *testing.T) {
	registries, err := BuildRegistries(fakeModule{})
	if err != nil {
		t.Fatalf("build registries: %v", err)
	}

	_, err = registries.Commands.ValidateForDecision(command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("session.start"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"session_id":"sess-1"}`),
	})
	if err != nil {
		t.Fatalf("validate core command: %v", err)
	}

	_, err = registries.Events.ValidateForAppend(event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("session.started"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "session",
		EntityID:    "sess-1",
		PayloadJSON: []byte(`{"session_id":"sess-1"}`),
	})
	if err != nil {
		t.Fatalf("validate core event: %v", err)
	}

	_, err = registries.Commands.ValidateForDecision(command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.system_1.action.test"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      "system-1",
		SystemVersion: "v1",
		PayloadJSON:   []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("validate system command: %v", err)
	}

	_, err = registries.Events.ValidateForAppend(event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("sys.system_1.action.tested"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     event.ActorTypeSystem,
		EntityType:    "action",
		EntityID:      "entity-1",
		SystemID:      "system-1",
		SystemVersion: "v1",
		PayloadJSON:   []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("validate system event: %v", err)
	}
}

func TestBuildRegistries_RegistersInvite(t *testing.T) {
	registries, err := BuildRegistries()
	if err != nil {
		t.Fatalf("build registries: %v", err)
	}

	_, err = registries.Commands.ValidateForDecision(command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("invite.create"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"invite_id":"inv-1","participant_id":"p-1"}`),
	})
	if err != nil {
		t.Fatalf("validate invite command: %v", err)
	}

	_, err = registries.Events.ValidateForAppend(event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("invite.created"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "invite",
		EntityID:    "inv-1",
		PayloadJSON: []byte(`{"invite_id":"inv-1","participant_id":"p-1","status":"pending"}`),
	})
	if err != nil {
		t.Fatalf("validate invite event: %v", err)
	}
}

func TestBuildRegistries_RegistersAction(t *testing.T) {
	registries, err := BuildRegistries()
	if err != nil {
		t.Fatalf("build registries: %v", err)
	}

	_, err = registries.Commands.ValidateForDecision(command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("action.roll.resolve"),
		ActorType:   command.ActorTypeSystem,
		SessionID:   "sess-1",
		PayloadJSON: []byte(`{"request_id":"req-1","roll_seq":1}`),
	})
	if err != nil {
		t.Fatalf("validate action command: %v", err)
	}

	_, err = registries.Events.ValidateForAppend(event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("action.roll_resolved"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		SessionID:   "sess-1",
		EntityType:  "roll",
		EntityID:    "req-1",
		PayloadJSON: []byte(`{"request_id":"req-1","roll_seq":1}`),
	})
	if err != nil {
		t.Fatalf("validate action event: %v", err)
	}
}

func TestBuildRegistries_SyntheticModulesNoTypeCollision(t *testing.T) {
	alpha := syntheticModule{
		id:          "GAME_SYSTEM_ALPHA",
		version:     "v1",
		commandType: command.Type("sys.alpha.action.attack.resolve"),
		eventType:   event.Type("sys.alpha.action.attack_resolved"),
	}
	beta := syntheticModule{
		id:          "GAME_SYSTEM_BETA",
		version:     "v1",
		commandType: command.Type("sys.beta.action.attack.resolve"),
		eventType:   event.Type("sys.beta.action.attack_resolved"),
	}

	registries, err := BuildRegistries(alpha, beta)
	if err != nil {
		t.Fatalf("build registries with synthetic modules: %v", err)
	}
	if registries.Systems.Get(alpha.id, alpha.version) == nil {
		t.Fatalf("expected module %s@%s to be registered", alpha.id, alpha.version)
	}
	if registries.Systems.Get(beta.id, beta.version) == nil {
		t.Fatalf("expected module %s@%s to be registered", beta.id, beta.version)
	}
}

func TestBuildRegistries_SyntheticModulesDetectTypeCollision(t *testing.T) {
	alpha := syntheticModule{
		id:          "GAME_SYSTEM_ALPHA",
		version:     "v1",
		commandType: command.Type("sys.alpha.action.attack.resolve"),
		eventType:   event.Type("sys.alpha.action.attack_resolved"),
	}
	beta := syntheticModule{
		id:          "GAME_SYSTEM_ALPHA",
		version:     "v2",
		commandType: command.Type("sys.alpha.action.attack.resolve"),
		eventType:   event.Type("sys.alpha.action.attack_resolved"),
	}

	_, err := BuildRegistries(alpha, beta)
	if err == nil {
		t.Fatal("expected duplicate command/event type registration error")
	}
	if !strings.Contains(err.Error(), "already registered") {
		t.Fatalf("expected duplicate registration error, got %v", err)
	}
}

func TestValidateFoldCoverage_CoreProjectionEventsHaveFoldHandlers(t *testing.T) {
	registries, err := BuildRegistries()
	if err != nil {
		t.Fatalf("build registries: %v", err)
	}

	if err := ValidateFoldCoverage(registries.Events); err != nil {
		t.Fatalf("fold coverage validation failed: %v", err)
	}
}

func TestValidateFoldCoverage_ReturnsErrorForMissingHandler(t *testing.T) {
	eventRegistry := event.NewRegistry()
	// Register a core projection-and-replay event with no fold handler.
	if err := eventRegistry.Register(event.Definition{
		Type:   event.Type("test.unhandled"),
		Owner:  event.OwnerCore,
		Intent: event.IntentProjectionAndReplay,
	}); err != nil {
		t.Fatalf("register event: %v", err)
	}

	err := ValidateFoldCoverage(eventRegistry)
	if err == nil {
		t.Fatal("expected error for unhandled projection event")
	}
	if !strings.Contains(err.Error(), "test.unhandled") {
		t.Fatalf("expected error to mention test.unhandled, got %v", err)
	}
}

func TestValidateFoldCoverage_IgnoresAuditOnlyEvents(t *testing.T) {
	eventRegistry := event.NewRegistry()
	// Register a core audit-only event — should not require a fold handler.
	if err := eventRegistry.Register(event.Definition{
		Type:   event.Type("test.audit_only"),
		Owner:  event.OwnerCore,
		Intent: event.IntentAuditOnly,
	}); err != nil {
		t.Fatalf("register event: %v", err)
	}

	if err := ValidateFoldCoverage(eventRegistry); err != nil {
		t.Fatalf("expected no error for audit-only event, got: %v", err)
	}
}

func TestValidateFoldCoverage_IgnoresSystemEvents(t *testing.T) {
	eventRegistry := event.NewRegistry()
	// Register a system event — fold coverage for system events is the
	// responsibility of the module projector, not core fold functions.
	if err := eventRegistry.Register(event.Definition{
		Type:   event.Type("sys.test.some_event"),
		Owner:  event.OwnerSystem,
		Intent: event.IntentProjectionAndReplay,
	}); err != nil {
		t.Fatalf("register event: %v", err)
	}

	if err := ValidateFoldCoverage(eventRegistry); err != nil {
		t.Fatalf("expected no error for system event, got: %v", err)
	}
}

func TestValidateProjectionCoverage_ReturnsErrorForMissingHandler(t *testing.T) {
	eventRegistry := event.NewRegistry()
	if err := eventRegistry.Register(event.Definition{
		Type:   event.Type("test.unhandled_projection"),
		Owner:  event.OwnerCore,
		Intent: event.IntentProjectionAndReplay,
	}); err != nil {
		t.Fatalf("register event: %v", err)
	}

	// Pass an empty handled-types list so the unhandled event is detected.
	err := ValidateProjectionCoverage(eventRegistry, nil)
	if err == nil {
		t.Fatal("expected error for unhandled projection event")
	}
	if !strings.Contains(err.Error(), "test.unhandled_projection") {
		t.Fatalf("expected error to mention test.unhandled_projection, got %v", err)
	}
}

func TestValidateProjectionCoverage_PassesWhenHandled(t *testing.T) {
	eventRegistry := event.NewRegistry()
	if err := eventRegistry.Register(event.Definition{
		Type:   event.Type("test.handled"),
		Owner:  event.OwnerCore,
		Intent: event.IntentProjectionAndReplay,
	}); err != nil {
		t.Fatalf("register event: %v", err)
	}

	err := ValidateProjectionCoverage(eventRegistry, []event.Type{event.Type("test.handled")})
	if err != nil {
		t.Fatalf("expected no error when handler exists, got: %v", err)
	}
}

func TestValidateFoldCoverage_RequiresHandlerForReplayOnlyEvents(t *testing.T) {
	eventRegistry := event.NewRegistry()
	if err := eventRegistry.Register(event.Definition{
		Type:   event.Type("test.replay_only"),
		Owner:  event.OwnerCore,
		Intent: event.IntentReplayOnly,
	}); err != nil {
		t.Fatalf("register event: %v", err)
	}

	err := ValidateFoldCoverage(eventRegistry)
	if err == nil {
		t.Fatal("expected error for unhandled replay-only event")
	}
	if !strings.Contains(err.Error(), "test.replay_only") {
		t.Fatalf("expected error to mention test.replay_only, got %v", err)
	}
}

func TestValidateProjectionCoverage_IgnoresReplayOnlyEvents(t *testing.T) {
	eventRegistry := event.NewRegistry()
	if err := eventRegistry.Register(event.Definition{
		Type:   event.Type("test.replay_only"),
		Owner:  event.OwnerCore,
		Intent: event.IntentReplayOnly,
	}); err != nil {
		t.Fatalf("register event: %v", err)
	}

	// No projection handler provided, but should pass because replay-only
	// events don't need projection handlers.
	if err := ValidateProjectionCoverage(eventRegistry, nil); err != nil {
		t.Fatalf("expected no error for replay-only event, got: %v", err)
	}
}

func TestValidateProjectionCoverage_IgnoresAuditOnlyEvents(t *testing.T) {
	eventRegistry := event.NewRegistry()
	if err := eventRegistry.Register(event.Definition{
		Type:   event.Type("test.audit_only"),
		Owner:  event.OwnerCore,
		Intent: event.IntentAuditOnly,
	}); err != nil {
		t.Fatalf("register event: %v", err)
	}

	if err := ValidateProjectionCoverage(eventRegistry, nil); err != nil {
		t.Fatalf("expected no error for audit-only event, got: %v", err)
	}
}

func TestValidateProjectionCoverage_IgnoresSystemEvents(t *testing.T) {
	eventRegistry := event.NewRegistry()
	if err := eventRegistry.Register(event.Definition{
		Type:   event.Type("sys.test.some_event"),
		Owner:  event.OwnerSystem,
		Intent: event.IntentProjectionAndReplay,
	}); err != nil {
		t.Fatalf("register event: %v", err)
	}

	if err := ValidateProjectionCoverage(eventRegistry, nil); err != nil {
		t.Fatalf("expected no error for system event, got: %v", err)
	}
}

func TestValidateAdapterEventCoverage_PassesWhenAllCovered(t *testing.T) {
	registries, err := BuildRegistries(fakeModule{})
	if err != nil {
		t.Fatalf("build registries: %v", err)
	}
	adapters := buildFakeAdapterRegistry(t, []event.Type{
		event.Type("sys.system_1.action.tested"),
	})

	if err := ValidateAdapterEventCoverage(registries.Systems, adapters, registries.Events); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateAdapterEventCoverage_FailsOnMissingHandler(t *testing.T) {
	registries, err := BuildRegistries(fakeModule{})
	if err != nil {
		t.Fatalf("build registries: %v", err)
	}
	// Adapter handles zero types — module emits one.
	adapters := buildFakeAdapterRegistry(t, nil)

	err = ValidateAdapterEventCoverage(registries.Systems, adapters, registries.Events)
	if err == nil {
		t.Fatal("expected error for uncovered emittable event type")
	}
	if !strings.Contains(err.Error(), "sys.system_1.action.tested") {
		t.Fatalf("expected error to mention uncovered type, got: %v", err)
	}
}

type fakeAdapterForCoverage struct {
	id           commonv1.GameSystem
	version      string
	handledTypes []event.Type
}

func (a fakeAdapterForCoverage) ID() commonv1.GameSystem                           { return a.id }
func (a fakeAdapterForCoverage) Version() string                                   { return a.version }
func (a fakeAdapterForCoverage) Apply(_ context.Context, _ event.Event) error      { return nil }
func (a fakeAdapterForCoverage) Snapshot(_ context.Context, _ string) (any, error) { return nil, nil }
func (a fakeAdapterForCoverage) HandledTypes() []event.Type                        { return a.handledTypes }

func buildFakeAdapterRegistry(t *testing.T, handledTypes []event.Type) *systems.AdapterRegistry {
	t.Helper()
	registry := systems.NewAdapterRegistry()
	if err := registry.Register(fakeAdapterForCoverage{
		id:           commonv1.GameSystem(999),
		version:      "v1",
		handledTypes: handledTypes,
	}); err != nil {
		t.Fatalf("register fake adapter: %v", err)
	}
	return registry
}

type unregisteredEmitModule struct {
	syntheticModule
	extraEmittable event.Type
}

func (m unregisteredEmitModule) EmittableEventTypes() []event.Type {
	return []event.Type{m.eventType, m.extraEmittable}
}

func TestBuildRegistries_ValidatesEmittableEventTypes(t *testing.T) {
	mod := unregisteredEmitModule{
		syntheticModule: syntheticModule{
			id:          "GAME_SYSTEM_ALPHA",
			version:     "v1",
			commandType: command.Type("sys.alpha.action.attack.resolve"),
			eventType:   event.Type("sys.alpha.action.attack_resolved"),
		},
		extraEmittable: event.Type("sys.alpha.action.not_registered"),
	}

	_, err := BuildRegistries(mod)
	if err == nil {
		t.Fatal("expected error for unregistered emittable event type")
	}
	if !strings.Contains(err.Error(), "sys.alpha.action.not_registered") {
		t.Fatalf("expected error to mention unregistered type, got: %v", err)
	}
}

func TestBuildRegistries_PassesWhenEmittableEventsAllRegistered(t *testing.T) {
	mod := syntheticModule{
		id:          "GAME_SYSTEM_ALPHA",
		version:     "v1",
		commandType: command.Type("sys.alpha.action.attack.resolve"),
		eventType:   event.Type("sys.alpha.action.attack_resolved"),
	}

	_, err := BuildRegistries(mod)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestBuildRegistries_SyntheticModuleRejectsLegacyActionPrefix(t *testing.T) {
	legacy := syntheticModule{
		id:          "GAME_SYSTEM_ALPHA",
		version:     "v1",
		commandType: command.Type("sys.daggerheart.attack.resolve"),
		eventType:   event.Type("sys.daggerheart.attack_resolved"),
	}

	_, err := BuildRegistries(legacy)
	if err == nil {
		t.Fatal("expected system prefix validation error")
	}
	if !strings.Contains(err.Error(), "sys.alpha.") {
		t.Fatalf("expected sys.alpha prefix guidance, got %v", err)
	}
}
