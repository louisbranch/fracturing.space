package campaigns

import (
	"strings"

	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// participantsView builds the participants detail view for one campaign.
func (p *campaignPageContext) participantsView(campaignID string, items []campaignapp.CampaignParticipant, viewerUserID string, canManage bool) campaignrender.ParticipantsPageView {
	view := campaignrender.ParticipantsPageView{CampaignDetailBaseView: p.baseDetailView(campaignID)}
	view.CanManageParticipants = canManage
	view.Participants = mapParticipantsView(items, viewerUserID)
	return view
}

// participantsBreadcrumbs returns breadcrumbs for the participants list page.
func (p *campaignPageContext) participantsBreadcrumbs() []sharedtemplates.BreadcrumbItem {
	return []sharedtemplates.BreadcrumbItem{
		{Label: webtemplates.T(p.loc, "game.participants.title")},
	}
}

// participantCreateView builds the participant-create detail view for one campaign.
func (p *campaignPageContext) participantCreateView(campaignID string, creator campaignapp.CampaignParticipantCreator) campaignrender.ParticipantCreatePageView {
	view := campaignrender.ParticipantCreatePageView{CampaignDetailBaseView: p.baseDetailView(campaignID)}
	view.CanManageParticipants = true
	view.ParticipantCreator = mapParticipantCreatorView(creator)
	return view
}

// participantCreateBreadcrumbs returns breadcrumbs for the participant-create page.
func (p *campaignPageContext) participantCreateBreadcrumbs(campaignID string) []sharedtemplates.BreadcrumbItem {
	return []sharedtemplates.BreadcrumbItem{
		{Label: webtemplates.T(p.loc, "game.participants.title"), URL: routepath.AppCampaignParticipants(campaignID)},
		{Label: webtemplates.T(p.loc, "game.participants.action_add")},
	}
}

// participantEditView builds the participant-edit detail view for one campaign.
func (p *campaignPageContext) participantEditView(
	campaignID string,
	participantID string,
	editor campaignapp.CampaignParticipantEditor,
	aiBinding *campaignapp.CampaignAIBindingEditor,
) campaignrender.ParticipantEditPageView {
	view := campaignrender.ParticipantEditPageView{CampaignDetailBaseView: p.baseDetailView(campaignID)}
	view.ParticipantID = strings.TrimSpace(editor.Participant.ID)
	if view.ParticipantID == "" {
		view.ParticipantID = participantID
	}
	view.ParticipantEditor = mapParticipantEditorView(editor)
	if aiBinding != nil {
		view.AIBindingEditor = mapAIBindingEditorView(*aiBinding)
	}
	return view
}

// participantEditBreadcrumbs returns breadcrumbs for the participant-edit page.
func (p *campaignPageContext) participantEditBreadcrumbs(campaignID string) []sharedtemplates.BreadcrumbItem {
	return []sharedtemplates.BreadcrumbItem{
		{Label: webtemplates.T(p.loc, "game.participants.title"), URL: routepath.AppCampaignParticipants(campaignID)},
		{Label: webtemplates.T(p.loc, "game.participants.action_edit")},
	}
}
