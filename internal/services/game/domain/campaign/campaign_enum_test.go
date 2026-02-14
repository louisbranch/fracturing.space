package campaign

import (
	"testing"
)

func TestCampaignStatusFromLabel(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    CampaignStatus
		wantErr bool
	}{
		{name: "short draft", input: "DRAFT", want: CampaignStatusDraft},
		{name: "prefixed draft", input: "CAMPAIGN_STATUS_DRAFT", want: CampaignStatusDraft},
		{name: "lowercase draft", input: "draft", want: CampaignStatusDraft},
		{name: "short active", input: "ACTIVE", want: CampaignStatusActive},
		{name: "prefixed active", input: "CAMPAIGN_STATUS_ACTIVE", want: CampaignStatusActive},
		{name: "short completed", input: "COMPLETED", want: CampaignStatusCompleted},
		{name: "prefixed completed", input: "CAMPAIGN_STATUS_COMPLETED", want: CampaignStatusCompleted},
		{name: "short archived", input: "ARCHIVED", want: CampaignStatusArchived},
		{name: "prefixed archived", input: "CAMPAIGN_STATUS_ARCHIVED", want: CampaignStatusArchived},
		{name: "whitespace trimmed", input: "  DRAFT  ", want: CampaignStatusDraft},
		{name: "mixed case", input: "Active", want: CampaignStatusActive},
		{name: "empty string", input: "", wantErr: true},
		{name: "unknown value", input: "INVALID", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CampaignStatusFromLabel(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestGmModeFromLabel(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    GmMode
		wantErr bool
	}{
		{name: "short human", input: "HUMAN", want: GmModeHuman},
		{name: "prefixed human", input: "GM_MODE_HUMAN", want: GmModeHuman},
		{name: "short ai", input: "AI", want: GmModeAI},
		{name: "prefixed ai", input: "GM_MODE_AI", want: GmModeAI},
		{name: "short hybrid", input: "HYBRID", want: GmModeHybrid},
		{name: "prefixed hybrid", input: "GM_MODE_HYBRID", want: GmModeHybrid},
		{name: "lowercase", input: "human", want: GmModeHuman},
		{name: "whitespace trimmed", input: "  AI  ", want: GmModeAI},
		{name: "empty string", input: "", wantErr: true},
		{name: "unknown value", input: "INVALID", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GmModeFromLabel(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCampaignIntentFromLabel(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  CampaignIntent
	}{
		{name: "short standard", input: "STANDARD", want: CampaignIntentStandard},
		{name: "prefixed standard", input: "CAMPAIGN_INTENT_STANDARD", want: CampaignIntentStandard},
		{name: "short starter", input: "STARTER", want: CampaignIntentStarter},
		{name: "prefixed starter", input: "CAMPAIGN_INTENT_STARTER", want: CampaignIntentStarter},
		{name: "short sandbox", input: "SANDBOX", want: CampaignIntentSandbox},
		{name: "prefixed sandbox", input: "CAMPAIGN_INTENT_SANDBOX", want: CampaignIntentSandbox},
		{name: "lowercase", input: "starter", want: CampaignIntentStarter},
		{name: "whitespace trimmed", input: "  SANDBOX  ", want: CampaignIntentSandbox},
		{name: "empty defaults to standard", input: "", want: CampaignIntentStandard},
		{name: "unknown defaults to standard", input: "INVALID", want: CampaignIntentStandard},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CampaignIntentFromLabel(tt.input)
			if got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCampaignAccessPolicyFromLabel(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  CampaignAccessPolicy
	}{
		{name: "short private", input: "PRIVATE", want: CampaignAccessPolicyPrivate},
		{name: "prefixed private", input: "CAMPAIGN_ACCESS_POLICY_PRIVATE", want: CampaignAccessPolicyPrivate},
		{name: "short restricted", input: "RESTRICTED", want: CampaignAccessPolicyRestricted},
		{name: "prefixed restricted", input: "CAMPAIGN_ACCESS_POLICY_RESTRICTED", want: CampaignAccessPolicyRestricted},
		{name: "short public", input: "PUBLIC", want: CampaignAccessPolicyPublic},
		{name: "prefixed public", input: "CAMPAIGN_ACCESS_POLICY_PUBLIC", want: CampaignAccessPolicyPublic},
		{name: "lowercase", input: "public", want: CampaignAccessPolicyPublic},
		{name: "whitespace trimmed", input: "  RESTRICTED  ", want: CampaignAccessPolicyRestricted},
		{name: "empty defaults to private", input: "", want: CampaignAccessPolicyPrivate},
		{name: "unknown defaults to private", input: "INVALID", want: CampaignAccessPolicyPrivate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CampaignAccessPolicyFromLabel(tt.input)
			if got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}
