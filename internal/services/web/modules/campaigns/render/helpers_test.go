package render

import "testing"

func TestCampaignInviteStatusLabel_Declined(t *testing.T) {
	t.Parallel()

	loc := testLocalizer{
		"game.campaign_invites.value_declined": "Declined",
	}

	if got := campaignInviteStatusLabel(loc, "declined"); got != "Declined" {
		t.Fatalf("campaignInviteStatusLabel(declined) = %q, want %q", got, "Declined")
	}
}
