package characters

import (
	"net/url"
	"testing"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
)

func TestParseCreateCharacterInputDefaultsAndValidation(t *testing.T) {
	t.Parallel()

	input, err := parseCreateCharacterInput(url.Values{"name": {"  Aria  "}, "pronouns": {"  she/her  "}})
	if err != nil {
		t.Fatalf("parseCreateCharacterInput() error = %v", err)
	}
	if input.Name != "Aria" {
		t.Fatalf("Name = %q, want %q", input.Name, "Aria")
	}
	if input.Kind != campaignapp.CharacterKindPC {
		t.Fatalf("Kind = %q, want %q", input.Kind, campaignapp.CharacterKindPC)
	}
	if input.Pronouns != "she/her" {
		t.Fatalf("Pronouns = %q, want %q", input.Pronouns, "she/her")
	}

	input, err = parseCreateCharacterInput(url.Values{"kind": {" npc "}})
	if err != nil {
		t.Fatalf("parseCreateCharacterInput() npc error = %v", err)
	}
	if input.Kind != campaignapp.CharacterKindNPC {
		t.Fatalf("Kind = %q, want %q", input.Kind, campaignapp.CharacterKindNPC)
	}

	if _, err := parseCreateCharacterInput(url.Values{"kind": {"invalid"}}); err == nil {
		t.Fatalf("expected invalid character kind error")
	}
}

func TestParseUpdateInputsTrimWhitespace(t *testing.T) {
	t.Parallel()

	character := parseUpdateCharacterInput(url.Values{
		"name":     {"  Aria  "},
		"pronouns": {"  she/her  "},
	})
	if character.Name != "Aria" || character.Pronouns != "she/her" {
		t.Fatalf("character input = %#v", character)
	}
}

func TestParseAppCharacterKind(t *testing.T) {
	t.Parallel()

	if kind, ok := parseAppCharacterKind("pc"); !ok || kind != campaignapp.CharacterKindPC {
		t.Fatalf("parseAppCharacterKind pc = (%v, %v)", kind, ok)
	}
	if kind, ok := parseAppCharacterKind("npc"); !ok || kind != campaignapp.CharacterKindNPC {
		t.Fatalf("parseAppCharacterKind npc = (%v, %v)", kind, ok)
	}
	if _, ok := parseAppCharacterKind("invalid"); ok {
		t.Fatalf("expected invalid character kind to fail parse")
	}
}
