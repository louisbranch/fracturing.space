package websupport

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
)

func TestResolveHTTPFallbackPortAndSanitizePort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "empty", raw: "", want: ""},
		{name: "host port", raw: "127.0.0.1:8080", want: "8080"},
		{name: "trimmed host port", raw: " example.com:443 ", want: "443"},
		{name: "bare port", raw: "3000", want: "3000"},
		{name: "hostish fallback", raw: "localhost:9090", want: "9090"},
		{name: "invalid port", raw: "localhost:not-a-port", want: ""},
		{name: "out of range", raw: "70000", want: ""},
		{name: "ipv6 host port", raw: "[::1]:8443", want: "8443"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := ResolveHTTPFallbackPort(tc.raw); got != tc.want {
				t.Fatalf("ResolveHTTPFallbackPort(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}

	sanitizeTests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "trimmed", raw: " 8080 ", want: "8080"},
		{name: "zero", raw: "0", want: ""},
		{name: "negative", raw: "-1", want: ""},
		{name: "too large", raw: "65536", want: ""},
		{name: "alpha", raw: "abc", want: ""},
	}

	for _, tc := range sanitizeTests {
		tc := tc
		t.Run("sanitize_"+tc.name, func(t *testing.T) {
			t.Parallel()
			if got := SanitizePort(tc.raw); got != tc.want {
				t.Fatalf("SanitizePort(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

func TestResolveWebAvatarSelectionFallbacks(t *testing.T) {
	t.Parallel()

	setID, assetID := ResolveWebAvatarSelection(
		catalog.AvatarRoleUser,
		"user-1",
		catalog.AvatarSetPeopleV1,
		"missing-asset",
	)
	if setID != catalog.AvatarSetPeopleV1 {
		t.Fatalf("setID = %q, want %q", setID, catalog.AvatarSetPeopleV1)
	}
	if assetID == "missing-asset" {
		t.Fatalf("assetID = %q, want deterministic fallback asset", assetID)
	}
	if !catalog.AvatarManifest().ValidateAssetInSet(setID, assetID) {
		t.Fatalf("assetID = %q, want asset valid in %q", assetID, setID)
	}

	setID, assetID = ResolveWebAvatarSelection(
		catalog.AvatarRoleUser,
		"user-1",
		"missing-set",
		"missing-asset",
	)
	if setID != catalog.AvatarSetBlankV1 {
		t.Fatalf("setID = %q, want %q", setID, catalog.AvatarSetBlankV1)
	}
	if !catalog.AvatarManifest().ValidateAssetInSet(setID, assetID) {
		t.Fatalf("assetID = %q, want asset valid in %q", assetID, setID)
	}
}

func TestDefaultWebAvatarAssetIDAlwaysReturnsConfiguredBlankAsset(t *testing.T) {
	t.Parallel()

	assetID := defaultWebAvatarAssetID()
	if assetID == "" {
		t.Fatal("defaultWebAvatarAssetID() returned empty asset id")
	}
	if !catalog.AvatarManifest().ValidateAssetInSet(catalog.AvatarSetBlankV1, assetID) {
		t.Fatalf("assetID = %q, want asset valid in %q", assetID, catalog.AvatarSetBlankV1)
	}
}

func TestResolveWebAvatarPortraitFallbacks(t *testing.T) {
	t.Parallel()

	peopleSheet, ok := catalog.AvatarSheetBySetID(catalog.AvatarSetPeopleV1)
	if !ok {
		t.Fatalf("expected sheet for %q", catalog.AvatarSetPeopleV1)
	}
	got := ResolveWebAvatarPortrait("", "user-1", "missing-set")
	if got != peopleSheet.Portraits[1] {
		t.Fatalf("ResolveWebAvatarPortrait(default role, missing set) = %+v, want %+v", got, peopleSheet.Portraits[1])
	}

	got = ResolveWebAvatarPortrait(catalog.AvatarRoleCharacter, "", "missing-set")
	if got != peopleSheet.Portraits[2] {
		t.Fatalf("ResolveWebAvatarPortrait(character missing set fallback) = %+v, want %+v", got, peopleSheet.Portraits[2])
	}

	got = ResolveWebAvatarPortrait("unknown-role", "entity-1", catalog.AvatarSetPeopleV1)
	if got != peopleSheet.Portraits[1] {
		t.Fatalf("ResolveWebAvatarPortrait(unknown role) = %+v, want %+v", got, peopleSheet.Portraits[1])
	}
}
