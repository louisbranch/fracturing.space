package engine

import (
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/system"
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
func (fakeModule) Decider() system.Decider           { return nil }
func (fakeModule) Projector() system.Projector       { return nil }
func (fakeModule) StateFactory() system.StateFactory { return nil }

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
func (m syntheticModule) Decider() system.Decider           { return nil }
func (m syntheticModule) Projector() system.Projector       { return nil }
func (m syntheticModule) StateFactory() system.StateFactory { return nil }

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
