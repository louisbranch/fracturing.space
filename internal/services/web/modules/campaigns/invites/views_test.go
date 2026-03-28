package invites

import (
	"testing"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
)

func TestMapInviteSeatOptionsShowsOnlyAvailableHumanSeats(t *testing.T) {
	t.Parallel()

	options := mapInviteSeatOptions(
		[]campaignapp.CampaignParticipant{
			{ID: "p-open-b", Name: "Bryn", Controller: "Human"},
			{ID: "p-pending", Name: "Ari", Controller: "Human"},
			{ID: "p-bound", Name: "Cato", Controller: "Human", UserID: "user-1"},
			{ID: "p-ai", Name: "Oracle", Controller: "AI"},
			{ID: "p-open-a", Name: "Ada", Controller: "controller_human"},
		},
		[]campaignapp.CampaignInvite{
			{ID: "inv-1", ParticipantID: "p-pending", Status: "Pending"},
			{ID: "inv-2", ParticipantID: "p-open-b", Status: "Claimed"},
		},
	)

	if len(options) != 2 {
		t.Fatalf("len(options) = %d, want 2", len(options))
	}
	if options[0].ParticipantID != "p-open-a" || options[0].Label != "Ada" {
		t.Fatalf("options[0] = %#v, want participant p-open-a / Ada", options[0])
	}
	if options[1].ParticipantID != "p-open-b" || options[1].Label != "Bryn" {
		t.Fatalf("options[1] = %#v, want participant p-open-b / Bryn", options[1])
	}
}
