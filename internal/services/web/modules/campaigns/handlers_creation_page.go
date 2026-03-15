package campaigns

import (
	"net/http"

	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// handleCharacterCreationPage renders the dedicated character creation page
// with a full-width layout (no campaign sidebar).
func (h handlers) handleCharacterCreationPage(w http.ResponseWriter, r *http.Request, campaignID, characterID string) {
	ctx, page, err := h.loadCampaignPage(w, r, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	if !h.creation.pages.Enabled(page.workspace.System) {
		h.WriteNotFound(w, r)
		return
	}

	creationPage, err := h.creation.pages.LoadPage(ctx, campaignID, characterID, page.locale, page.workspace.System)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	// Resolve character name for breadcrumbs from the already-fetched profile.
	characterName := creationPage.CharacterName
	if characterName == "" {
		characterName = webtemplates.T(page.loc, "game.character_detail.title")
	}

	crumbs := campaignBreadcrumbs(campaignID, page.workspace.Name, page.loc,
		sharedtemplates.BreadcrumbItem{Label: webtemplates.T(page.loc, "game.characters.title"), URL: routepath.AppCampaignCharacters(campaignID)},
		sharedtemplates.BreadcrumbItem{Label: characterName, URL: routepath.AppCampaignCharacter(campaignID, characterID)},
		sharedtemplates.BreadcrumbItem{Label: webtemplates.T(page.loc, "game.character_creation.title")},
	)

	header := &webtemplates.AppMainHeader{
		Title:       webtemplates.T(page.loc, "game.character_creation.title"),
		Breadcrumbs: crumbs,
	}

	// Full-width layout: no SideMenu, campaign workspace route area.
	layout := webtemplates.AppMainLayoutOptions{
		Metadata: webtemplates.AppMainLayoutMetadata{
			RouteArea: webtemplates.RouteAreaCampaignWorkspace,
		},
	}

	h.WritePage(w, r, webtemplates.T(page.loc, "game.character_creation.title"), http.StatusOK,
		header, layout, campaignrender.CharacterCreationPage(campaignrender.NewCharacterCreationPageView(campaignID, characterID, creationPage), page.loc))
}
