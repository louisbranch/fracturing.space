package policy

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/participant"
)

func TestCanRejectsUnknownAction(t *testing.T) {
	actor := participant.Participant{CampaignAccess: participant.CampaignAccessOwner}
	if Can(actor, Action(99), campaign.Campaign{}) {
		t.Fatal("expected unknown action to be rejected")
	}
}

func TestCanManageParticipants(t *testing.T) {
	tests := []struct {
		name   string
		access participant.CampaignAccess
		want   bool
	}{
		{"owner", participant.CampaignAccessOwner, true},
		{"manager", participant.CampaignAccessManager, true},
		{"member", participant.CampaignAccessMember, false},
		{"unspecified", participant.CampaignAccessUnspecified, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actor := participant.Participant{CampaignAccess: tt.access}
			if got := Can(actor, ActionManageParticipants, campaign.Campaign{}); got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestCanManageInvites(t *testing.T) {
	tests := []struct {
		name   string
		access participant.CampaignAccess
		want   bool
	}{
		{"owner", participant.CampaignAccessOwner, true},
		{"manager", participant.CampaignAccessManager, true},
		{"member", participant.CampaignAccessMember, false},
		{"unspecified", participant.CampaignAccessUnspecified, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actor := participant.Participant{CampaignAccess: tt.access}
			if got := Can(actor, ActionManageInvites, campaign.Campaign{}); got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}
