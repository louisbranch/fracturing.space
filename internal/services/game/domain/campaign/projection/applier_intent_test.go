package projection

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
)

func TestParseCampaignIntentDefaulting(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  campaign.CampaignIntent
	}{
		{"empty", "", campaign.CampaignIntentStandard},
		{"standard", "standard", campaign.CampaignIntentStandard},
		{"starter", "STARTER", campaign.CampaignIntentStarter},
		{"sandbox", "campaign_intent_sandbox", campaign.CampaignIntentSandbox},
		{"unknown", "invalid", campaign.CampaignIntentStandard},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCampaignIntent(tt.input)
			if got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestParseCampaignAccessPolicyDefaulting(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  campaign.CampaignAccessPolicy
	}{
		{"empty", "", campaign.CampaignAccessPolicyPrivate},
		{"private", "private", campaign.CampaignAccessPolicyPrivate},
		{"restricted", "RESTRICTED", campaign.CampaignAccessPolicyRestricted},
		{"public", "campaign_access_policy_public", campaign.CampaignAccessPolicyPublic},
		{"unknown", "invalid", campaign.CampaignAccessPolicyPrivate},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCampaignAccessPolicy(tt.input)
			if got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}
