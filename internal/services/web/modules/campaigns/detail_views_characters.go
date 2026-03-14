package campaigns

import (
	"strings"

	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// charactersView builds the characters detail view for one campaign.
func (p *campaignPageContext) charactersView(campaignID string, items []campaignapp.CampaignCharacter, canCreate, creationEnabled bool) campaignrender.CharactersPageView {
	view := campaignrender.CharactersPageView{CampaignDetailBaseView: p.baseDetailView(campaignID)}
	view.CanCreateCharacter = canCreate
	view.Characters = mapCharactersView(items)
	view.CharacterCreationEnabled = creationEnabled
	return view
}

// charactersBreadcrumbs returns breadcrumbs for the characters list page.
func (p *campaignPageContext) charactersBreadcrumbs() []sharedtemplates.BreadcrumbItem {
	return []sharedtemplates.BreadcrumbItem{
		{Label: webtemplates.T(p.loc, "game.characters.title")},
	}
}

// characterCreateView builds the character-create detail view for one campaign.
func (p *campaignPageContext) characterCreateView(campaignID string) campaignrender.CharacterCreatePageView {
	view := campaignrender.CharacterCreatePageView{CampaignDetailBaseView: p.baseDetailView(campaignID)}
	view.CanCreateCharacter = true
	view.CharacterEditor = campaignrender.CharacterEditorView{Kind: "PC"}
	return view
}

// characterCreateBreadcrumbs returns breadcrumbs for the character-create page.
func (p *campaignPageContext) characterCreateBreadcrumbs(campaignID string) []sharedtemplates.BreadcrumbItem {
	return []sharedtemplates.BreadcrumbItem{
		{Label: webtemplates.T(p.loc, "game.characters.title"), URL: routepath.AppCampaignCharacters(campaignID)},
		{Label: webtemplates.T(p.loc, "game.characters.submit_create")},
	}
}

// characterEditView builds the character-edit detail view for one campaign.
func (p *campaignPageContext) characterEditView(campaignID, characterID string, editor campaignapp.CampaignCharacterEditor) campaignrender.CharacterEditPageView {
	view := campaignrender.CharacterEditPageView{CampaignDetailBaseView: p.baseDetailView(campaignID)}
	view.CharacterID = strings.TrimSpace(characterID)
	view.CharacterEditor = mapCharacterEditorView(editor)
	view.Character = mapCharacterView(editor.Character)
	return view
}

// characterEditBreadcrumbs returns breadcrumbs for the character-edit page.
func (p *campaignPageContext) characterEditBreadcrumbs(campaignID, characterID string, view campaignrender.CharacterEditPageView) []sharedtemplates.BreadcrumbItem {
	return []sharedtemplates.BreadcrumbItem{
		{Label: webtemplates.T(p.loc, "game.characters.title"), URL: routepath.AppCampaignCharacters(campaignID)},
		{Label: campaignCharacterEditBreadcrumbLabel(p.loc, view), URL: routepath.AppCampaignCharacter(campaignID, characterID)},
		{Label: webtemplates.T(p.loc, "game.characters.action_edit_page")},
	}
}

// characterDetailView builds the character-detail view for one campaign.
func (p *campaignPageContext) characterDetailView(
	campaignID string,
	characterID string,
	character campaignapp.CampaignCharacter,
	control campaignapp.CampaignCharacterControl,
	creationEnabled bool,
	creation campaignrender.CampaignCharacterCreationView,
) campaignrender.CharacterDetailPageView {
	view := campaignrender.CharacterDetailPageView{CampaignDetailBaseView: p.baseDetailView(campaignID)}
	view.CharacterID = strings.TrimSpace(characterID)
	view.Character = mapCharacterView(character)
	view.CharacterControl = mapCharacterControlView(control)
	view.CharacterCreationEnabled = creationEnabled
	if creationEnabled {
		view.CharacterCreation = creation
	}
	return view
}

// characterDetailBreadcrumbs returns breadcrumbs for the character-detail page.
func (p *campaignPageContext) characterDetailBreadcrumbs(campaignID string, view campaignrender.CharacterDetailPageView) []sharedtemplates.BreadcrumbItem {
	return []sharedtemplates.BreadcrumbItem{
		{Label: webtemplates.T(p.loc, "game.characters.title"), URL: routepath.AppCampaignCharacters(campaignID)},
		{Label: campaignCharacterBreadcrumbLabel(p.loc, view)},
	}
}

// campaignCharacterBreadcrumbLabel resolves the selected character breadcrumb label.
func campaignCharacterBreadcrumbLabel(loc webtemplates.Localizer, view campaignrender.CharacterDetailPageView) string {
	characterName := strings.TrimSpace(view.Character.Name)
	if characterName != "" {
		return characterName
	}
	return webtemplates.T(loc, "game.character_detail.title")
}

// campaignCharacterEditBreadcrumbLabel resolves the selected character label for edit breadcrumbs.
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
