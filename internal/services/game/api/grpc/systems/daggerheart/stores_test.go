package daggerheart

import (
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/manifest"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestStoresValidate_MissingEvents(t *testing.T) {
	s := Stores{
		Campaign:         &fakeCampaignStore{},
		Character:        &fakeCharacterStore{},
		Session:          &fakeSessionStore{},
		SessionGate:      &fakeSessionGateStore{},
		SessionSpotlight: &fakeSessionSpotlightStore{},
		Daggerheart:      &fakeDaggerheartStore{},
		Event:            &fakeEventStore{},
		Write:            domainwriteexec.WritePath{Executor: &fakeDomainEngine{}, Runtime: domainwrite.NewRuntime()},
	}

	err := s.Validate()
	if err == nil {
		t.Fatal("expected validate error when events registry is missing")
	}
	if !strings.Contains(err.Error(), "Events") {
		t.Fatalf("validate error = %v, want mention of Events", err)
	}
}

func TestStoresApplier(t *testing.T) {
	s := Stores{
		Campaign:         &fakeCampaignStore{},
		Character:        &fakeCharacterStore{},
		Session:          &fakeSessionStore{},
		SessionGate:      &fakeSessionGateStore{},
		SessionSpotlight: &fakeSessionSpotlightStore{},
		Daggerheart:      &fakeDaggerheartStore{},
		Event:            &fakeEventStore{},
		Write:            domainwriteexec.WritePath{Executor: &fakeDomainEngine{}, Runtime: domainwrite.NewRuntime()},
		Events:           event.NewRegistry(),
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("validate stores: %v", err)
	}
	applier := s.Applier()

	if applier.Campaign == nil {
		t.Error("expected Campaign to be set")
	}
	if applier.Character == nil {
		t.Error("expected Character to be set")
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
	if applier.Adapters == nil {
		t.Error("expected Adapters to be set")
	}
	if applier.Events == nil {
		t.Error("expected Events registry to be set")
	}
}

func TestNewStoresFromProjection(t *testing.T) {
	projectionStore := &projectionStoreBundleStub{
		CampaignStore:            &fakeCampaignStore{},
		CharacterStore:           &fakeCharacterStore{},
		SessionStore:             &fakeSessionStore{},
		SessionGateStore:         &fakeSessionGateStore{},
		SessionSpotlightStore:    &fakeSessionSpotlightStore{},
		DaggerheartStore:         &fakeDaggerheartStore{},
		ProjectionWatermarkStore: stubProjectionWatermarkStore{},
	}

	stores := NewStoresFromProjection(StoresFromProjectionConfig{
		ProjectionStore: projectionStore,
		EventStore:      &fakeEventStore{},
		ContentStore:    stubDaggerheartContentStore{},
		Domain:          &fakeDomainEngine{},
		WriteRuntime:    domainwrite.NewRuntime(),
		Events:          event.NewRegistry(),
	})

	if stores.Campaign == nil || stores.Character == nil || stores.Daggerheart == nil {
		t.Fatal("expected projection-backed stores to be populated")
	}
	if stores.Event == nil || stores.Write.Runtime == nil || stores.Events == nil {
		t.Fatal("expected runtime stores to be propagated")
	}
	if stores.DaggerheartContent == nil {
		t.Fatal("expected content store to be propagated")
	}
}

// TestStoresAdapterRegistryMatchesManifest verifies that the adapter registry
// built by TryApplier registers the same adapters as the canonical
// manifest.AdapterRegistry, preventing duplication drift.
func TestStoresAdapterRegistryMatchesManifest(t *testing.T) {
	daggerheartStore := &fakeDaggerheartStore{}
	s := Stores{
		Campaign:         &fakeCampaignStore{},
		Character:        &fakeCharacterStore{},
		Session:          &fakeSessionStore{},
		SessionGate:      &fakeSessionGateStore{},
		SessionSpotlight: &fakeSessionSpotlightStore{},
		Daggerheart:      daggerheartStore,
		Event:            &fakeEventStore{},
		Write:            domainwriteexec.WritePath{Executor: &fakeDomainEngine{}, Runtime: domainwrite.NewRuntime()},
		Events:           event.NewRegistry(),
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("validate stores: %v", err)
	}
	applier, err := s.TryApplier()
	if err != nil {
		t.Fatalf("try applier: %v", err)
	}

	manifestRegistry, err := systemmanifest.AdapterRegistry(systemmanifest.ProjectionStores{
		Daggerheart: daggerheartStore,
	})
	if err != nil {
		t.Fatalf("manifest adapter registry: %v", err)
	}

	storesAdapters := applier.Adapters.Adapters()
	manifestAdapters := manifestRegistry.Adapters()

	if len(storesAdapters) != len(manifestAdapters) {
		t.Fatalf("adapter count mismatch: stores=%d, manifest=%d", len(storesAdapters), len(manifestAdapters))
	}
	manifestIDs := make(map[string]string)
	for _, a := range manifestAdapters {
		manifestIDs[a.ID()] = a.Version()
	}
	for _, a := range storesAdapters {
		manifestVersion, ok := manifestIDs[a.ID()]
		if !ok {
			t.Errorf("stores adapter %s not in manifest registry", a.ID())
			continue
		}
		if a.Version() != manifestVersion {
			t.Errorf("adapter %s version mismatch: stores=%s, manifest=%s", a.ID(), a.Version(), manifestVersion)
		}
	}
}

type projectionStoreBundleStub struct {
	storage.CampaignStore
	storage.CharacterStore
	storage.SessionStore
	storage.SessionGateStore
	storage.SessionSpotlightStore
	storage.DaggerheartStore
	storage.ProjectionWatermarkStore
}

type stubProjectionWatermarkStore struct {
	storage.ProjectionWatermarkStore
}
type stubDaggerheartContentStore struct {
	storage.DaggerheartContentReadStore
}
