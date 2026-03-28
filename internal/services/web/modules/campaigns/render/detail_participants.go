package render

import (
	"github.com/a-h/templ"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// ParticipantsPageView carries participant-list page state only.
type ParticipantsPageView struct {
	CampaignDetailBaseView
	Participants []ParticipantView
}

// ParticipantCreatePageView carries participant-create page state only.
type ParticipantCreatePageView struct {
	CampaignDetailBaseView
	ParticipantCreator ParticipantCreatorView
}

// ParticipantEditPageView carries participant-edit page state only.
type ParticipantEditPageView struct {
	CampaignDetailBaseView
	ParticipantID     string
	ParticipantEditor ParticipantEditorView
}

// ParticipantsFragment renders the participant-list page.
func ParticipantsFragment(view ParticipantsPageView, loc webtemplates.Localizer) templ.Component {
	return participantsFragment(view, loc)
}

// ParticipantCreateFragment renders the participant-create page.
func ParticipantCreateFragment(view ParticipantCreatePageView, loc webtemplates.Localizer) templ.Component {
	return participantCreateFragment(view, loc)
}

// ParticipantEditFragment renders the participant-edit page.
func ParticipantEditFragment(view ParticipantEditPageView, loc webtemplates.Localizer) templ.Component {
	return participantEditFragment(view, loc)
}

// ParticipantView carries participant rows without forcing handlers to depend
// on shared template models directly.
type ParticipantView struct {
	ID             string
	Name           string
	Role           string
	CampaignAccess string
	Controller     string
	Pronouns       string
	AvatarURL      string
	IsViewer       bool
	CanEdit        bool
	EditReasonCode string
}

// ParticipantAccessOptionView keeps participant-access select options local to
// the campaigns detail render seam.
type ParticipantAccessOptionView struct {
	Value   string
	Allowed bool
}

// AIAgentOptionView carries AI agent choices for campaign AI-binding forms.
type AIAgentOptionView struct {
	ID       string
	Name     string
	Enabled  bool
	Selected bool
}

// ParticipantEditorView carries participant edit form state for campaign
// detail pages.
type ParticipantEditorView struct {
	ID             string
	Name           string
	Role           string
	Controller     string
	Pronouns       string
	CampaignAccess string
	AllowGMRole    bool
	RoleReadOnly   bool
	AccessOptions  []ParticipantAccessOptionView
	AccessReadOnly bool
	Delete         ParticipantDeleteView
}

// ParticipantDeleteView carries participant delete danger-zone state for
// campaign detail pages.
type ParticipantDeleteView struct {
	Visible                       bool
	Enabled                       bool
	HasAssociatedUser             bool
	BlockedByOwnedCharacters      bool
	BlockedByControlledCharacters bool
}

// ParticipantCreatorView carries participant creation form state for campaign
// detail pages.
type ParticipantCreatorView struct {
	Name           string
	Role           string
	CampaignAccess string
	AllowGMRole    bool
	AccessOptions  []ParticipantAccessOptionView
}
