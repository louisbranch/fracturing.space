package settings

import (
	"net/url"
	"testing"

	settingsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/app"
)

func TestParseProfileInputTrimsAndPreservesAvatarIDs(t *testing.T) {
	t.Parallel()

	existing := settingsapp.SettingsProfile{
		Username:      "rhea",
		AvatarSetID:   "set-a",
		AvatarAssetID: "asset-1",
	}
	form := url.Values{
		"name":     {"  Rhea Vale  "},
		"pronouns": {"  she/her  "},
		"bio":      {"  Traveler  "},
	}
	profile := parseProfileInput(form, existing)
	if profile.Username != "rhea" {
		t.Fatalf("Username = %q, want %q", profile.Username, "rhea")
	}
	if profile.Name != "Rhea Vale" {
		t.Fatalf("Name = %q, want %q", profile.Name, "Rhea Vale")
	}
	if profile.Pronouns != "she/her" {
		t.Fatalf("Pronouns = %q, want %q", profile.Pronouns, "she/her")
	}
	if profile.Bio != "Traveler" {
		t.Fatalf("Bio = %q, want %q", profile.Bio, "Traveler")
	}
	if profile.AvatarSetID != "set-a" || profile.AvatarAssetID != "asset-1" {
		t.Fatalf("Avatar IDs should be preserved, got set=%q asset=%q", profile.AvatarSetID, profile.AvatarAssetID)
	}
}

func TestParseLocaleInputTrimsWhitespace(t *testing.T) {
	t.Parallel()

	locale := parseLocaleInput(url.Values{"locale": {"  pt-BR  "}})
	if locale != "pt-BR" {
		t.Fatalf("locale = %q, want %q", locale, "pt-BR")
	}
}

func TestParseAIKeyCreateInputTrimsWhitespace(t *testing.T) {
	t.Parallel()

	input := parseAIKeyCreateInput(url.Values{
		"label":    {"  Primary  "},
		"provider": {"  anthropic  "},
		"secret":   {"  sk-test  "},
	})
	if input.Label != "Primary" || input.Provider != "anthropic" || input.Secret != "sk-test" {
		t.Fatalf("input = %+v", input)
	}
}
