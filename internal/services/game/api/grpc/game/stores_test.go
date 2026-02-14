package game

import (
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestStoresValidate(t *testing.T) {
	t.Run("all fields set returns nil", func(t *testing.T) {
		s := validStores()
		if err := s.Validate(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("zero value returns error listing all fields", func(t *testing.T) {
		s := Stores{}
		err := s.Validate()
		if err == nil {
			t.Fatal("expected error for empty stores")
		}
		msg := err.Error()
		for _, name := range []string{
			"Campaign", "Participant", "ClaimIndex", "Invite",
			"Character", "Daggerheart", "Session", "SessionGate",
			"SessionSpotlight", "Event", "Telemetry", "Statistics",
			"Outcome", "Snapshot", "CampaignFork", "DaggerheartContent",
		} {
			if !strings.Contains(msg, name) {
				t.Errorf("error should mention %q, got: %s", name, msg)
			}
		}
	})

	t.Run("single nil field returns error", func(t *testing.T) {
		s := validStores()
		s.Event = nil
		err := s.Validate()
		if err == nil {
			t.Fatal("expected error for nil Event store")
		}
		if !strings.Contains(err.Error(), "Event") {
			t.Errorf("error should mention Event, got: %s", err.Error())
		}
	})
}

// validStores returns a Stores with all fields populated using minimal stubs.
func validStores() Stores {
	return Stores{
		Campaign:           newFakeCampaignStore(),
		Participant:        newFakeParticipantStore(),
		ClaimIndex:         stubClaimIndex{},
		Invite:             newFakeInviteStore(),
		Character:          newFakeCharacterStore(),
		Daggerheart:        &fakeDaggerheartStore{},
		Session:            newFakeSessionStore(),
		SessionGate:        &fakeSessionGateStore{},
		SessionSpotlight:   &fakeSessionSpotlightStore{},
		Event:              newFakeEventStore(),
		Telemetry:          stubTelemetry{},
		Statistics:         &fakeStatisticsStore{},
		Outcome:            stubRollOutcome{},
		Snapshot:           stubSnapshot{},
		CampaignFork:       &fakeCampaignForkStore{},
		DaggerheartContent: stubDaggerheartContent{},
	}
}

func TestStoresApplier(t *testing.T) {
	s := validStores()
	applier := s.Applier()

	if applier.Campaign == nil {
		t.Error("expected Campaign to be set")
	}
	if applier.Participant == nil {
		t.Error("expected Participant to be set")
	}
	if applier.Character == nil {
		t.Error("expected Character to be set")
	}
	if applier.ClaimIndex == nil {
		t.Error("expected ClaimIndex to be set")
	}
	if applier.Invite == nil {
		t.Error("expected Invite to be set")
	}
	if applier.Daggerheart == nil {
		t.Error("expected Daggerheart to be set")
	}
	if applier.Session == nil {
		t.Error("expected Session to be set")
	}
	if applier.SessionGate == nil {
		t.Error("expected SessionGate to be set")
	}
	if applier.SessionSpotlight == nil {
		t.Error("expected SessionSpotlight to be set")
	}
	if applier.CampaignFork == nil {
		t.Error("expected CampaignFork to be set")
	}
	if applier.Adapters == nil {
		t.Error("expected Adapters to be set")
	}
}

// Minimal stubs for stores that don't have fakes in fakes_test.go.
// These only exist to satisfy non-nil checks in Validate().

type stubClaimIndex struct{ storage.ClaimIndexStore }
type stubTelemetry struct{ storage.TelemetryStore }
type stubRollOutcome struct{ storage.RollOutcomeStore }
type stubSnapshot struct{ storage.SnapshotStore }
type stubDaggerheartContent struct {
	storage.DaggerheartContentStore
}
