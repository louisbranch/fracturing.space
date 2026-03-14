package campaigns

import (
	"net/http"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
)

// handleCharacters handles this route in the module transport layer.
func (h handlers) handleCharacters(w http.ResponseWriter, r *http.Request, campaignID string) {
	viewerUserID := h.RequestUserID(r)
	ctx, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	readContext := campaignapp.CharacterReadContext{
		System:       page.workspace.System,
		Locale:       page.locale,
		ViewerUserID: viewerUserID,
	}
	items, err := h.characterReads.CampaignCharacters(ctx, campaignID, readContext)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := page.charactersView(campaignID, items, h.authorization.RequireMutateCharacters(ctx, campaignID) == nil, h.creationPages.Enabled(page.workspace.System))
	h.writeCampaignDetailPage(w, r, page, campaignID, campaignrender.CharactersFragment(view, page.loc), page.charactersBreadcrumbs()...)
}

// handleCharacterCreatePage handles this route in the module transport layer.
func (h handlers) handleCharacterCreatePage(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	if err := h.authorization.RequireMutateCharacters(ctx, campaignID); err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := page.characterCreateView(campaignID)
	h.writeCampaignDetailPage(
		w,
		r,
		page,
		campaignID,
		campaignrender.CharacterCreateFragment(view, page.loc),
		page.characterCreateBreadcrumbs(campaignID)...,
	)
}

// handleCharacterEdit handles this route in the module transport layer.
func (h handlers) handleCharacterEdit(w http.ResponseWriter, r *http.Request, campaignID, characterID string) {
	ctx, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	editor, err := h.characterReads.CampaignCharacterEditor(ctx, campaignID, characterID, campaignapp.CharacterReadContext{
		System:       page.workspace.System,
		Locale:       page.locale,
		ViewerUserID: h.RequestUserID(r),
	})
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := page.characterEditView(campaignID, characterID, editor)
	h.writeCampaignDetailPage(
		w,
		r,
		page,
		campaignID,
		campaignrender.CharacterEditFragment(view, page.loc),
		page.characterEditBreadcrumbs(campaignID, characterID, view)...,
	)
}

// handleCharacterDetail handles this route in the module transport layer.
func (h handlers) handleCharacterDetail(w http.ResponseWriter, r *http.Request, campaignID, characterID string) {
	userID := h.RequestUserID(r)
	ctx, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	readContext := campaignapp.CharacterReadContext{
		System:       page.workspace.System,
		Locale:       page.locale,
		ViewerUserID: userID,
	}
	characterItem, err := h.characterReads.CampaignCharacter(ctx, campaignID, characterID, readContext)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	control, err := h.characterControl.CampaignCharacterControl(ctx, campaignID, characterID, userID, readContext)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	creationEnabled := h.creationPages.Enabled(page.workspace.System)
	var creation campaignrender.CampaignCharacterCreationView
	if creationEnabled {
		creationPage, err := h.creationPages.LoadPage(ctx, campaignID, characterID, page.locale, page.workspace.System)
		if err != nil {
			h.WriteError(w, r, err)
			return
		}
		creation = creationPage.Creation
	}
	view := page.characterDetailView(campaignID, characterID, characterItem, control, creationEnabled, creation)
	h.writeCampaignDetailPage(
		w,
		r,
		page,
		campaignID,
		campaignrender.CharacterDetailFragment(view, page.loc),
		page.characterDetailBreadcrumbs(campaignID, view)...,
	)
}
