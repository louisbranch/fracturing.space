package engine

import (
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
		Type:  command.Type("action.test"),
		Owner: command.OwnerSystem,
	})
}
func (fakeModule) RegisterEvents(registry *event.Registry) error {
	return registry.Register(event.Definition{
		Type:  event.Type("action.tested"),
		Owner: event.OwnerSystem,
	})
}
func (fakeModule) Decider() system.Decider           { return nil }
func (fakeModule) Projector() system.Projector       { return nil }
func (fakeModule) StateFactory() system.StateFactory { return nil }

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
		PayloadJSON: []byte(`{"session_id":"sess-1"}`),
	})
	if err != nil {
		t.Fatalf("validate core event: %v", err)
	}

	_, err = registries.Commands.ValidateForDecision(command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.test"),
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
		Type:          event.Type("action.tested"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     event.ActorTypeSystem,
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
		PayloadJSON: []byte(`{"request_id":"req-1"}`),
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
		PayloadJSON: []byte(`{"request_id":"req-1"}`),
	})
	if err != nil {
		t.Fatalf("validate action event: %v", err)
	}
}
