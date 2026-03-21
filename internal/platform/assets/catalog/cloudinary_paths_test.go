package catalog

import (
	"encoding/json"
	"regexp"
	"testing"
)

func TestResolveCDNAssetID_UsesVersionedCloudinaryCampaignPath(t *testing.T) {
	got := ResolveCDNAssetID(CampaignCoverSetV1, "ashen_city_gate")
	pattern := regexp.MustCompile(`^v[0-9]+/high_fantasy/campaign_scene/v1/ashen_city_gate$`)
	if !pattern.MatchString(got) {
		t.Fatalf("ResolveCDNAssetID(...) = %q, want match %q", got, pattern.String())
	}
}

func TestResolveCDNAssetID_UsesVersionedCloudinaryAvatarPath(t *testing.T) {
	got := ResolveCDNAssetID(AvatarSetPeopleV1, "apothecary_journeyman")
	pattern := regexp.MustCompile(`^v[0-9]+/high_fantasy/avatar_set/v1/apothecary_journeyman$`)
	if !pattern.MatchString(got) {
		t.Fatalf("ResolveCDNAssetID(...) = %q, want match %q", got, pattern.String())
	}
}

func TestResolveCDNAssetID_FallsBackToCanonicalAssetID(t *testing.T) {
	got := ResolveCDNAssetID("unknown_set", "unknown_asset")
	want := "unknown_asset"
	if got != want {
		t.Fatalf("ResolveCDNAssetID(...) = %q, want %q", got, want)
	}
}

func TestCloudinaryPublicID_RejectsMissingSelectors(t *testing.T) {
	if _, ok := CloudinaryPublicID("", "ashen_city_gate"); ok {
		t.Fatal("expected missing set id to fail")
	}
	if _, ok := CloudinaryPublicID(CampaignCoverSetV1, ""); ok {
		t.Fatal("expected missing asset id to fail")
	}
}

func TestCloudinaryPublicID_ResolvesDaggerheartCatalogIDs(t *testing.T) {
	tests := []struct {
		name    string
		setID   string
		assetID string
		pattern string
	}{
		{
			name:    "domain card",
			setID:   "daggerheart_domain_card_set_v1",
			assetID: "domain_card.book-of-ava",
			pattern: `^v[0-9]+/high_fantasy/daggerheart_domain_card_illustration/v1/codex_book_of_ava$`,
		},
		{
			name:    "weapon none",
			setID:   "daggerheart_weapon_set_v1",
			assetID: "weapon.no-secondary",
			pattern: `^v[0-9]+/high_fantasy/daggerheart_weapon_illustration/v1/no_secondary_weapon$`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			publicID, ok := CloudinaryPublicID(tc.setID, tc.assetID)
			if !ok {
				t.Fatalf("CloudinaryPublicID(%q, %q) missing", tc.setID, tc.assetID)
			}
			matcher := regexp.MustCompile(tc.pattern)
			if !matcher.MatchString(publicID) {
				t.Fatalf("CloudinaryPublicID(%q, %q) = %q, want match %q", tc.setID, tc.assetID, publicID, tc.pattern)
			}
		})
	}
}

func TestDecodeCloudinaryAssetPaths_ParsesAnyAssetArrayKey(t *testing.T) {
	raw := []byte(`{
		"schema_version": 1,
		"daggerheart_class_icon": [
			{
				"set_id": "daggerheart_class_icon_set_v1",
				"fs_asset_id": "class.guardian",
				"cloudinary": {
					"public_id": "high_fantasy/daggerheart_class_icon/v1/class.guardian",
					"version": 1234
				}
			}
		],
		"batch_ids": ["hf_v1_r1"]
	}`)

	decoded := decodeCloudinaryAssetPaths(raw)
	key := cloudinaryAssetPathLookupKey("daggerheart_class_icon_set_v1", "class.guardian")
	if got := decoded[key]; got != "v1234/high_fantasy/daggerheart_class_icon/v1/class.guardian" {
		t.Fatalf("decoded cloudinary path = %q, want %q", got, "v1234/high_fantasy/daggerheart_class_icon/v1/class.guardian")
	}
}

func TestLooksLikeJSONArrayOfObjects(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want bool
	}{
		{name: "object array", raw: `[{"x":1}]`, want: true},
		{name: "empty array", raw: `[]`, want: true},
		{name: "string array", raw: `["a","b"]`, want: false},
		{name: "object", raw: `{"x":1}`, want: false},
		{name: "number", raw: `1`, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := looksLikeJSONArrayOfObjects(json.RawMessage(tc.raw)); got != tc.want {
				t.Fatalf("looksLikeJSONArrayOfObjects(%s) = %t, want %t", tc.raw, got, tc.want)
			}
		})
	}
}
