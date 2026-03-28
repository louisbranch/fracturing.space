package participants

import (
	"strings"

	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigndetail "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/detail"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// participantsView builds the participant list page view from workspace state.
func participantsView(page *campaigndetail.PageContext, campaignID string, items []campaignapp.CampaignParticipant, viewerUserID string, canManage bool) campaignrender.ParticipantsPageView {
	view := campaignrender.ParticipantsPageView{CampaignDetailBaseView: page.BaseDetailView(campaignID)}
	view.CanManageParticipants = canManage
	view.Participants = mapParticipantsView(items, viewerUserID)
	return view
}

// participantsBreadcrumbs returns the root breadcrumb trail for the participants surface.
func participantsBreadcrumbs(page *campaigndetail.PageContext) []sharedtemplates.BreadcrumbItem {
	return []sharedtemplates.BreadcrumbItem{
		{Label: webtemplates.T(page.Loc, "game.participants.title")},
	}
}

// participantCreateView builds the participant-create page view from creator state.
func participantCreateView(page *campaigndetail.PageContext, campaignID string, creator campaignapp.CampaignParticipantCreator) campaignrender.ParticipantCreatePageView {
	view := campaignrender.ParticipantCreatePageView{CampaignDetailBaseView: page.BaseDetailView(campaignID)}
	view.CanManageParticipants = true
	view.ParticipantCreator = mapParticipantCreatorView(creator)
	return view
}

// participantCreateBreadcrumbs returns breadcrumbs for the participant-create page.
func participantCreateBreadcrumbs(page *campaigndetail.PageContext, campaignID string) []sharedtemplates.BreadcrumbItem {
	return []sharedtemplates.BreadcrumbItem{
		{Label: webtemplates.T(page.Loc, "game.participants.title"), URL: routepath.AppCampaignParticipants(campaignID)},
		{Label: webtemplates.T(page.Loc, "game.participants.action_add")},
	}
}

// participantEditView builds the participant-edit page view from editor state.
func participantEditView(page *campaigndetail.PageContext, campaignID string, participantID string, editor campaignapp.CampaignParticipantEditor) campaignrender.ParticipantEditPageView {
	view := campaignrender.ParticipantEditPageView{CampaignDetailBaseView: page.BaseDetailView(campaignID)}
	view.ParticipantID = strings.TrimSpace(editor.Participant.ID)
	if view.ParticipantID == "" {
		view.ParticipantID = participantID
	}
	view.ParticipantEditor = mapParticipantEditorView(editor)
	return view
}

// participantEditBreadcrumbs returns breadcrumbs for the participant-edit page.
func participantEditBreadcrumbs(page *campaigndetail.PageContext, campaignID string) []sharedtemplates.BreadcrumbItem {
	return []sharedtemplates.BreadcrumbItem{
		{Label: webtemplates.T(page.Loc, "game.participants.title"), URL: routepath.AppCampaignParticipants(campaignID)},
		{Label: webtemplates.T(page.Loc, "game.participants.action_edit")},
	}
}

// mapParticipantsView projects app participants into render rows.
func mapParticipantsView(items []campaignapp.CampaignParticipant, viewerUserID string) []campaignrender.ParticipantView {
	viewerUserID = strings.TrimSpace(viewerUserID)
	result := make([]campaignrender.ParticipantView, 0, len(items))
	for _, p := range items {
		result = append(result, campaignrender.ParticipantView{
			ID:             p.ID,
			Name:           p.Name,
			Role:           p.Role,
			CampaignAccess: p.CampaignAccess,
			Controller:     p.Controller,
			Pronouns:       p.Pronouns,
			AvatarURL:      p.AvatarURL,
			IsViewer:       viewerUserID != "" && strings.EqualFold(strings.TrimSpace(p.UserID), viewerUserID),
			CanEdit:        p.CanEdit,
			EditReasonCode: p.EditReasonCode,
		})
	}
	return result
}

// mapParticipantEditorView projects editor state into render-owned form values.
func mapParticipantEditorView(editor campaignapp.CampaignParticipantEditor) campaignrender.ParticipantEditorView {
	accessOptions := make([]campaignrender.ParticipantAccessOptionView, 0, len(editor.AccessOptions))
	for _, option := range editor.AccessOptions {
		accessOptions = append(accessOptions, campaignrender.ParticipantAccessOptionView{
			Value:   option.Value,
			Allowed: option.Allowed,
		})
	}
	return campaignrender.ParticipantEditorView{
		ID:             editor.Participant.ID,
		Name:           editor.Participant.Name,
		Role:           editor.Participant.Role,
		Controller:     editor.Participant.Controller,
		Pronouns:       editor.Participant.Pronouns,
		CampaignAccess: editor.Participant.CampaignAccess,
		AllowGMRole:    editor.AllowGMRole,
		RoleReadOnly:   editor.RoleReadOnly,
		AccessOptions:  accessOptions,
		AccessReadOnly: editor.AccessReadOnly,
	}
}

// mapParticipantCreatorView projects creator defaults into render-owned form values.
func mapParticipantCreatorView(creator campaignapp.CampaignParticipantCreator) campaignrender.ParticipantCreatorView {
	accessOptions := make([]campaignrender.ParticipantAccessOptionView, 0, len(creator.AccessOptions))
	for _, option := range creator.AccessOptions {
		accessOptions = append(accessOptions, campaignrender.ParticipantAccessOptionView{
			Value:   option.Value,
			Allowed: option.Allowed,
		})
	}
	return campaignrender.ParticipantCreatorView{
		Name:           creator.Name,
		Role:           creator.Role,
		CampaignAccess: creator.CampaignAccess,
		AllowGMRole:    creator.AllowGMRole,
		AccessOptions:  accessOptions,
	}
}
