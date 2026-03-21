package game

import (
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestStoresValidate(t *testing.T) {
	t.Run("all fields set returns nil", func(t *testing.T) {
		groups := validRootStoreGroups()
		if err := ValidateRootStoreGroups(
			groups.projection,
			groups.system,
			groups.infrastructure,
			groups.content,
			groups.runtime,
		); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("zero value returns error listing all fields", func(t *testing.T) {
		err := ValidateRootStoreGroups(
			ProjectionStores{},
			SystemStores{},
			InfrastructureStores{},
			ContentStores{},
			RuntimeStores{},
		)
		if err == nil {
			t.Fatal("expected error for empty stores")
		}
		msg := err.Error()
		for _, name := range []string{
			"Campaign", "Participant", "ClaimIndex", "Invite",
			"Character", "SystemStores.Daggerheart", "Session", "SessionGate",
			"SessionSpotlight", "SessionInteraction", "Scene", "SceneCharacter",
			"SceneGate", "SceneSpotlight", "SceneInteraction",
			"Event", "Audit", "Statistics",
			"Snapshot", "CampaignFork", "DaggerheartContent",
			"Write.Executor", "Write.Runtime",
		} {
			if !strings.Contains(msg, name) {
				t.Errorf("error should mention %q, got: %s", name, msg)
			}
		}
	})

	t.Run("single nil field returns error", func(t *testing.T) {
		groups := validRootStoreGroups()
		groups.infrastructure.Event = nil
		err := ValidateRootStoreGroups(
			groups.projection,
			groups.system,
			groups.infrastructure,
			groups.content,
			groups.runtime,
		)
		if err == nil {
			t.Fatal("expected error for nil Event store")
		}
		if !strings.Contains(err.Error(), "Event") {
			t.Errorf("error should mention Event, got: %s", err.Error())
		}
	})
}

func TestRootStoreConcernBuilders(t *testing.T) {
	projectionStore := &projectionStoreBundleStub{
		CampaignStore:            gametest.NewFakeCampaignStore(),
		ParticipantStore:         gametest.NewFakeParticipantStore(),
		ClaimIndexStore:          stubClaimIndex{},
		InviteStore:              gametest.NewFakeInviteStore(),
		CharacterStore:           gametest.NewFakeCharacterStore(),
		SessionStore:             gametest.NewFakeSessionStore(),
		SessionGateStore:         &gametest.FakeSessionGateStore{},
		SessionSpotlightStore:    &gametest.FakeSessionSpotlightStore{},
		SceneStore:               stubSceneStore{},
		SceneCharacterStore:      stubSceneCharacterStore{},
		SceneGateStore:           stubSceneGateStore{},
		SceneSpotlightStore:      stubSceneSpotlightStore{},
		CampaignForkStore:        &gametest.FakeCampaignForkStore{},
		StatisticsStore:          &gametest.FakeStatisticsStore{},
		SnapshotStore:            stubSnapshot{},
		ProjectionWatermarkStore: stubProjectionWatermarkStore{},
	}

	systemStores := SystemStores{Daggerheart: &gametest.FakeDaggerheartStore{}}
	infrastructure := NewInfrastructureStores(projectionStore, StoresInfrastructureConfig{
		EventStore: eventAuditStoreStub{
			EventStore:      gametest.NewFakeEventStore(),
			AuditEventStore: stubAudit{},
		},
		AuditStore: stubAudit{},
	})
	content := NewContentStores(StoresContentConfig{
		ContentStore: stubDaggerheartContent{},
	})
	runtime := NewRuntimeStores(StoresRuntimeConfig{
		Domain:       fakeDomainExecutor{},
		WriteRuntime: domainwrite.NewRuntime(),
	}, infrastructure.Audit)
	projection := NewProjectionStores(StoresProjectionConfig{
		ProjectionStore: projectionStore,
		SystemStores:    systemStores,
	})

	if projection.Campaign == nil || projection.Participant == nil || projection.Character == nil {
		t.Fatal("expected projection-backed stores to be populated")
	}
	if systemStores.Daggerheart == nil {
		t.Fatal("expected Daggerheart system store to be set from config")
	}
	if infrastructure.Audit == nil {
		t.Fatal("expected audit store to be propagated explicitly")
	}
	if content.DaggerheartContent == nil {
		t.Fatal("expected content store to be propagated explicitly")
	}
	if runtime.Write.Executor == nil || runtime.Write.Runtime == nil {
		t.Fatal("expected runtime dependencies to be propagated")
	}
}

func TestNewRuntimeStores_AuditWiring(t *testing.T) {
	projectionStore := &projectionStoreBundleStub{
		CampaignStore:            gametest.NewFakeCampaignStore(),
		ParticipantStore:         gametest.NewFakeParticipantStore(),
		ClaimIndexStore:          stubClaimIndex{},
		InviteStore:              gametest.NewFakeInviteStore(),
		CharacterStore:           gametest.NewFakeCharacterStore(),
		SessionStore:             gametest.NewFakeSessionStore(),
		SessionGateStore:         &gametest.FakeSessionGateStore{},
		SessionSpotlightStore:    &gametest.FakeSessionSpotlightStore{},
		SceneStore:               stubSceneStore{},
		SceneCharacterStore:      stubSceneCharacterStore{},
		SceneGateStore:           stubSceneGateStore{},
		SceneSpotlightStore:      stubSceneSpotlightStore{},
		CampaignForkStore:        &gametest.FakeCampaignForkStore{},
		StatisticsStore:          &gametest.FakeStatisticsStore{},
		SnapshotStore:            stubSnapshot{},
		ProjectionWatermarkStore: stubProjectionWatermarkStore{},
	}

	t.Run("explicit audit store is used for stores and write path", func(t *testing.T) {
		explicitAudit := stubAudit{}
		infrastructure := NewInfrastructureStores(projectionStore, StoresInfrastructureConfig{
			EventStore: gametest.NewFakeEventStore(),
			AuditStore: explicitAudit,
		})
		runtime := NewRuntimeStores(StoresRuntimeConfig{}, infrastructure.Audit)

		if _, ok := infrastructure.Audit.(stubAudit); !ok {
			t.Fatalf("infrastructure.Audit type = %T, want %T", infrastructure.Audit, explicitAudit)
		}
		if _, ok := runtime.Write.Audit.(stubAudit); !ok {
			t.Fatalf("runtime.Write.Audit type = %T, want %T", runtime.Write.Audit, explicitAudit)
		}
	})

	t.Run("event store does not imply audit store", func(t *testing.T) {
		infrastructure := NewInfrastructureStores(projectionStore, StoresInfrastructureConfig{
			EventStore: eventAuditStoreStub{
				EventStore:      gametest.NewFakeEventStore(),
				AuditEventStore: stubAudit{},
			},
		})
		runtime := NewRuntimeStores(StoresRuntimeConfig{}, infrastructure.Audit)
		if infrastructure.Audit != nil {
			t.Fatalf("infrastructure.Audit = %T, want nil", infrastructure.Audit)
		}
		if runtime.Write.Audit != nil {
			t.Fatalf("runtime.Write.Audit = %T, want nil", runtime.Write.Audit)
		}
	})
}

type rootStoreGroupsFixture struct {
	projection     ProjectionStores
	system         SystemStores
	infrastructure InfrastructureStores
	content        ContentStores
	runtime        RuntimeStores
}

// validRootStoreGroups returns fully configured root store concerns using
// minimal stubs so validation tests can exercise one missing dependency at a time.
func validRootStoreGroups() rootStoreGroupsFixture {
	return rootStoreGroupsFixture{
		projection: ProjectionStores{
			Campaign:           gametest.NewFakeCampaignStore(),
			Participant:        gametest.NewFakeParticipantStore(),
			ClaimIndex:         stubClaimIndex{},
			Invite:             gametest.NewFakeInviteStore(),
			Character:          gametest.NewFakeCharacterStore(),
			Session:            gametest.NewFakeSessionStore(),
			SessionGate:        &gametest.FakeSessionGateStore{},
			SessionSpotlight:   &gametest.FakeSessionSpotlightStore{},
			SessionInteraction: &gametest.FakeSessionInteractionStore{},
			Scene:              stubSceneStore{},
			SceneCharacter:     stubSceneCharacterStore{},
			SceneGate:          stubSceneGateStore{},
			SceneSpotlight:     stubSceneSpotlightStore{},
			SceneInteraction:   stubSceneInteractionStore{},
			CampaignFork:       &gametest.FakeCampaignForkStore{},
		},
		system: SystemStores{Daggerheart: &gametest.FakeDaggerheartStore{}},
		infrastructure: InfrastructureStores{
			Event:      gametest.NewFakeEventStore(),
			Audit:      stubAudit{},
			Statistics: &gametest.FakeStatisticsStore{},
			Snapshot:   stubSnapshot{},
		},
		content: ContentStores{
			DaggerheartContent: stubDaggerheartContent{},
		},
		runtime: RuntimeStores{
			Write: domainwriteexec.WritePath{Executor: fakeDomainExecutor{}, Runtime: domainwrite.NewRuntime()},
		},
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
type stubSceneInteractionStore struct{ storage.SceneInteractionStore }

type projectionStoreBundleStub struct {
	storage.CampaignStore
	storage.ParticipantStore
	storage.ClaimIndexStore
	storage.InviteStore
	storage.CharacterStore
	storage.SessionStore
	storage.SessionInteractionStore
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
	storage.SceneInteractionStore
}

type eventAuditStoreStub struct {
	storage.EventStore
	storage.AuditEventStore
}
