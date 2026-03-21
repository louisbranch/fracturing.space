package catalog

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestDaggerheartAssetsManifest_ResolvesSecondaryNoneWeaponCard(t *testing.T) {
	manifest := DaggerheartAssetsManifest()

	resolved := manifest.ResolveEntityAsset(
		DaggerheartEntityTypeWeapon,
		"weapon.no-secondary",
		DaggerheartAssetTypeWeaponIllustration,
	)

	if resolved.Status != DaggerheartAssetResolutionStatusMapped {
		t.Fatalf("status = %q, want %q", resolved.Status, DaggerheartAssetResolutionStatusMapped)
	}
	if resolved.SetID != "daggerheart_weapon_set_v1" {
		t.Fatalf("set id = %q, want %q", resolved.SetID, "daggerheart_weapon_set_v1")
	}
	if resolved.AssetID != "weapon.no-secondary" {
		t.Fatalf("asset id = %q, want %q", resolved.AssetID, "weapon.no-secondary")
	}
	if resolved.CDNAssetID == "" {
		t.Fatal("expected non-empty cdn asset id")
	}
}

func TestDaggerheartAssetsManifest_ResolvesCanonicalCatalogIDs(t *testing.T) {
	manifest := DaggerheartAssetsManifest()

	tests := []struct {
		name        string
		entityType  string
		entityID    string
		assetType   string
		wantSetID   string
		wantAssetID string
	}{
		{
			name:        "ancestry",
			entityType:  DaggerheartEntityTypeAncestry,
			entityID:    "heritage.clank",
			assetType:   DaggerheartAssetTypeAncestryIllustration,
			wantSetID:   "daggerheart_ancestry_set_v1",
			wantAssetID: "heritage.clank",
		},
		{
			name:        "adversary",
			entityType:  DaggerheartEntityTypeAdversary,
			entityID:    "adversary.acid-burrower",
			assetType:   DaggerheartAssetTypeAdversaryIllustration,
			wantSetID:   "daggerheart_adversary_set_v1",
			wantAssetID: "adversary.acid-burrower",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resolved := manifest.ResolveEntityAsset(tc.entityType, tc.entityID, tc.assetType)

			if resolved.Status != DaggerheartAssetResolutionStatusMapped {
				t.Fatalf("status = %q, want %q", resolved.Status, DaggerheartAssetResolutionStatusMapped)
			}
			if resolved.SetID != tc.wantSetID {
				t.Fatalf("set id = %q, want %q", resolved.SetID, tc.wantSetID)
			}
			if resolved.AssetID != tc.wantAssetID {
				t.Fatalf("asset id = %q, want %q", resolved.AssetID, tc.wantAssetID)
			}
			if resolved.CDNAssetID == "" {
				t.Fatal("expected non-empty cdn asset id")
			}
		})
	}
}

func TestDaggerheartAssetsManifest_ResolvesCanonicalDomainCardIDs(t *testing.T) {
	manifest := DaggerheartAssetsManifest()

	tests := []struct {
		name        string
		entityID    string
		wantAssetID string
	}{
		{
			name:        "book of ava",
			entityID:    "domain_card.book-of-ava",
			wantAssetID: "domain_card.book-of-ava",
		},
		{
			name:        "rune ward",
			entityID:    "domain_card.rune-ward",
			wantAssetID: "domain_card.rune-ward",
		},
		{
			name:        "rain of blades",
			entityID:    "domain_card.rain-of-blades",
			wantAssetID: "domain_card.rain-of-blades",
		},
		{
			name:        "natures tongue",
			entityID:    "domain_card.nature-s-tongue",
			wantAssetID: "domain_card.nature-s-tongue",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resolved := manifest.ResolveEntityAsset(
				DaggerheartEntityTypeDomainCard,
				tc.entityID,
				DaggerheartAssetTypeDomainCardIllustration,
			)

			if resolved.Status != DaggerheartAssetResolutionStatusMapped {
				t.Fatalf("status = %q, want %q", resolved.Status, DaggerheartAssetResolutionStatusMapped)
			}
			if resolved.SetID != "daggerheart_domain_card_set_v1" {
				t.Fatalf("set id = %q, want %q", resolved.SetID, "daggerheart_domain_card_set_v1")
			}
			if resolved.AssetID != tc.wantAssetID {
				t.Fatalf("asset id = %q, want %q", resolved.AssetID, tc.wantAssetID)
			}
			if resolved.CDNAssetID == "" {
				t.Fatal("expected non-empty cdn asset id")
			}
		})
	}
}

