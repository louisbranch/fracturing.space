package game

import (
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
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
			"Character", "SystemStores.Daggerheart", "Session", "SessionGate",
			"SessionSpotlight", "Scene", "SceneCharacter", "SceneGate",
			"SceneSpotlight", "Event", "Audit", "Statistics",
			"Snapshot", "CampaignFork", "DaggerheartContent",
			"Write.Executor", "Write.Runtime", "Events",
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

func TestNewStoresFromProjection(t *testing.T) {
	projectionStore := &projectionStoreBundleStub{
		CampaignStore:            newFakeCampaignStore(),
		ParticipantStore:         newFakeParticipantStore(),
		ClaimIndexStore:          stubClaimIndex{},
		InviteStore:              newFakeInviteStore(),
		CharacterStore:           newFakeCharacterStore(),
		SessionStore:             newFakeSessionStore(),
		SessionGateStore:         &fakeSessionGateStore{},
		SessionSpotlightStore:    &fakeSessionSpotlightStore{},
		SceneStore:               stubSceneStore{},
		SceneCharacterStore:      stubSceneCharacterStore{},
		SceneGateStore:           stubSceneGateStore{},
		SceneSpotlightStore:      stubSceneSpotlightStore{},
		CampaignForkStore:        &fakeCampaignForkStore{},
		StatisticsStore:          &fakeStatisticsStore{},
		SnapshotStore:            stubSnapshot{},
		ProjectionWatermarkStore: stubProjectionWatermarkStore{},
	}

	stores := NewStoresFromProjection(StoresFromProjectionConfig{
		ProjectionStore: projectionStore,
		SystemStores:    SystemStores{Daggerheart: &fakeDaggerheartStore{}},
		EventStore: eventAuditStoreStub{
			EventStore:      newFakeEventStore(),
			AuditEventStore: stubAudit{},
		},
		ContentStore: stubDaggerheartContent{},
		Domain:       fakeDomainExecutor{},
		WriteRuntime: domainwrite.NewRuntime(),
		Events:       event.NewRegistry(),
	})

	if stores.Campaign == nil || stores.Participant == nil || stores.Character == nil {
		t.Fatal("expected projection-backed stores to be populated")
	}
	if stores.SystemStores.Daggerheart == nil {
		t.Fatal("expected Daggerheart system store to be set from config")
	}
	if stores.Audit == nil {
		t.Fatal("expected audit store to be inferred from event store when compatible")
	}
	if stores.Write.Executor == nil || stores.Write.Runtime == nil || stores.Events == nil {
		t.Fatal("expected runtime dependencies to be propagated")
	}
}

func TestNewStoresFromProjection_AuditStoreSelection(t *testing.T) {
	projectionStore := &projectionStoreBundleStub{
		CampaignStore:            newFakeCampaignStore(),
		ParticipantStore:         newFakeParticipantStore(),
		ClaimIndexStore:          stubClaimIndex{},
		InviteStore:              newFakeInviteStore(),
		CharacterStore:           newFakeCharacterStore(),
		SessionStore:             newFakeSessionStore(),
		SessionGateStore:         &fakeSessionGateStore{},
		SessionSpotlightStore:    &fakeSessionSpotlightStore{},
		SceneStore:               stubSceneStore{},
		SceneCharacterStore:      stubSceneCharacterStore{},
		SceneGateStore:           stubSceneGateStore{},
		SceneSpotlightStore:      stubSceneSpotlightStore{},
		CampaignForkStore:        &fakeCampaignForkStore{},
		StatisticsStore:          &fakeStatisticsStore{},
		SnapshotStore:            stubSnapshot{},
		ProjectionWatermarkStore: stubProjectionWatermarkStore{},
	}

	t.Run("explicit audit store wins", func(t *testing.T) {
		explicitAudit := stubAudit{}
		stores := NewStoresFromProjection(StoresFromProjectionConfig{
			ProjectionStore: projectionStore,
			EventStore:      newFakeEventStore(),
			AuditStore:      explicitAudit,
		})

		if _, ok := stores.Audit.(stubAudit); !ok {
			t.Fatalf("stores.Audit type = %T, want %T", stores.Audit, explicitAudit)
		}
	})

	t.Run("non-audit event store does not infer audit store", func(t *testing.T) {
		stores := NewStoresFromProjection(StoresFromProjectionConfig{
			ProjectionStore: projectionStore,
			EventStore:      newFakeEventStore(),
		})
		if stores.Audit != nil {
			t.Fatalf("stores.Audit = %T, want nil", stores.Audit)
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
		SystemStores:       SystemStores{Daggerheart: &fakeDaggerheartStore{}},
		Session:            newFakeSessionStore(),
		SessionGate:        &fakeSessionGateStore{},
		SessionSpotlight:   &fakeSessionSpotlightStore{},
		Scene:              stubSceneStore{},
		SceneCharacter:     stubSceneCharacterStore{},
		SceneGate:          stubSceneGateStore{},
		SceneSpotlight:     stubSceneSpotlightStore{},
		Event:              newFakeEventStore(),
		Audit:              stubAudit{},
		Statistics:         &fakeStatisticsStore{},
		Snapshot:           stubSnapshot{},
		CampaignFork:       &fakeCampaignForkStore{},
		DaggerheartContent: stubDaggerheartContent{},
		Write:              domainwriteexec.WritePath{Executor: fakeDomainExecutor{}, Runtime: domainwrite.NewRuntime()},
		Events:             event.NewRegistry(),
	}
}

func TestStoresApplier(t *testing.T) {
	s := validStores()
	if err := s.Validate(); err != nil {
		t.Fatalf("validate stores: %v", err)
	}
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
	if applier.Adapters == nil {
		t.Error("expected Adapters to be set")
	}
	if applier.Events == nil {
		t.Error("expected Events registry to be set")
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
type stubAudit struct{ storage.AuditEventStore }
type stubSnapshot struct{ storage.SnapshotStore }
type stubProjectionWatermarkStore struct {
	storage.ProjectionWatermarkStore
}
type stubDaggerheartContent struct {
	contentstore.DaggerheartContentReadStore
}

type stubSceneStore struct{ storage.SceneStore }
type stubSceneCharacterStore struct{ storage.SceneCharacterStore }
type stubSceneGateStore struct{ storage.SceneGateStore }
type stubSceneSpotlightStore struct{ storage.SceneSpotlightStore }

type projectionStoreBundleStub struct {
	storage.CampaignStore
	storage.ParticipantStore
	storage.ClaimIndexStore
	storage.InviteStore
	storage.CharacterStore
	storage.SessionStore
	storage.SnapshotStore
	storage.CampaignForkStore
	storage.StatisticsStore
	storage.ProjectionWatermarkStore
	storage.SessionGateStore
	storage.SessionSpotlightStore
	storage.SceneStore
	storage.SceneCharacterStore
	storage.SceneGateStore
	storage.SceneSpotlightStore
}

type eventAuditStoreStub struct {
	storage.EventStore
	storage.AuditEventStore
}
