package campaigns

import (
	"net/url"
	"testing"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
)

func TestParseCreateCampaignInputDefaultsAndValidation(t *testing.T) {
	t.Parallel()

	input, err := parseCreateCampaignInput(url.Values{
		"name":         {"  Voyage  "},
		"theme_prompt": {"  stormy sea  "},
	})
	if err != nil {
		t.Fatalf("parseCreateCampaignInput() error = %v", err)
	}
	if input.Name != "Voyage" {
		t.Fatalf("Name = %q, want %q", input.Name, "Voyage")
	}
	if input.System != campaignapp.GameSystemDaggerheart {
		t.Fatalf("System = %q, want %q", input.System, campaignapp.GameSystemDaggerheart)
	}
	if input.GMMode != campaignapp.GmModeAI {
		t.Fatalf("GMMode = %q, want %q", input.GMMode, campaignapp.GmModeAI)
	}
	if input.ThemePrompt != "stormy sea" {
		t.Fatalf("ThemePrompt = %q, want %q", input.ThemePrompt, "stormy sea")
	}

	if _, err := parseCreateCampaignInput(url.Values{"system": {"unknown"}}); err == nil {
		t.Fatalf("expected invalid system error")
	}
	if _, err := parseCreateCampaignInput(url.Values{"gm_mode": {"nope"}}); err == nil {
		t.Fatalf("expected invalid gm mode error")
	}
}

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

	participant := parseUpdateParticipantInput("  p-1  ", url.Values{
		"name":            {"  Lead  "},
		"role":            {"  gm  "},
		"pronouns":        {"  they/them  "},
		"campaign_access": {"  owner  "},
	})
	if participant.ParticipantID != "p-1" || participant.Name != "Lead" || participant.Role != "gm" || participant.Pronouns != "they/them" || participant.CampaignAccess != "owner" {
		t.Fatalf("participant input = %#v", participant)
	}

	campaign := parseUpdateCampaignInput(url.Values{
		"name":         {"  Voyage  "},
		"theme_prompt": {"  storm  "},
		"locale":       {"  pt-BR  "},
	})
	if campaign.Name == nil || campaign.ThemePrompt == nil || campaign.Locale == nil {
		t.Fatalf("campaign patch pointers should be set: %#v", campaign)
	}
	if *campaign.Name != "Voyage" || *campaign.ThemePrompt != "storm" || *campaign.Locale != "pt-BR" {
		t.Fatalf("campaign input = %#v", campaign)
	}
}
