package projection

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
)

func TestParseCampaignIntentDefaulting(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  campaign.Intent
	}{
		{"empty", "", campaign.IntentStandard},
		{"standard", "standard", campaign.IntentStandard},
		{"starter", "STARTER", campaign.IntentStarter},
		{"sandbox", "campaign_intent_sandbox", campaign.IntentSandbox},
		{"unknown", "invalid", campaign.IntentStandard},
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
		want  campaign.AccessPolicy
	}{
		{"empty", "", campaign.AccessPolicyPrivate},
		{"private", "private", campaign.AccessPolicyPrivate},
		{"restricted", "RESTRICTED", campaign.AccessPolicyRestricted},
		{"public", "campaign_access_policy_public", campaign.AccessPolicyPublic},
		{"unknown", "invalid", campaign.AccessPolicyPrivate},
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
