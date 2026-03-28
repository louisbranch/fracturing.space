package overview

import (
	"net/url"
	"testing"
)

func TestParseUpdateCampaignInputTrimsWhitespace(t *testing.T) {
	t.Parallel()

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
