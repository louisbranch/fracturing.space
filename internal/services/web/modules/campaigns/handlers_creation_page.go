package campaigns

import (
	"net/http"

	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
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

	workflow := h.resolveWorkflow(page.workspace.System)
	if workflow == nil {
		h.WriteNotFound(w, r)
		return
	}

	creation, err := h.service.CampaignCharacterCreation(ctx, campaignID, characterID, page.locale, workflow)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	creationView := workflow.CreationView(creation)

	// Resolve character name for breadcrumbs from the already-fetched profile.
	characterName := creation.Profile.CharacterName
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

	view := webtemplates.CharacterCreationPageView{
		CampaignID:  campaignID,
		CharacterID: characterID,
		Creation:    creationView,
	}

	h.WritePage(w, r, webtemplates.T(page.loc, "game.character_creation.title"), http.StatusOK,
		header, layout, webtemplates.CharacterCreationPage(view, page.loc))
}
