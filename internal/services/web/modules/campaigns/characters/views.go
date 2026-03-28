package characters

import (
	"strings"

	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigndetail "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/detail"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// charactersView builds the character-list page view from workspace data.
func charactersView(page *campaigndetail.PageContext, campaignID string, items []campaignapp.CampaignCharacter, canCreate, creationEnabled bool) campaignrender.CharactersPageView {
	view := campaignrender.CharactersPageView{CampaignDetailBaseView: page.BaseDetailView(campaignID)}
	view.CanCreateCharacter = canCreate
	view.Characters = mapCharactersView(items)
	view.CharacterCreationEnabled = creationEnabled
	return view
}

// charactersBreadcrumbs returns the root breadcrumb trail for the characters surface.
func charactersBreadcrumbs(page *campaigndetail.PageContext) []sharedtemplates.BreadcrumbItem {
	return []sharedtemplates.BreadcrumbItem{
		{Label: webtemplates.T(page.Loc, "game.characters.title")},
	}
}

// characterCreateView builds the dedicated character-create page view.
func characterCreateView(page *campaigndetail.PageContext, campaignID string) campaignrender.CharacterCreatePageView {
	view := campaignrender.CharacterCreatePageView{CampaignDetailBaseView: page.BaseDetailView(campaignID)}
	view.CanCreateCharacter = true
	view.CharacterEditor = campaignrender.CharacterEditorView{Kind: "PC"}
	return view
}

// characterCreateBreadcrumbs returns breadcrumbs for the character-create page.
func characterCreateBreadcrumbs(page *campaigndetail.PageContext, campaignID string) []sharedtemplates.BreadcrumbItem {
	return []sharedtemplates.BreadcrumbItem{
		{Label: webtemplates.T(page.Loc, "game.characters.title"), URL: routepath.AppCampaignCharacters(campaignID)},
		{Label: webtemplates.T(page.Loc, "game.characters.submit_create")},
	}
}

// characterEditView builds the character-edit page view from editor state.
func characterEditView(page *campaigndetail.PageContext, campaignID, characterID string, editor campaignapp.CampaignCharacterEditor) campaignrender.CharacterEditPageView {
	view := campaignrender.CharacterEditPageView{CampaignDetailBaseView: page.BaseDetailView(campaignID)}
	view.CharacterID = strings.TrimSpace(characterID)
	view.CharacterEditor = mapCharacterEditorView(editor)
	view.Character = mapCharacterView(editor.Character)
	return view
}

// characterEditBreadcrumbs returns breadcrumbs for the character-edit page.
func characterEditBreadcrumbs(page *campaigndetail.PageContext, campaignID, characterID string, view campaignrender.CharacterEditPageView) []sharedtemplates.BreadcrumbItem {
	return []sharedtemplates.BreadcrumbItem{
		{Label: webtemplates.T(page.Loc, "game.characters.title"), URL: routepath.AppCampaignCharacters(campaignID)},
		{Label: campaignCharacterEditBreadcrumbLabel(page.Loc, view), URL: routepath.AppCampaignCharacter(campaignID, characterID)},
		{Label: webtemplates.T(page.Loc, "game.characters.action_edit_page")},
	}
}

// characterDetailView builds the character-detail page view for one selected character.
func characterDetailView(
	page *campaigndetail.PageContext,
	campaignID string,
	characterID string,
	character campaignapp.CampaignCharacter,
	ownership campaignapp.CampaignCharacterOwnership,
	creationEnabled bool,
	creation campaignrender.CampaignCharacterCreationView,
) campaignrender.CharacterDetailPageView {
	view := campaignrender.CharacterDetailPageView{CampaignDetailBaseView: page.BaseDetailView(campaignID)}
	view.CharacterID = strings.TrimSpace(characterID)
	view.Character = mapCharacterView(character)
	view.CharacterOwnership = mapCharacterOwnershipView(ownership)
	view.CharacterCreationEnabled = creationEnabled
	if creationEnabled {
		view.CharacterCreation = creation
	}
	return view
}

// characterDetailBreadcrumbs returns breadcrumbs for the character-detail page.
func characterDetailBreadcrumbs(page *campaigndetail.PageContext, campaignID string, view campaignrender.CharacterDetailPageView) []sharedtemplates.BreadcrumbItem {
	return []sharedtemplates.BreadcrumbItem{
		{Label: webtemplates.T(page.Loc, "game.characters.title"), URL: routepath.AppCampaignCharacters(campaignID)},
		{Label: campaignCharacterBreadcrumbLabel(page.Loc, view)},
	}
}

// campaignCharacterBreadcrumbLabel picks the most specific breadcrumb label for a character page.
func campaignCharacterBreadcrumbLabel(loc webtemplates.Localizer, view campaignrender.CharacterDetailPageView) string {
	characterName := strings.TrimSpace(view.Character.Name)
	if characterName != "" {
		return characterName
	}
	return webtemplates.T(loc, "game.character_detail.title")
}

// campaignCharacterEditBreadcrumbLabel keeps the edit breadcrumb aligned with the current form state.
func campaignCharacterEditBreadcrumbLabel(loc webtemplates.Localizer, view campaignrender.CharacterEditPageView) string {
	if name := strings.TrimSpace(view.CharacterEditor.Name); name != "" {
		return name
	}
	characterName := strings.TrimSpace(view.Character.Name)
	if characterName != "" {
		return characterName
	}
	return webtemplates.T(loc, "game.character_detail.title")
}

// mapCharactersView projects app characters into render rows.
func mapCharactersView(items []campaignapp.CampaignCharacter) []campaignrender.CharacterView {
	result := make([]campaignrender.CharacterView, 0, len(items))
	for _, c := range items {
		result = append(result, mapCharacterView(c))
	}
	return result
}

// mapCharacterView projects one app character into render state.
func mapCharacterView(c campaignapp.CampaignCharacter) campaignrender.CharacterView {
	return campaignrender.CharacterView{
		ID:                 c.ID,
		Name:               c.Name,
		Kind:               c.Kind,
		Owner:              c.Owner,
		OwnerParticipantID: c.OwnerParticipantID,
		Pronouns:           c.Pronouns,
		Aliases:            append([]string(nil), c.Aliases...),
		AvatarURL:          c.AvatarURL,
		OwnedByViewer:      c.OwnedByViewer,
		CanEdit:            c.CanEdit,
		EditReasonCode:     c.EditReasonCode,
		Daggerheart:        mapCharacterDaggerheartSummaryView(c.Daggerheart),
	}
}

// mapCharacterDaggerheartSummaryView keeps optional system metadata local to the render seam.
func mapCharacterDaggerheartSummaryView(summary *campaignapp.CampaignCharacterDaggerheartSummary) *campaignrender.CharacterDaggerheartSummaryView {
	if summary == nil {
		return nil
	}
	return &campaignrender.CharacterDaggerheartSummaryView{
		Level:         summary.Level,
		ClassName:     summary.ClassName,
		SubclassName:  summary.SubclassName,
		HeritageName:  summary.HeritageName,
		CommunityName: summary.CommunityName,
	}
}

// mapCharacterEditorView projects editor state into render-friendly form values.
func mapCharacterEditorView(editor campaignapp.CampaignCharacterEditor) campaignrender.CharacterEditorView {
	return campaignrender.CharacterEditorView{
		ID:       editor.Character.ID,
		Name:     editor.Character.Name,
		Pronouns: editor.Character.Pronouns,
		Kind:     editor.Character.Kind,
	}
}

// mapCharacterOwnershipView projects ownership options into render-owned card state.
func mapCharacterOwnershipView(ownership campaignapp.CampaignCharacterOwnership) campaignrender.CharacterOwnershipView {
	options := make([]campaignrender.CharacterOwnershipOptionView, 0, len(ownership.Options))
	for _, option := range ownership.Options {
		options = append(options, campaignrender.CharacterOwnershipOptionView{
			ParticipantID: option.ParticipantID,
			Label:         option.Label,
			Selected:      option.Selected,
		})
	}
	return campaignrender.CharacterOwnershipView{
		CurrentOwnerName:   ownership.CurrentOwnerName,
		CanManageOwnership: ownership.CanManageOwnership,
		Options:            options,
	}
}
