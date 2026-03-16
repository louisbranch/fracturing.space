package gameplaystores

import (
	"context"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/manifest"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/test/mock/gamefakes"
)

func TestStoresValidate_MissingEvents(t *testing.T) {
	s := Stores{
		Campaign:         gamefakes.NewCampaignStore(),
		Character:        gamefakes.NewCharacterStore(),
		Session:          gamefakes.NewSessionStore(),
		SessionGate:      &fakeSessionGateStore{},
		SessionSpotlight: &fakeSessionSpotlightStore{},
		Daggerheart:      gamefakes.NewDaggerheartStore(),
		Content:          &fakeContentStore{},
		Event:            gamefakes.NewEventStore(),
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
		Campaign:         gamefakes.NewCampaignStore(),
		Character:        gamefakes.NewCharacterStore(),
		Session:          gamefakes.NewSessionStore(),
		SessionGate:      &fakeSessionGateStore{},
		SessionSpotlight: &fakeSessionSpotlightStore{},
		Daggerheart:      gamefakes.NewDaggerheartStore(),
		Content:          &fakeContentStore{},
		Event:            gamefakes.NewEventStore(),
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

func TestNewFromProjection(t *testing.T) {
	projectionStore := &projectionStoreBundleStub{
		CampaignStore:            gamefakes.NewCampaignStore(),
		CharacterStore:           gamefakes.NewCharacterStore(),
		SessionStore:             gamefakes.NewSessionStore(),
		SessionGateStore:         &fakeSessionGateStore{},
		SessionSpotlightStore:    &fakeSessionSpotlightStore{},
		ProjectionWatermarkStore: stubProjectionWatermarkStore{},
	}
	daggerheartStore := gamefakes.NewDaggerheartStore()

	stores := NewFromProjection(FromProjectionConfig{
		ProjectionStore:  projectionStore,
		DaggerheartStore: daggerheartStore,
		ContentStore:     &fakeContentStore{},
		EventStore:       gamefakes.NewEventStore(),
		Domain:           &fakeDomainEngine{},
		WriteRuntime:     domainwrite.NewRuntime(),
		Events:           event.NewRegistry(),
	})

	if stores.Campaign == nil || stores.Character == nil || stores.Daggerheart == nil {
		t.Fatal("expected projection-backed stores to be populated")
	}
	if stores.Daggerheart != daggerheartStore {
		t.Fatal("expected Daggerheart store to come from explicit system stores")
	}
	if stores.Event == nil || stores.Write.Runtime == nil || stores.Events == nil {
		t.Fatal("expected runtime stores to be propagated")
	}
}

// TestStoresAdapterRegistryMatchesManifest verifies that the adapter registry
// built by TryApplier registers the same adapters as the canonical
// manifest.AdapterRegistry, preventing duplication drift.
func TestStoresAdapterRegistryMatchesManifest(t *testing.T) {
	daggerheartStore := gamefakes.NewDaggerheartStore()
	s := Stores{
		Campaign:         gamefakes.NewCampaignStore(),
		Character:        gamefakes.NewCharacterStore(),
		Session:          gamefakes.NewSessionStore(),
		SessionGate:      &fakeSessionGateStore{},
		SessionSpotlight: &fakeSessionSpotlightStore{},
		Daggerheart:      daggerheartStore,
		Content:          &fakeContentStore{},
		Event:            gamefakes.NewEventStore(),
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

	manifestRegistry, err := systemmanifest.AdapterRegistry(daggerheartStore)
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
	storage.ProjectionWatermarkStore
}

type stubProjectionWatermarkStore struct {
	storage.ProjectionWatermarkStore
}

type fakeDomainEngine struct{}

type fakeContentStore struct {
	contentstore.DaggerheartContentReadStore
}

func (f *fakeDomainEngine) Execute(context.Context, command.Command) (engine.Result, error) {
	return engine.Result{}, nil
}

type fakeSessionGateStore struct{}

func (s *fakeSessionGateStore) PutSessionGate(context.Context, storage.SessionGate) error {
	return nil
}

func (s *fakeSessionGateStore) GetSessionGate(context.Context, string, string, string) (storage.SessionGate, error) {
	return storage.SessionGate{}, storage.ErrNotFound
}

func (s *fakeSessionGateStore) GetOpenSessionGate(context.Context, string, string) (storage.SessionGate, error) {
	return storage.SessionGate{}, storage.ErrNotFound
}

type fakeSessionSpotlightStore struct{}

func (s *fakeSessionSpotlightStore) PutSessionSpotlight(context.Context, storage.SessionSpotlight) error {
	return nil
}

func (s *fakeSessionSpotlightStore) GetSessionSpotlight(context.Context, string, string) (storage.SessionSpotlight, error) {
	return storage.SessionSpotlight{}, storage.ErrNotFound
}

func (s *fakeSessionSpotlightStore) ClearSessionSpotlight(context.Context, string, string) error {
	return nil
}
