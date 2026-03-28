package render

import (
	"github.com/a-h/templ"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// CharactersPageView carries character-list page state only.
type CharactersPageView struct {
	CampaignDetailBaseView
	Characters               []CharacterView
	CharacterCreationEnabled bool
}

// CharacterCreatePageView carries character-create page state only.
type CharacterCreatePageView struct {
	CampaignDetailBaseView
	CharacterEditor CharacterEditorView
}

// CharacterEditPageView carries character-edit page state only.
type CharacterEditPageView struct {
	CampaignDetailBaseView
	CharacterID     string
	Character       CharacterView
	CharacterEditor CharacterEditorView
}

// CharacterDetailPageView carries character-detail page state only.
type CharacterDetailPageView struct {
	CampaignDetailBaseView
	CharacterID              string
	Character                CharacterView
	CharacterControl         CharacterControlView
	CharacterCreationEnabled bool
	CharacterCreation        CampaignCharacterCreationView
}

// CharactersFragment renders the character-list page.
func CharactersFragment(view CharactersPageView, loc webtemplates.Localizer) templ.Component {
	return charactersFragment(view, loc)
}

// CharacterCreateFragment renders the character-create page.
func CharacterCreateFragment(view CharacterCreatePageView, loc webtemplates.Localizer) templ.Component {
	return characterCreateFragment(view, loc)
}

// CharacterEditFragment renders the character-edit page.
func CharacterEditFragment(view CharacterEditPageView, loc webtemplates.Localizer) templ.Component {
	return characterEditFragment(view, loc)
}

// CharacterDetailFragment renders the character-detail page.
func CharacterDetailFragment(view CharacterDetailPageView, loc webtemplates.Localizer) templ.Component {
	return characterDetailFragment(view, loc)
}

// CharacterView carries character summary rows for campaign detail pages.
type CharacterView struct {
	ID                      string
	Name                    string
	Kind                    string
	Controller              string
	ControllerParticipantID string
	Pronouns                string
	Aliases                 []string
	AvatarURL               string
	OwnedByViewer           bool
	CanEdit                 bool
	EditReasonCode          string
	Daggerheart             *CharacterDaggerheartSummaryView
}

// CharacterDaggerheartSummaryView keeps optional Daggerheart card metadata
// local to the campaigns detail render seam.
type CharacterDaggerheartSummaryView struct {
	Level         int32
	ClassName     string
	SubclassName  string
	HeritageName  string
	CommunityName string
}

// CharacterEditorView carries character edit form state for campaign detail
// pages.
type CharacterEditorView struct {
	ID       string
	Name     string
	Pronouns string
	Kind     string
}

// CharacterControlOptionView carries control-target options for one character.
type CharacterControlOptionView struct {
	ParticipantID string
	Label         string
	Selected      bool
}

// CharacterControlView carries character control affordances for the selected
// character detail page.
type CharacterControlView struct {
	CurrentParticipantName string
	CanSelfClaim           bool
	CanSelfRelease         bool
	CanManageControl       bool
	Options                []CharacterControlOptionView
}
