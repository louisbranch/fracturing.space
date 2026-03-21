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

func TestBuildDaggerheartAssetMap_PublishesCanonicalCatalogAssetIDs(t *testing.T) {
	store := newFakeContentStore()
	store.heritages["heritage.clank"] = contentstore.DaggerheartHeritage{
		ID:   "heritage.clank",
		Name: "Clank",
		Kind: "ancestry",
	}
	store.adversaries["adversary.acid-burrower"] = contentstore.DaggerheartAdversaryEntry{
		ID:   "adversary.acid-burrower",
		Name: "Acid Burrower",
	}

	assetMap, err := buildDaggerheartAssetMap(context.Background(), store, commonv1.Locale_LOCALE_UNSPECIFIED)
	if err != nil {
		t.Fatalf("buildDaggerheartAssetMap() error = %v", err)
	}

	ancestryRef := findAssetRef(assetMap.GetAssets(), "ancestry", "heritage.clank", pb.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_ANCESTRY_ILLUSTRATION)
	if ancestryRef == nil {
		t.Fatal("asset map missing ancestry illustration ref for heritage.clank")
	}
	if ancestryRef.GetStatus() != pb.DaggerheartAssetStatus_DAGGERHEART_ASSET_STATUS_MAPPED {
		t.Fatalf("ancestry status = %v, want %v", ancestryRef.GetStatus(), pb.DaggerheartAssetStatus_DAGGERHEART_ASSET_STATUS_MAPPED)
	}
	if ancestryRef.GetSetId() != "daggerheart_ancestry_set_v1" {
		t.Fatalf("ancestry set id = %q, want %q", ancestryRef.GetSetId(), "daggerheart_ancestry_set_v1")
	}
	if ancestryRef.GetAssetId() != "heritage.clank" {
		t.Fatalf("ancestry asset id = %q, want %q", ancestryRef.GetAssetId(), "heritage.clank")
	}
	if ancestryRef.GetCdnAssetId() == "" {
		t.Fatal("expected non-empty ancestry cdn asset id")
	}

	adversaryRef := findAssetRef(assetMap.GetAssets(), "adversary", "adversary.acid-burrower", pb.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_ADVERSARY_ILLUSTRATION)
	if adversaryRef == nil {
		t.Fatal("asset map missing adversary illustration ref for adversary.acid-burrower")
	}
	if adversaryRef.GetStatus() != pb.DaggerheartAssetStatus_DAGGERHEART_ASSET_STATUS_MAPPED {
		t.Fatalf("adversary status = %v, want %v", adversaryRef.GetStatus(), pb.DaggerheartAssetStatus_DAGGERHEART_ASSET_STATUS_MAPPED)
	}
	if adversaryRef.GetSetId() != "daggerheart_adversary_set_v1" {
		t.Fatalf("adversary set id = %q, want %q", adversaryRef.GetSetId(), "daggerheart_adversary_set_v1")
	}
	if adversaryRef.GetAssetId() != "adversary.acid-burrower" {
		t.Fatalf("adversary asset id = %q, want %q", adversaryRef.GetAssetId(), "adversary.acid-burrower")
	}
	if adversaryRef.GetCdnAssetId() == "" {
		t.Fatal("expected non-empty adversary cdn asset id")
	}
}

func TestBuildDaggerheartAssetMap_ResolvesCanonicalDomainCardIDs(t *testing.T) {
	store := newFakeContentStore()
	store.domainCards["domain_card.book-of-ava"] = contentstore.DaggerheartDomainCard{
		ID:       "domain_card.book-of-ava",
		Name:     "Book of Ava",
		DomainID: "domain.codex",
	}

	assetMap, err := buildDaggerheartAssetMap(context.Background(), store, commonv1.Locale_LOCALE_UNSPECIFIED)
	if err != nil {
		t.Fatalf("buildDaggerheartAssetMap() error = %v", err)
	}

	cardRef := findAssetRef(assetMap.GetAssets(), "domain_card", "domain_card.book-of-ava", pb.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_DOMAIN_CARD_ILLUSTRATION)
	if cardRef == nil {
		t.Fatal("asset map missing domain-card illustration ref for domain_card.book-of-ava")
	}
	if cardRef.GetStatus() != pb.DaggerheartAssetStatus_DAGGERHEART_ASSET_STATUS_MAPPED {
		t.Fatalf("domain-card status = %v, want %v", cardRef.GetStatus(), pb.DaggerheartAssetStatus_DAGGERHEART_ASSET_STATUS_MAPPED)
	}
	if cardRef.GetSetId() != "daggerheart_domain_card_set_v1" {
		t.Fatalf("domain-card set id = %q, want %q", cardRef.GetSetId(), "daggerheart_domain_card_set_v1")
	}
	if cardRef.GetAssetId() != "domain_card.book-of-ava" {
		t.Fatalf("domain-card asset id = %q, want %q", cardRef.GetAssetId(), "domain_card.book-of-ava")
	}
	if cardRef.GetCdnAssetId() == "" {
		t.Fatal("expected non-empty domain-card cdn asset id")
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
	return findAssetRef(assets, entityType, entityID, assetType) != nil
}

func findAssetRef(assets []*pb.DaggerheartAssetRef, entityType, entityID string, assetType pb.DaggerheartAssetType) *pb.DaggerheartAssetRef {
	for _, asset := range assets {
		if asset.GetEntityType() == entityType && asset.GetEntityId() == entityID && asset.GetType() == assetType {
			return asset
		}
	}
	return nil
}
