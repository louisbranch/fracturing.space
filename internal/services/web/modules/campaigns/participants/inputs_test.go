package participants

import (
	"net/url"
	"testing"
)

func TestParseUpdateInputsTrimWhitespace(t *testing.T) {
	t.Parallel()

	participant := parseUpdateParticipantInput("  p-1  ", url.Values{
		"name":            {"  Lead  "},
		"role":            {"  gm  "},
		"pronouns":        {"  they/them  "},
		"campaign_access": {"  owner  "},
	})
	if participant.ParticipantID != "p-1" || participant.Name != "Lead" || participant.Role != "gm" || participant.Pronouns != "they/them" || participant.CampaignAccess != "owner" {
		t.Fatalf("participant input = %#v", participant)
	}
}
