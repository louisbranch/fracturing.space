package render

import (
	"github.com/a-h/templ"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// InvitesPageView carries invite-management page state only.
type InvitesPageView struct {
	CampaignDetailBaseView
	Invites []InviteView
}

// InviteCreatePageView carries invite-create page state only.
type InviteCreatePageView struct {
	CampaignDetailBaseView
	InviteSeatOptions []InviteSeatOptionView
}

// InvitesFragment renders the invite-management page.
func InvitesFragment(view InvitesPageView, loc webtemplates.Localizer) templ.Component {
	return invitesFragment(view, loc)
}

// InviteCreateFragment renders the invite-create page.
func InviteCreateFragment(view InviteCreatePageView, loc webtemplates.Localizer) templ.Component {
	return inviteCreateFragment(view, loc)
}

// InviteView carries invite rows for the invites detail page.
type InviteView struct {
	ID                string
	ParticipantID     string
	ParticipantName   string
	RecipientUsername string
	HasRecipient      bool
	PublicURL         string
	Status            string
}

// InviteSeatOptionView carries eligible invite-seat targets for invite forms.
type InviteSeatOptionView struct {
	ParticipantID string
	Label         string
}