func TestEmbeddedDaggerheartAssets_DoNotRequireAliasing(t *testing.T) {
	var cloudinaryCatalog map[string]json.RawMessage
	if err := json.Unmarshal(cloudinaryAssetsCatalogJSON, &cloudinaryCatalog); err != nil {
		t.Fatalf("decode embedded cloudinary catalog: %v", err)
	}

	for key, raw := range cloudinaryCatalog {
		if !strings.HasPrefix(key, "daggerheart_") {
			continue
		}
		var entries []struct {
			FSAssetID  string `json:"fs_asset_id"`
			CatalogRef struct {
				EntityID string `json:"entity_id"`
			} `json:"catalog_ref"`
		}
		if err := json.Unmarshal(raw, &entries); err != nil {
			t.Fatalf("decode %s entries: %v", key, err)
		}
		for _, entry := range entries {
			entityID := strings.TrimSpace(entry.CatalogRef.EntityID)
			if entityID == "" {
				continue
			}
			if got := strings.TrimSpace(entry.FSAssetID); got != entityID {
				t.Fatalf("%s fs_asset_id = %q, want catalog entity id %q", key, got, entityID)
			}
		}
	}

	var suggestions struct {
		DaggerheartV1 daggerheartAssetManifestJSONDocument `json:"daggerheart_v1"`
	}
	if err := json.Unmarshal(cloudinaryCatalog["fs_catalog_suggestions"], &suggestions); err != nil {
		t.Fatalf("decode daggerheart suggestions: %v", err)
	}

	var embeddedManifest daggerheartAssetManifestJSONDocument
	if err := json.Unmarshal(daggerheartAssetManifestJSON, &embeddedManifest); err != nil {
		t.Fatalf("decode embedded daggerheart manifest: %v", err)
	}

	if embeddedManifest.ID != suggestions.DaggerheartV1.ID ||
		embeddedManifest.SystemID != suggestions.DaggerheartV1.SystemID ||
		embeddedManifest.SystemVersion != suggestions.DaggerheartV1.SystemVersion ||
		embeddedManifest.Locale != suggestions.DaggerheartV1.Locale ||
		embeddedManifest.Theme != suggestions.DaggerheartV1.Theme {
		t.Fatalf("embedded manifest metadata does not match cloudinary suggestions")
	}

	if !reflect.DeepEqual(daggerheartSetAssetIDsByKey(embeddedManifest), daggerheartSetAssetIDsByKey(suggestions.DaggerheartV1)) {
		t.Fatal("embedded manifest sets do not match cloudinary daggerheart suggestions")
	}
	if !reflect.DeepEqual(daggerheartEntityAssetIDsByKey(embeddedManifest), daggerheartEntityAssetIDsByKey(suggestions.DaggerheartV1)) {
		t.Fatal("embedded manifest entity asset map does not match cloudinary daggerheart suggestions")
	}
	for _, entry := range embeddedManifest.EntityAssetMap {
		if strings.TrimSpace(entry.AssetID) != strings.TrimSpace(entry.EntityID) {
			t.Fatalf("entity asset map asset_id = %q, want canonical entity id %q", entry.AssetID, entry.EntityID)
		}
	}
}

func TestResolveEntityAsset_UsesMappedAssetWhenDeliverable(t *testing.T) {
	manifest := mustDecodeDaggerheartAssetManifest(t, `{
		"id": "daggerheart-assets-v1",
		"system_id": "daggerheart",
		"system_version": "v1",
		"sets": [
			{
				"id": "daggerheart_class_icon_set_v1",
				"asset_type": "daggerheart_class_icon",
				"asset_ids": ["class.guardian"]
			}
		],
		"entity_asset_map": [
			{
				"entity_type": "class",
				"entity_id": "class.guardian",
				"asset_type": "daggerheart_class_icon",
				"set_id": "daggerheart_class_icon_set_v1",
				"asset_id": "class.guardian"
			}
		]
	}`)

	resolved := manifest.resolveEntityAsset(
		DaggerheartEntityTypeClass,
		"class.guardian",
		DaggerheartAssetTypeClassIcon,
		testCloudinaryLookup(map[string]string{
			"daggerheart_class_icon_set_v1\x00class.guardian": "v123/high_fantasy/daggerheart_class_icon/v1/class.guardian",
		}),
	)

	if resolved.Status != DaggerheartAssetResolutionStatusMapped {
		t.Fatalf("status = %q, want %q", resolved.Status, DaggerheartAssetResolutionStatusMapped)
	}
	if resolved.SetID != "daggerheart_class_icon_set_v1" {
		t.Fatalf("set id = %q, want %q", resolved.SetID, "daggerheart_class_icon_set_v1")
	}
	if resolved.AssetID != "class.guardian" {
		t.Fatalf("asset id = %q, want %q", resolved.AssetID, "class.guardian")
	}
	if resolved.CDNAssetID != "v123/high_fantasy/daggerheart_class_icon/v1/class.guardian" {
		t.Fatalf("cdn asset id = %q, want %q", resolved.CDNAssetID, "v123/high_fantasy/daggerheart_class_icon/v1/class.guardian")
	}
}

