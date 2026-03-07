package game

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
)

func TestShouldClearCampaignAIBindingOnAccessChange(t *testing.T) {
	tests := []struct {
		name   string
		before participant.CampaignAccess
		after  participant.CampaignAccess
		want   bool
	}{
		{
			name:   "unchanged owner access does not clear",
			before: participant.CampaignAccessOwner,
			after:  participant.CampaignAccessOwner,
			want:   false,
		},
		{
			name:   "unchanged member access does not clear",
			before: participant.CampaignAccessMember,
			after:  participant.CampaignAccessMember,
			want:   false,
		},
		{
			name:   "demoting owner clears binding",
			before: participant.CampaignAccessOwner,
			after:  participant.CampaignAccessManager,
			want:   true,
		},
		{
			name:   "promoting to owner clears binding",
			before: participant.CampaignAccessManager,
			after:  participant.CampaignAccessOwner,
			want:   true,
		},
		{
			name:   "non-owner access change does not clear",
			before: participant.CampaignAccessMember,
			after:  participant.CampaignAccessManager,
			want:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldClearCampaignAIBindingOnAccessChange(tc.before, tc.after)
			if got != tc.want {
				t.Fatalf("shouldClearCampaignAIBindingOnAccessChange(%q, %q) = %t, want %t", tc.before, tc.after, got, tc.want)
			}
		})
	}
}
