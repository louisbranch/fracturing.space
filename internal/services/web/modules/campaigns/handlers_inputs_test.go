package campaigns

import (
	"net/url"
	"testing"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
)

func TestParseCreateCampaignInputDefaultsAndValidation(t *testing.T) {
	t.Parallel()

	systems := newTestCampaignSystems()
	input, err := parseCreateCampaignInput(url.Values{
		"name":         {"  Voyage  "},
		"theme_prompt": {"  stormy sea  "},
	}, systems)
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

	if _, err := parseCreateCampaignInput(url.Values{"system": {"unknown"}}, systems); err == nil {
		t.Fatalf("expected invalid system error")
	}
	if _, err := parseCreateCampaignInput(url.Values{"gm_mode": {"nope"}}, systems); err == nil {
		t.Fatalf("expected invalid gm mode error")
	}
}

func TestParseUpdateInputsTrimWhitespace(t *testing.T) {
	t.Parallel()
}
