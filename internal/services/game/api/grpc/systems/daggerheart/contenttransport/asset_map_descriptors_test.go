package contenttransport

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
)

func TestCollectDaggerheartAssetDescriptors(t *testing.T) {
	descriptors := collectDaggerheartAssetDescriptors(
		[]contentstore.DaggerheartClass{
			{ID: " class-2 "},
			{ID: "class-2"},
			{ID: ""},
		},
		[]contentstore.DaggerheartSubclass{
			{ID: "sub-1"},
		},
		[]contentstore.DaggerheartHeritage{
			{ID: "heritage-1", Kind: "ancestry"},
			{ID: "heritage-2", Kind: "community"},
			{ID: "heritage-3", Kind: "unknown"},
		},
		[]contentstore.DaggerheartDomain{
			{ID: "domain-1"},
		},
		[]contentstore.DaggerheartDomainCard{
			{ID: "card-1"},
		},
		[]contentstore.DaggerheartAdversaryEntry{
			{ID: "adversary-1"},
		},
		[]contentstore.DaggerheartEnvironment{
			{ID: "environment-1"},
		},
		[]contentstore.DaggerheartWeapon{
			{ID: "weapon-1"},
		},
		[]contentstore.DaggerheartArmor{
			{ID: "armor-1"},
		},
		[]contentstore.DaggerheartItem{
			{ID: "item-1"},
		},
	)

	got := make([]string, 0, len(descriptors))
	for _, descriptor := range descriptors {
		got = append(got, descriptor.EntityType+"|"+descriptor.EntityID+"|"+descriptor.AssetType)
	}

	want := []string{
		catalog.DaggerheartEntityTypeAdversary + "|adversary-1|" + catalog.DaggerheartAssetTypeAdversaryIllustration,
		catalog.DaggerheartEntityTypeAncestry + "|heritage-1|" + catalog.DaggerheartAssetTypeAncestryIllustration,
		catalog.DaggerheartEntityTypeArmor + "|armor-1|" + catalog.DaggerheartAssetTypeArmorIllustration,
		catalog.DaggerheartEntityTypeClass + "|class-2|" + catalog.DaggerheartAssetTypeClassIcon,
		catalog.DaggerheartEntityTypeClass + "|class-2|" + catalog.DaggerheartAssetTypeClassIllustration,
		catalog.DaggerheartEntityTypeCommunity + "|heritage-2|" + catalog.DaggerheartAssetTypeCommunityIllustration,
		catalog.DaggerheartEntityTypeDomain + "|domain-1|" + catalog.DaggerheartAssetTypeDomainIcon,
		catalog.DaggerheartEntityTypeDomain + "|domain-1|" + catalog.DaggerheartAssetTypeDomainIllustration,
		catalog.DaggerheartEntityTypeDomainCard + "|card-1|" + catalog.DaggerheartAssetTypeDomainCardIllustration,
		catalog.DaggerheartEntityTypeEnvironment + "|environment-1|" + catalog.DaggerheartAssetTypeEnvironmentIllustration,
		catalog.DaggerheartEntityTypeItem + "|item-1|" + catalog.DaggerheartAssetTypeItemIllustration,
		catalog.DaggerheartEntityTypeSubclass + "|sub-1|" + catalog.DaggerheartAssetTypeSubclassIllustration,
		catalog.DaggerheartEntityTypeWeapon + "|weapon-1|" + catalog.DaggerheartAssetTypeWeaponIllustration,
	}

	if len(got) != len(want) {
		t.Fatalf("len(descriptors) = %d, want %d\n got: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("descriptors[%d] = %q, want %q\nfull: %v", i, got[i], want[i], got)
		}
	}
}

func TestAppendAssetDescriptorIgnoresBlankAndDuplicateDescriptors(t *testing.T) {
	descriptors := make([]daggerheartAssetDescriptor, 0, 1)
	seen := map[string]struct{}{}

	appendAssetDescriptor(&descriptors, seen, " class ", "class-1", catalog.DaggerheartAssetTypeClassIcon)
	appendAssetDescriptor(&descriptors, seen, "class", "class-1", catalog.DaggerheartAssetTypeClassIcon)
	appendAssetDescriptor(&descriptors, seen, "class", "", catalog.DaggerheartAssetTypeClassIllustration)

	if len(descriptors) != 1 {
		t.Fatalf("len(descriptors) = %d, want 1", len(descriptors))
	}
	if descriptors[0].EntityType != "class" || descriptors[0].EntityID != "class-1" || descriptors[0].AssetType != catalog.DaggerheartAssetTypeClassIcon {
		t.Fatalf("descriptor = %+v, want normalized class icon ref", descriptors[0])
	}
}