func TestResolveEntityAsset_UsesSetDefaultWhenMappingMissing(t *testing.T) {
	manifest := mustDecodeDaggerheartAssetManifest(t, `{
		"id": "daggerheart-assets-v1",
		"system_id": "daggerheart",
		"system_version": "v1",
		"sets": [
			{
				"id": "daggerheart_subclass_set_v1",
				"asset_type": "daggerheart_subclass_illustration",
				"asset_ids": ["subclass.guardian"]
			}
		],
		"entity_asset_map": []
	}`)

	resolved := manifest.resolveEntityAsset(
		DaggerheartEntityTypeSubclass,
		"subclass.guardian",
		DaggerheartAssetTypeSubclassIllustration,
		testCloudinaryLookup(map[string]string{
			"daggerheart_subclass_set_v1\x00subclass.guardian": "v456/high_fantasy/daggerheart_subclass_illustration/v1/subclass.guardian",
		}),
	)

	if resolved.Status != DaggerheartAssetResolutionStatusSetDefault {
		t.Fatalf("status = %q, want %q", resolved.Status, DaggerheartAssetResolutionStatusSetDefault)
	}
	if resolved.AssetID != "subclass.guardian" {
		t.Fatalf("asset id = %q, want %q", resolved.AssetID, "subclass.guardian")
	}
	if resolved.CDNAssetID == "" {
		t.Fatal("expected non-empty cdn asset id")
	}
}

func TestResolveEntityAsset_FallsBackToSetDefaultWhenMappedAssetUnavailable(t *testing.T) {
	manifest := mustDecodeDaggerheartAssetManifest(t, `{
		"id": "daggerheart-assets-v1",
		"system_id": "daggerheart",
		"system_version": "v1",
		"sets": [
			{
				"id": "daggerheart_domain_icon_set_v1",
				"asset_type": "daggerheart_domain_icon",
				"asset_ids": ["domain.arcana", "domain.blade"]
			}
		],
		"entity_asset_map": [
			{
				"entity_type": "domain",
				"entity_id": "domain.arcana",
				"asset_type": "daggerheart_domain_icon",
				"set_id": "daggerheart_domain_icon_set_v1",
				"asset_id": "domain.arcana"
			}
		]
	}`)

	resolved := manifest.resolveEntityAsset(
		DaggerheartEntityTypeDomain,
		"domain.arcana",
		DaggerheartAssetTypeDomainIcon,
		testCloudinaryLookup(map[string]string{
			"daggerheart_domain_icon_set_v1\x00domain.blade": "v789/high_fantasy/daggerheart_domain_icon/v1/domain.blade",
		}),
	)

	if resolved.Status != DaggerheartAssetResolutionStatusSetDefault {
		t.Fatalf("status = %q, want %q", resolved.Status, DaggerheartAssetResolutionStatusSetDefault)
	}
	if resolved.AssetID != "domain.blade" {
		t.Fatalf("asset id = %q, want %q", resolved.AssetID, "domain.blade")
	}
}

func TestResolveEntityAsset_UnavailableWithoutDeliverableAssets(t *testing.T) {
	manifest := mustDecodeDaggerheartAssetManifest(t, `{
		"id": "daggerheart-assets-v1",
		"system_id": "daggerheart",
		"system_version": "v1",
		"sets": [
			{
				"id": "daggerheart_environment_set_v1",
				"asset_type": "daggerheart_environment_illustration",
				"asset_ids": ["environment.void"]
			}
		],
		"entity_asset_map": []
	}`)

	resolved := manifest.resolveEntityAsset(
		DaggerheartEntityTypeEnvironment,
		"environment.void",
		DaggerheartAssetTypeEnvironmentIllustration,
		testCloudinaryLookup(map[string]string{}),
	)

	if resolved.Status != DaggerheartAssetResolutionStatusUnavailable {
		t.Fatalf("status = %q, want %q", resolved.Status, DaggerheartAssetResolutionStatusUnavailable)
	}
	if resolved.CDNAssetID != "" {
		t.Fatalf("cdn asset id = %q, want empty", resolved.CDNAssetID)
	}
}

func mustDecodeDaggerheartAssetManifest(t *testing.T, raw string) DaggerheartAssetManifest {
	t.Helper()
	manifest, err := decodeDaggerheartAssetManifest([]byte(raw))
	if err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	return manifest
}

func testCloudinaryLookup(values map[string]string) cloudinaryAssetLookupFn {
	return func(setID, assetID string) (string, bool) {
		value, ok := values[setID+"\x00"+assetID]
		if !ok {
			return "", false
		}
		return value, true
	}
}

func daggerheartSetAssetIDsByKey(manifest daggerheartAssetManifestJSONDocument) map[string][]string {
	sets := make(map[string][]string, len(manifest.Sets))
	for _, set := range manifest.Sets {
		key := strings.TrimSpace(set.ID) + "\x00" + strings.ToLower(strings.TrimSpace(set.AssetType))
		sets[key] = append([]string(nil), set.AssetIDs...)
	}
	return sets
}

func daggerheartEntityAssetIDsByKey(manifest daggerheartAssetManifestJSONDocument) map[string]string {
	entityAssets := make(map[string]string, len(manifest.EntityAssetMap))
	for _, entry := range manifest.EntityAssetMap {
		key := strings.ToLower(strings.TrimSpace(entry.EntityType)) +
			"\x00" +
			strings.ToLower(strings.TrimSpace(entry.AssetType)) +
			"\x00" +
			strings.TrimSpace(entry.EntityID)
		entityAssets[key] = strings.TrimSpace(entry.AssetID)
	}
	return entityAssets
}
