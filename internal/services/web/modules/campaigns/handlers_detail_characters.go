package campaigns

import (
	"net/http"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
	campaignworkflow "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow"
)

// characterHandlerServices groups character read, control, and mutation
// behavior.
type characterHandlerServices struct {
	reads    campaignapp.CampaignCharacterReadService
	control  campaignapp.CampaignCharacterControlService
	mutation campaignapp.CampaignCharacterMutationService
}

// campaignCreationAppServices keeps creation app-service assembly separate
// from workflow-owned transport services.
type campaignCreationAppServices struct {
	Pages campaignworkflow.PageAppService
	Flow  campaignworkflow.MutationAppService
}

// creationHandlerServices groups character-creation workflow behavior.
type creationHandlerServices struct {
	pages    campaignworkflow.PageService
	mutation campaignworkflow.MutationService
}

// characterHandlers owns character read, control, and mutation routes.
type characterHandlers struct {
	campaignDetailHandlers
	characters characterHandlerServices
	creation   creationHandlerServices
}

// creationHandlers owns character-creation page and workflow routes.
type creationHandlers struct {
	campaignDetailHandlers
	creation creationHandlerServices
}

// newCharacterHandlerServices keeps character transport dependencies owned by
// the character surface instead of the root constructor.
func newCharacterHandlerServices(config characterServiceConfig) characterHandlerServices {
	return characterHandlerServices{
		reads:    campaignapp.NewCharacterReadService(config.Read, config.Authorization),
		control:  campaignapp.NewCharacterControlService(config.Control, config.Authorization),
		mutation: campaignapp.NewCharacterMutationService(config.Mutation, config.Authorization),
	}
}

// newCampaignCreationAppServices keeps creation app-service assembly separate
// from workflow-owned route handling.
func newCampaignCreationAppServices(config characterServiceConfig) campaignCreationAppServices {
	return campaignCreationAppServices{
		Pages: campaignworkflow.NewPageAppService(campaignapp.NewCharacterCreationPageService(config.Creation)),
		Flow:  campaignworkflow.NewMutationAppService(campaignapp.NewCharacterCreationMutationService(config.Creation, config.Authorization)),
	}
}

// newCreationHandlerServices assembles workflow-owned creation services from
// the installed workflow registry.
func newCreationHandlerServices(services campaignCreationAppServices, workflows campaignworkflow.Registry) creationHandlerServices {
	return creationHandlerServices{
		pages:    campaignworkflow.NewPageService(services.Pages, workflows),
		mutation: campaignworkflow.NewMutationService(services.Flow, workflows),
	}
}

// newCharacterHandlers assembles the character route-owner handler.
func newCharacterHandlers(detail campaignDetailHandlers, services characterHandlerServices, creation creationHandlerServices) characterHandlers {
	return characterHandlers{
		campaignDetailHandlers: detail,
		characters:             services,
		creation:               creation,
	}
}

// newStandaloneCreationHandlers assembles the workflow route-owner handler.
func newStandaloneCreationHandlers(detail campaignDetailHandlers, creation creationHandlerServices) creationHandlers {
	return creationHandlers{
		campaignDetailHandlers: detail,
		creation:               creation,
	}
}

// missingCharacterHandlerServices reports which character routes would fail
// closed so module composition can reject an incomplete surface bundle early.
func missingCharacterHandlerServices(services characterHandlerServices) []string {
	missing := []string{}
	if services.reads == nil {
		missing = append(missing, "character-reads")
	}
	if services.control == nil {
		missing = append(missing, "character-control")
	}
	if services.mutation == nil {
		missing = append(missing, "character-mutation")
	}
	return missing
}

// missingCampaignCreationAppServices reports which workflow-owned creation
// services are absent before the standalone creation routes are mounted.
func missingCampaignCreationAppServices(services campaignCreationAppServices) []string {
	missing := []string{}
	if services.Pages == nil {
		missing = append(missing, "creation-pages")
	}
	if services.Flow == nil {
		missing = append(missing, "creation-flow")
	}
	return missing
}

// handleCharacters handles this route in the module transport layer.
func (h characterHandlers) handleCharacters(w http.ResponseWriter, r *http.Request, campaignID string) {
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
	items, err := h.characters.reads.CampaignCharacters(ctx, campaignID, readContext)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := page.charactersView(campaignID, items, h.pages.authorization.RequireMutateCharacters(ctx, campaignID) == nil, h.creation.pages.Enabled(page.workspace.System))
	h.writeCampaignDetailPage(w, r, page, campaignID, campaignrender.CharactersFragment(view, page.loc), page.charactersBreadcrumbs()...)
}

// handleCharacterCreatePage handles this route in the module transport layer.
func (h characterHandlers) handleCharacterCreatePage(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	if err := h.pages.authorization.RequireMutateCharacters(ctx, campaignID); err != nil {
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
func (h characterHandlers) handleCharacterEdit(w http.ResponseWriter, r *http.Request, campaignID, characterID string) {
	ctx, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	editor, err := h.characters.reads.CampaignCharacterEditor(ctx, campaignID, characterID, campaignapp.CharacterReadContext{
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
func (h characterHandlers) handleCharacterDetail(w http.ResponseWriter, r *http.Request, campaignID, characterID string) {
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
	characterItem, err := h.characters.reads.CampaignCharacter(ctx, campaignID, characterID, readContext)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	control, err := h.characters.control.CampaignCharacterControl(ctx, campaignID, characterID, userID, readContext)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	creationEnabled := h.creation.pages.Enabled(page.workspace.System)
	var creation campaignrender.CampaignCharacterCreationView
	if creationEnabled {
		creationPage, err := h.creation.pages.LoadPage(ctx, campaignID, characterID, page.locale, page.workspace.System)
		if err != nil {
			h.WriteError(w, r, err)
			return
		}
		creation = campaignrender.NewCharacterCreationView(creationPage.Creation)
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
