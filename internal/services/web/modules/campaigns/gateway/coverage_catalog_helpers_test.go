package gateway

import (
	"testing"

	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func TestMapCatalogFeatureHelpersNormalizeAndFilter(t *testing.T) {
	t.Parallel()

	feature := mapCatalogFeature(&daggerheartv1.DaggerheartFeature{
		Name:        "  Battle Rhythm  ",
		Description: "  Gain momentum after a hit.  ",
	})
	if feature.Name != "Battle Rhythm" || feature.Description != "Gain momentum after a hit." {
		t.Fatalf("mapCatalogFeature() = %#v", feature)
	}

	mapped := mapCatalogFeatures([]*daggerheartv1.DaggerheartFeature{
		nil,
		{Name: "  ", Description: "ignored"},
		{Name: "  Keen Edge ", Description: "  Boosts blade damage. "},
	})
	if len(mapped) != 1 {
		t.Fatalf("len(mapped) = %d, want 1", len(mapped))
	}
	if mapped[0].Name != "Keen Edge" || mapped[0].Description != "Boosts blade damage." {
		t.Fatalf("mapped features = %#v", mapped)
	}
}

func TestDaggerheartAssetLookupGetUsesCanonicalKeys(t *testing.T) {
	t.Parallel()

	lookup := daggerheartAssetLookupFromResponse(&daggerheartv1.GetDaggerheartAssetMapResponse{
		AssetMap: &daggerheartv1.DaggerheartAssetMap{
			Assets: []*daggerheartv1.DaggerheartAssetRef{{
				EntityId:   " class-1 ",
				EntityType: "Class",
				Type:       daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_CLASS_ILLUSTRATION,
				CdnAssetId: "cdn-1",
			}},
		},
	})

	got := lookup.get("class-1", "class", daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_CLASS_ILLUSTRATION)
	if got == nil || got.GetCdnAssetId() != "cdn-1" {
		t.Fatalf("lookup.get() = %#v, want mapped asset", got)
	}
	if missing := (daggerheartAssetLookup(nil)).get("class-1", "class", daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_CLASS_ILLUSTRATION); missing != nil {
		t.Fatalf("nil lookup get() = %#v, want nil", missing)
	}
}
