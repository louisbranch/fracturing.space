package participants

import (
	"net/url"
	"strings"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
)

// parseCreateParticipantInput normalizes the participant-create form into app input.
func parseCreateParticipantInput(form url.Values) campaignapp.CreateParticipantInput {
	return campaignapp.CreateParticipantInput{
		Name:           strings.TrimSpace(form.Get("name")),
		Role:           strings.TrimSpace(form.Get("role")),
		CampaignAccess: strings.TrimSpace(form.Get("campaign_access")),
	}
}

// parseUpdateParticipantInput normalizes the participant-edit form into app input.
func parseUpdateParticipantInput(participantID string, form url.Values) campaignapp.UpdateParticipantInput {
	return campaignapp.UpdateParticipantInput{
		ParticipantID:  strings.TrimSpace(participantID),
		Name:           strings.TrimSpace(form.Get("name")),
		Role:           strings.TrimSpace(form.Get("role")),
		Pronouns:       strings.TrimSpace(form.Get("pronouns")),
		CampaignAccess: strings.TrimSpace(form.Get("campaign_access")),
	}
}
