package contenttransport

import (
	"context"
	"errors"
	"strings"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
)

type failingAssetMapStore struct {
	*fakeContentStore
	listItemsErr error
}

func (s *failingAssetMapStore) ListDaggerheartItems(ctx context.Context) ([]contentstore.DaggerheartItem, error) {
	if s.listItemsErr != nil {
		return nil, s.listItemsErr
	}
	return s.fakeContentStore.ListDaggerheartItems(ctx)
}

func TestBuildDaggerheartAssetMap(t *testing.T) {
	store := newFakeContentStore()

	assetMap, err := buildDaggerheartAssetMap(context.Background(), store, commonv1.Locale_LOCALE_PT_BR)
	if err != nil {
		t.Fatalf("buildDaggerheartAssetMap() error = %v", err)
	}

	descriptors := collectDaggerheartAssetDescriptors(
		mustList(t, store.ListDaggerheartClasses),
		mustList(t, store.ListDaggerheartSubclasses),
		mustList(t, store.ListDaggerheartHeritages),
		mustList(t, store.ListDaggerheartDomains),
		mustList(t, store.ListDaggerheartDomainCards),
		mustList(t, store.ListDaggerheartAdversaryEntries),
		mustList(t, store.ListDaggerheartEnvironments),
		mustList(t, store.ListDaggerheartWeapons),
		mustList(t, store.ListDaggerheartArmor),
		mustList(t, store.ListDaggerheartItems),
	)

	if assetMap.GetId() == "" {
		t.Fatal("assetMap.Id = empty, want manifest or fallback id")
	}
	if assetMap.GetSystemId() == "" || assetMap.GetSystemVersion() == "" {
		t.Fatalf("assetMap system metadata = (%q, %q), want non-empty values", assetMap.GetSystemId(), assetMap.GetSystemVersion())
	}
	if assetMap.GetLocale() != commonv1.Locale_LOCALE_PT_BR {
		t.Fatalf("assetMap.Locale = %v, want %v", assetMap.GetLocale(), commonv1.Locale_LOCALE_PT_BR)
	}
	if len(assetMap.GetAssets()) != len(descriptors) {
		t.Fatalf("len(assetMap.Assets) = %d, want %d", len(assetMap.GetAssets()), len(descriptors))
	}
	if !hasAssetRef(assetMap.GetAssets(), "class", "class-1", pb.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_CLASS_ICON) {
		t.Fatal("asset map missing class icon ref for class-1")
	}
}

func TestBuildDaggerheartAssetMapWrapsLoadFailure(t *testing.T) {
	store := &failingAssetMapStore{
		fakeContentStore: newFakeContentStore(),
		listItemsErr:     errors.New("boom"),
	}

	_, err := buildDaggerheartAssetMap(context.Background(), store, commonv1.Locale_LOCALE_UNSPECIFIED)
	if err == nil {
		t.Fatal("buildDaggerheartAssetMap() error = nil, want failure")
	}
	if !strings.Contains(err.Error(), "list items") {
		t.Fatalf("buildDaggerheartAssetMap() error = %q, want step name %q", err, "list items")
	}
}

func TestBuildDaggerheartAssetMapDefaultsLocale(t *testing.T) {
	assetMap, err := buildDaggerheartAssetMap(context.Background(), newFakeContentStore(), commonv1.Locale_LOCALE_UNSPECIFIED)
	if err != nil {
		t.Fatalf("buildDaggerheartAssetMap() error = %v", err)
	}
	if assetMap.GetLocale() != defaultDaggerheartAssetMapLocale {
		t.Fatalf("assetMap.Locale = %v, want %v", assetMap.GetLocale(), defaultDaggerheartAssetMapLocale)
	}
}

func mustList[T any](t *testing.T, list func(context.Context) ([]T, error)) []T {
	t.Helper()
	items, err := list(context.Background())
	if err != nil {
		t.Fatalf("list() error = %v", err)
	}
	return items
}

func hasAssetRef(assets []*pb.DaggerheartAssetRef, entityType, entityID string, assetType pb.DaggerheartAssetType) bool {
	for _, asset := range assets {
		if asset.GetEntityType() == entityType && asset.GetEntityId() == entityID && asset.GetType() == assetType {
			return true
		}
	}
	return false
}
