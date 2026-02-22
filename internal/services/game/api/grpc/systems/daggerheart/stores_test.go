package daggerheart

import (
	"testing"

	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/manifest"
)

func TestStoresApplier(t *testing.T) {
	s := Stores{
		Campaign:         &fakeCampaignStore{},
		Character:        &fakeCharacterStore{},
		Domain:           &fakeDomainEngine{},
		Session:          &fakeSessionStore{},
		SessionGate:      &fakeSessionGateStore{},
		SessionSpotlight: &fakeSessionSpotlightStore{},
		Daggerheart:      &fakeDaggerheartStore{},
		Event:            &fakeEventStore{},
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
}

// TestStoresAdapterRegistryMatchesManifest verifies that the adapter registry
// built by TryApplier registers the same adapters as the canonical
// manifest.AdapterRegistry, preventing duplication drift.
func TestStoresAdapterRegistryMatchesManifest(t *testing.T) {
	daggerheartStore := &fakeDaggerheartStore{}
	s := Stores{
		Campaign:         &fakeCampaignStore{},
		Character:        &fakeCharacterStore{},
		Domain:           &fakeDomainEngine{},
		Session:          &fakeSessionStore{},
		SessionGate:      &fakeSessionGateStore{},
		SessionSpotlight: &fakeSessionSpotlightStore{},
		Daggerheart:      daggerheartStore,
		Event:            &fakeEventStore{},
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
