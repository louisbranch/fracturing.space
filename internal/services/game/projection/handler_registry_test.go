package projection

import (
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
)

// TestHandlerRegistry_AllEntriesHaveApply verifies that every entry in the
// core router has a non-nil apply function.
func TestHandlerRegistry_AllEntriesHaveApply(t *testing.T) {
	for et, h := range coreRouter.handlers {
		if h.apply == nil {
			t.Errorf("handler for %s has nil apply function", et)
		}
	}
}

func TestCheckMissingStores_AllPresent(t *testing.T) {
	applier := Applier{
		Campaign:           newProjectionCampaignStore(),
		Character:          newFakeCharacterStore(),
		CampaignFork:       newFakeCampaignForkStore(),
		Participant:        newProjectionParticipantStore(),
		Session:            &fakeSessionStore{},
		SessionGate:        newFakeSessionGateStore(),
		SessionSpotlight:   newFakeSessionSpotlightStore(),
		SessionInteraction: newFakeSessionInteractionStore(),
		Scene:              newFakeSceneStore(),
		SceneCharacter:     newFakeSceneCharacterStore(),
		SceneGate:          newFakeSceneGateStore(),
		SceneSpotlight:     newFakeSceneSpotlightStore(),
		SceneInteraction:   newFakeSceneInteractionStore(),
		SceneGMInteraction: newFakeSceneGMInteractionStore(),
		Adapters:           bridge.NewAdapterRegistry(),
	}
	missing := checkMissingStores(storeCampaign|storeCharacter|storeParticipant, applier)
	if len(missing) > 0 {
		t.Fatalf("expected no missing stores, got: %v", missing)
	}
}

func TestCheckMissingStores_SomeMissing(t *testing.T) {
	// Zero-value Applier has all stores nil.
	applier := Applier{}
	missing := checkMissingStores(storeCampaign|storeCharacter, applier)
	if len(missing) != 2 {
		t.Fatalf("expected 2 missing stores, got %d: %v", len(missing), missing)
	}
}

func TestValidateCoreStorePreconditions_ReportsNilStores(t *testing.T) {
	// Zero-value Applier has all stores nil.
	applier := Applier{}
	err := applier.ValidateCoreStorePreconditions()
	if err == nil {
		t.Fatal("expected error for nil stores")
	}
	// Every store required by at least one handler should appear in the error.
	for _, keyword := range []string{"campaign", "character", "participant", "session"} {
		if !strings.Contains(err.Error(), keyword) {
			t.Errorf("expected error to mention %q, got: %v", keyword, err)
		}
	}
}

func TestValidateRuntimePreconditions_PassesWhenAllConfigured(t *testing.T) {
	events := event.NewRegistry()
	if err := events.Register(event.Definition{Type: "sys.test.happened", Owner: event.OwnerSystem}); err != nil {
		t.Fatalf("register system event: %v", err)
	}
	applier := Applier{
		Events:             events,
		Campaign:           newProjectionCampaignStore(),
		Character:          newFakeCharacterStore(),
		CampaignFork:       newFakeCampaignForkStore(),
		Participant:        newProjectionParticipantStore(),
		Session:            &fakeSessionStore{},
		SessionGate:        newFakeSessionGateStore(),
		SessionSpotlight:   newFakeSessionSpotlightStore(),
		SessionInteraction: newFakeSessionInteractionStore(),
		Scene:              newFakeSceneStore(),
		SceneCharacter:     newFakeSceneCharacterStore(),
		SceneGate:          newFakeSceneGateStore(),
		SceneSpotlight:     newFakeSceneSpotlightStore(),
		SceneInteraction:   newFakeSceneInteractionStore(),
		SceneGMInteraction: newFakeSceneGMInteractionStore(),
		Adapters:           bridge.NewAdapterRegistry(),
	}
	if err := applier.ValidateRuntimePreconditions(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateRuntimePreconditions_RequiresAdaptersForSystemEvents(t *testing.T) {
	events := event.NewRegistry()
	if err := events.Register(event.Definition{Type: "sys.test.happened", Owner: event.OwnerSystem}); err != nil {
		t.Fatalf("register system event: %v", err)
	}

	applier := Applier{
		Events:             events,
		Campaign:           newProjectionCampaignStore(),
		Character:          newFakeCharacterStore(),
		CampaignFork:       newFakeCampaignForkStore(),
		Participant:        newProjectionParticipantStore(),
		Session:            &fakeSessionStore{},
		SessionGate:        newFakeSessionGateStore(),
		SessionSpotlight:   newFakeSessionSpotlightStore(),
		SessionInteraction: newFakeSessionInteractionStore(),
		Scene:              newFakeSceneStore(),
		SceneCharacter:     newFakeSceneCharacterStore(),
		SceneGate:          newFakeSceneGateStore(),
		SceneSpotlight:     newFakeSceneSpotlightStore(),
		SceneInteraction:   newFakeSceneInteractionStore(),
		SceneGMInteraction: newFakeSceneGMInteractionStore(),
	}
	err := applier.ValidateRuntimePreconditions()
	if err == nil {
		t.Fatal("expected error for missing system adapters")
	}
	if !strings.Contains(err.Error(), "system adapters") {
		t.Fatalf("error = %v, want system adapters mention", err)
	}
}

func TestRequirements_MapsTypedDependenciesAndEnvelopeFields(t *testing.T) {
	req := requirements(
		needsStores(storeCampaign, storeSessionGate, storeAdapters),
		needsEnvelope(fieldCampaignID, fieldSessionID),
	)

	if req.stores != storeCampaign|storeSessionGate|storeAdapters {
		t.Fatalf("stores = %v, want campaign|session_gate|adapters", req.stores)
	}
	if req.ids != fieldCampaignID|fieldSessionID {
		t.Fatalf("ids = %v, want campaign_id|session_id", req.ids)
	}
}
