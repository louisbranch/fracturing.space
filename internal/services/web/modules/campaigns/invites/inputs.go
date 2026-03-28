package invites

import (
	"net/url"
	"strings"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
)

// parseCreateInviteInput normalizes the invite-create form into app input.
func parseCreateInviteInput(form url.Values) campaignapp.CreateInviteInput {
	return campaignapp.CreateInviteInput{
		ParticipantID:     strings.TrimSpace(form.Get("participant_id")),
		RecipientUsername: strings.TrimSpace(form.Get("username")),
	}
}

// parseRevokeInviteInput normalizes the invite revoke form into app input.
func parseRevokeInviteInput(form url.Values) campaignapp.RevokeInviteInput {
	return campaignapp.RevokeInviteInput{InviteID: strings.TrimSpace(form.Get("invite_id"))}
}
