package campaigns

import (
	"net/http"
	"strings"

	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// handleCharacters handles this route in the module transport layer.
func (h handlers) handleCharacters(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	items, err := h.service.CampaignCharacters(ctx, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := page.detailView(campaignID, markerCharacters)
	if err := h.service.RequireMutateCharacters(ctx, campaignID); err == nil {
		view.CanCreateCharacter = true
	}
	view.Characters = mapCharactersView(items)
	view.CharacterCreationEnabled = h.resolveWorkflow(page.workspace.System) != nil
	h.writeCampaignDetailPage(w, r, page, campaignID, view, sharedtemplates.BreadcrumbItem{Label: webtemplates.T(page.loc, "game.characters.title")})
}

// handleCharacterCreatePage handles this route in the module transport layer.
func (h handlers) handleCharacterCreatePage(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	view := page.detailView(campaignID, markerCharacterCreate)
	if err := h.service.RequireMutateCharacters(ctx, campaignID); err != nil {
		h.WriteError(w, r, err)
		return
	}
	view.CanCreateCharacter = true
	view.CharacterEditor = webtemplates.CampaignCharacterEditorView{Kind: "PC"}
	h.writeCampaignDetailPage(
		w,
		r,
		page,
		campaignID,
		view,
		sharedtemplates.BreadcrumbItem{Label: webtemplates.T(page.loc, "game.characters.title"), URL: routepath.AppCampaignCharacters(campaignID)},
		sharedtemplates.BreadcrumbItem{Label: webtemplates.T(page.loc, "game.characters.submit_create")},
	)
}

// handleCharacterEdit handles this route in the module transport layer.
func (h handlers) handleCharacterEdit(w http.ResponseWriter, r *http.Request, campaignID, characterID string) {
	ctx, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	editor, err := h.service.CampaignCharacterEditor(ctx, campaignID, characterID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := page.detailView(campaignID, markerCharacterEdit)
	view.CharacterID = characterID
	view.CharacterEditor = mapCharacterEditorView(editor)
	if strings.TrimSpace(view.CharacterID) == "" {
		view.CharacterID = characterID
	}
	view.Characters = []webtemplates.CampaignCharacterView{{
		ID:                      editor.Character.ID,
		Name:                    editor.Character.Name,
		Kind:                    editor.Character.Kind,
		Controller:              editor.Character.Controller,
		ControllerParticipantID: editor.Character.ControllerParticipantID,
		Pronouns:                editor.Character.Pronouns,
		CanEdit:                 true,
		EditReasonCode:          editor.Character.EditReasonCode,
	}}
	h.writeCampaignDetailPage(
		w,
		r,
		page,
		campaignID,
		view,
		sharedtemplates.BreadcrumbItem{Label: webtemplates.T(page.loc, "game.characters.title"), URL: routepath.AppCampaignCharacters(campaignID)},
		sharedtemplates.BreadcrumbItem{Label: campaignCharacterEditBreadcrumbLabel(page.loc, view), URL: routepath.AppCampaignCharacter(campaignID, characterID)},
		sharedtemplates.BreadcrumbItem{Label: webtemplates.T(page.loc, "game.characters.action_edit_page")},
	)
}

// handleCharacterDetail handles this route in the module transport layer.
func (h handlers) handleCharacterDetail(w http.ResponseWriter, r *http.Request, campaignID, characterID string) {
	userID := h.RequestUserID(r)
	ctx, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	characterItems, err := h.service.CampaignCharacters(ctx, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := page.detailView(campaignID, markerCharacterDetail)
	view.CharacterID = characterID
	view.Characters = mapCharactersView(characterItems)
	control, err := h.service.CampaignCharacterControl(ctx, campaignID, characterID, userID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view.CharacterControl = mapCharacterControlView(control)
	workflow := h.resolveWorkflow(page.workspace.System)
	view.CharacterCreationEnabled = workflow != nil
	if view.CharacterCreationEnabled {
		creation, err := h.service.CampaignCharacterCreation(ctx, campaignID, characterID, page.locale, workflow)
		if err != nil {
			h.WriteError(w, r, err)
			return
		}
		view.CharacterCreation = workflow.CreationView(creation)
	}
	h.writeCampaignDetailPage(
		w,
		r,
		page,
		campaignID,
		view,
		sharedtemplates.BreadcrumbItem{Label: webtemplates.T(page.loc, "game.characters.title"), URL: routepath.AppCampaignCharacters(campaignID)},
		sharedtemplates.BreadcrumbItem{Label: campaignCharacterBreadcrumbLabel(page.loc, view)},
	)
}

// campaignCharacterBreadcrumbLabel resolves the selected character breadcrumb label.
func campaignCharacterBreadcrumbLabel(loc webtemplates.Localizer, view webtemplates.CampaignDetailView) string {
	selectedCharacterID := strings.TrimSpace(view.CharacterID)
	if selectedCharacterID == "" {
		return webtemplates.T(loc, "game.character_detail.title")
	}
	for _, character := range view.Characters {
		if strings.TrimSpace(character.ID) != selectedCharacterID {
			continue
		}
		characterName := strings.TrimSpace(character.Name)
		if characterName != "" {
			return characterName
		}
		break
	}
	return webtemplates.T(loc, "game.character_detail.title")
}

// campaignCharacterEditBreadcrumbLabel resolves the selected character label for edit breadcrumbs.
func campaignCharacterEditBreadcrumbLabel(loc webtemplates.Localizer, view webtemplates.CampaignDetailView) string {
	if name := strings.TrimSpace(view.CharacterEditor.Name); name != "" {
		return name
	}
	return campaignCharacterBreadcrumbLabel(loc, view)
}
