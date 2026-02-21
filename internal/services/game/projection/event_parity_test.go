package projection

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	daggerheartsys "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
)

func TestApplyProjectionRequiredCoreEventsAreHandled(t *testing.T) {
	var unhandled []string
	registries, err := engine.BuildRegistries(daggerheartsys.NewModule())
	if err != nil {
		t.Fatalf("build registries: %v", err)
	}

	coreAdapters := systems.NewAdapterRegistry()
	if err := coreAdapters.Register(daggerheartsys.NewAdapter(newProjectionDaggerheartStore())); err != nil {
		t.Fatalf("register daggerheart adapter: %v", err)
	}
	applier := Applier{
		Events:           registries.Events,
		Campaign:         newProjectionCampaignStore(),
		Character:        newFakeCharacterStore(),
		CampaignFork:     newFakeCampaignForkStore(),
		ClaimIndex:       newFakeClaimIndexStore(),
		Invite:           newFakeInviteStore(),
		Participant:      newProjectionParticipantStore(),
		Session:          &fakeSessionStore{},
		SessionGate:      newFakeSessionGateStore(),
		SessionSpotlight: newFakeSessionSpotlightStore(),
		Adapters:         coreAdapters,
	}

	for _, def := range registries.Events.ListDefinitions() {
		if def.Owner != event.OwnerCore || def.Intent != event.IntentProjectionAndReplay {
			continue
		}
		if err := applier.Apply(context.Background(), baselineProjectionEvent(def.Type)); err != nil &&
			isUnhandledProjectionEventError(err, def.Type) {
			unhandled = append(unhandled, string(def.Type))
		}
	}
	if len(unhandled) > 0 {
		t.Fatalf("core projection-required events without handlers: %s", strings.Join(unhandled, ", "))
	}
}

func TestApplyProjectionRequiredSystemEventsAreHandled(t *testing.T) {
	var unhandled []string
	registries, err := engine.BuildRegistries(daggerheartsys.NewModule())
	if err != nil {
		t.Fatalf("build registries: %v", err)
	}

	adapters := systems.NewAdapterRegistry()
	if err := adapters.Register(daggerheartsys.NewAdapter(newProjectionDaggerheartStore())); err != nil {
		t.Fatalf("register daggerheart adapter: %v", err)
	}

	applier := Applier{
		Adapters: adapters,
	}

	for _, def := range registries.Events.ListDefinitions() {
		if def.Owner != event.OwnerSystem || def.Intent != event.IntentProjectionAndReplay {
			continue
		}

		systemID, err := parseSystemIDFromEventType(def.Type)
		if err != nil {
			t.Fatalf("parse system id for %s: %v", def.Type, err)
		}
		system := registries.Systems.Get(systemID, "")
		if system == nil {
			t.Fatalf("no system module registered for %s", def.Type)
		}
		version := strings.TrimSpace(system.Version())
		if version == "" {
			t.Fatalf("system version required for %s", def.Type)
		}

		evt := baselineProjectionEvent(def.Type)
		evt.SystemID = systemID
		evt.SystemVersion = version
		if err := applier.Apply(context.Background(), evt); err != nil &&
			isUnhandledSystemEventError(err, def.Type) {
			unhandled = append(unhandled, string(def.Type))
		}
	}
	if len(unhandled) > 0 {
		t.Fatalf("system projection-required events without handlers: %s", strings.Join(unhandled, ", "))
	}
}

func baselineProjectionEvent(def event.Type) event.Event {
	return event.Event{
		CampaignID:  "camp-1",
		EntityID:    "entity-1",
		SessionID:   "sess-1",
		Type:        def,
		ActorType:   event.ActorTypeSystem,
		PayloadJSON: []byte("{}"),
		Timestamp:   time.Now().UTC(),
	}
}

func isUnhandledProjectionEventError(err error, eventType event.Type) bool {
	return strings.Contains(err.Error(), fmt.Sprintf("unhandled projection event type: %s", eventType))
}

func isUnhandledSystemEventError(err error, eventType event.Type) bool {
	return strings.Contains(err.Error(), "unhandled") && strings.Contains(err.Error(), string(eventType))
}

func parseSystemIDFromEventType(eventType event.Type) (string, error) {
	parts := strings.Split(string(eventType), ".")
	if len(parts) == 3 {
		if parts[0] != "sys" {
			return "", fmt.Errorf("event type %s is not a system namespace", eventType)
		}
		return parts[1], nil
	}
	if len(parts) < 4 {
		return "", fmt.Errorf("event type %s missing system segment", eventType)
	}
	if parts[0] != "sys" {
		return "", fmt.Errorf("event type %s is not a system namespace", eventType)
	}
	return parts[1], nil
}
