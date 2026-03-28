package characters

import (
	"fmt"
	"net/http"

	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigndetail "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/detail"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
	campaignworkflow "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// ServiceConfig groups character read, control, mutation, and creation app
// config.
type ServiceConfig struct {
	Read          campaignapp.CharacterReadServiceConfig
	Control       campaignapp.CharacterControlServiceConfig
	Mutation      campaignapp.CharacterMutationServiceConfig
	Creation      campaignapp.CharacterCreationServiceConfig
	Authorization campaignapp.AuthorizationGateway
}

// HandlerServices groups character read, control, mutation, and creation
// workflow behavior.
type HandlerServices struct {
	reads            campaignapp.CampaignCharacterReadService
	control          campaignapp.CampaignCharacterControlService
	mutation         campaignapp.CampaignCharacterMutationService
	creationPages    campaignworkflow.PageService
	creationMutation campaignworkflow.MutationService
}

// NewHandlerServices keeps character transport dependencies owned by the
// character surface instead of the campaigns root constructor.
func NewHandlerServices(config ServiceConfig, workflows campaignworkflow.Registry) (HandlerServices, error) {
	reads, err := campaignapp.NewCharacterReadService(config.Read, config.Authorization)
	if err != nil {
		return HandlerServices{}, fmt.Errorf("character-reads: %w", err)
	}
	control, err := campaignapp.NewCharacterControlService(config.Control, config.Authorization)
	if err != nil {
		return HandlerServices{}, fmt.Errorf("character-control: %w", err)
	}
	mutation, err := campaignapp.NewCharacterMutationService(config.Mutation, config.Authorization)
	if err != nil {
		return HandlerServices{}, fmt.Errorf("character-mutation: %w", err)
	}

	pageService, err := campaignapp.NewCharacterCreationPageService(config.Creation)
	if err != nil {
		return HandlerServices{}, fmt.Errorf("creation-pages: %w", err)
	}
	pages, err := campaignworkflow.NewPageAppService(pageService)
	if err != nil {
		return HandlerServices{}, fmt.Errorf("creation-pages adapter: %w", err)
	}
	mutationService, err := campaignapp.NewCharacterCreationMutationService(config.Creation, config.Authorization)
	if err != nil {
		return HandlerServices{}, fmt.Errorf("creation-flow: %w", err)
	}
	flow, err := campaignworkflow.NewMutationAppService(mutationService)
	if err != nil {
		return HandlerServices{}, fmt.Errorf("creation-flow adapter: %w", err)
	}

	return HandlerServices{
		reads:            reads,
		control:          control,
		mutation:         mutation,
		creationPages:    campaignworkflow.NewPageService(pages, workflows),
		creationMutation: campaignworkflow.NewMutationService(flow, workflows),
	}, nil
}

// Handler owns character detail and character-creation routes.
type Handler struct {
	campaigndetail.Handler
	characters HandlerServices
}

// NewHandler assembles the character route-owner handler.
func NewHandler(detail campaigndetail.Handler, services HandlerServices) Handler {
	return Handler{
		Handler:    detail,
		characters: services,
	}
}

// HandleCharacters renders the campaign character collection.
func (h Handler) HandleCharacters(w http.ResponseWriter, r *http.Request, campaignID string) {
	viewerUserID := h.RequestUserID(r)
	ctx, page, ok := h.LoadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	readContext := campaignapp.CharacterReadContext{
		System:       page.Workspace.System,
		Locale:       page.Locale,
		ViewerUserID: viewerUserID,
	}
	items, err := h.characters.reads.CampaignCharacters(ctx, campaignID, readContext)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := charactersView(page, campaignID, items, h.Pages.Authorization.RequireMutateCharacters(ctx, campaignID) == nil, h.characters.creationPages.Enabled(page.Workspace.System))
	h.WriteCampaignDetailPage(w, r, page, campaignID, campaignrender.CharactersFragment(view, page.Loc), charactersBreadcrumbs(page)...)
}

// HandleCharacterCreatePage renders the dedicated character create page.
func (h Handler) HandleCharacterCreatePage(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, ok := h.LoadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	if err := h.Pages.Authorization.RequireMutateCharacters(ctx, campaignID); err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := characterCreateView(page, campaignID)
	h.WriteCampaignDetailPage(
		w,
		r,
		page,
		campaignID,
		campaignrender.CharacterCreateFragment(view, page.Loc),
		characterCreateBreadcrumbs(page, campaignID)...,
	)
}

// HandleCharacterCreate creates a character and redirects into the creation
// workflow when that workflow is enabled for the campaign system.
func (h Handler) HandleCharacterCreate(w http.ResponseWriter, r *http.Request, campaignID string) {
	if !httpx.ParseFormOrRedirectErrorNotice(w, r, "error.web.message.failed_to_parse_character_create_form", routepath.AppCampaign(campaignID)) {
		return
	}
	input, err := parseCreateCharacterInput(r.Form)
	if err != nil {
		h.WriteMutationError(w, r, err, "error.web.message.failed_to_create_character", routepath.AppCampaign(campaignID))
		return
	}

	ctx, _ := h.RequestContextAndUserID(r)
	created, err := h.characters.mutation.CreateCharacter(ctx, campaignID, input)
	if err != nil {
		h.WriteMutationError(w, r, err, "error.web.message.failed_to_create_character", routepath.AppCampaign(campaignID))
		return
	}

	workspace, err := h.Pages.Workspace.CampaignWorkspace(ctx, campaignID)
	if err == nil && h.characters.creationPages.Enabled(workspace.System) {
		h.WriteMutationSuccess(w, r, "web.campaigns.notice_character_created", routepath.AppCampaignCharacterCreation(campaignID, created.CharacterID))
		return
	}
	h.WriteMutationSuccess(w, r, "web.campaigns.notice_character_created", routepath.AppCampaignCharacter(campaignID, created.CharacterID))
}

// HandleCharacterEdit renders the character editor.
func (h Handler) HandleCharacterEdit(w http.ResponseWriter, r *http.Request, campaignID, characterID string) {
	ctx, page, ok := h.LoadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	editor, err := h.characters.reads.CampaignCharacterEditor(ctx, campaignID, characterID, campaignapp.CharacterReadContext{
		System:       page.Workspace.System,
		Locale:       page.Locale,
		ViewerUserID: h.RequestUserID(r),
	})
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := characterEditView(page, campaignID, characterID, editor)
	h.WriteCampaignDetailPage(
		w,
		r,
		page,
		campaignID,
		campaignrender.CharacterEditFragment(view, page.Loc),
		characterEditBreadcrumbs(page, campaignID, characterID, view)...,
	)
}

// HandleCharacterDetail renders one campaign character.
func (h Handler) HandleCharacterDetail(w http.ResponseWriter, r *http.Request, campaignID, characterID string) {
	userID := h.RequestUserID(r)
	ctx, page, ok := h.LoadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	readContext := campaignapp.CharacterReadContext{
		System:       page.Workspace.System,
		Locale:       page.Locale,
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
	creationEnabled := h.characters.creationPages.Enabled(page.Workspace.System)
	var creation campaignrender.CampaignCharacterCreationView
	if creationEnabled {
		creationPage, err := h.characters.creationPages.LoadPage(ctx, campaignID, characterID, page.Locale, page.Workspace.System)
		if err != nil {
			h.WriteError(w, r, err)
			return
		}
		creation = campaignrender.NewCharacterCreationView(creationPage.Creation)
	}
	view := characterDetailView(page, campaignID, characterID, characterItem, control, creationEnabled, creation)
	h.WriteCampaignDetailPage(
		w,
		r,
		page,
		campaignID,
		campaignrender.CharacterDetailFragment(view, page.Loc),
		characterDetailBreadcrumbs(page, campaignID, view)...,
	)
}

// HandleCharacterUpdate updates character metadata.
func (h Handler) HandleCharacterUpdate(w http.ResponseWriter, r *http.Request, campaignID, characterID string) {
	if !httpx.ParseFormOrRedirectErrorNotice(w, r, "error.web.message.failed_to_parse_character_update_form", routepath.AppCampaignCharacter(campaignID, characterID)) {
		return
	}
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.characters.mutation.UpdateCharacter(ctx, campaignID, characterID, parseUpdateCharacterInput(r.Form)); err != nil {
		h.WriteMutationError(w, r, err, "error.web.message.failed_to_update_character", routepath.AppCampaignCharacter(campaignID, characterID))
		return
	}
	h.WriteMutationSuccess(w, r, "web.campaigns.notice_character_updated", routepath.AppCampaignCharacter(campaignID, characterID))
}

// HandleCharacterControlSet updates the character controller from the detail
// page.
func (h Handler) HandleCharacterControlSet(w http.ResponseWriter, r *http.Request, campaignID, characterID string) {
	redirectURL := routepath.AppCampaignCharacter(campaignID, characterID)
	if !httpx.ParseFormOrRedirectErrorNotice(w, r, "error.web.message.failed_to_parse_character_controller_form", redirectURL) {
		return
	}
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.characters.control.SetCharacterController(ctx, campaignID, characterID, parseSetCharacterControllerInput(r.Form)); err != nil {
		h.WriteMutationError(w, r, err, "error.web.message.failed_to_set_character_controller", redirectURL)
		return
	}
	h.WriteMutationSuccess(w, r, "web.campaigns.notice_character_controller_updated", redirectURL)
}

// HandleCharacterControlClaim claims an unassigned character for the current
// participant.
func (h Handler) HandleCharacterControlClaim(w http.ResponseWriter, r *http.Request, campaignID, characterID string) {
	redirectURL := routepath.AppCampaignCharacter(campaignID, characterID)
	if !httpx.ParseFormOrRedirectErrorNotice(w, r, "error.web.message.failed_to_parse_character_controller_form", redirectURL) {
		return
	}
	ctx, userID := h.RequestContextAndUserID(r)
	if err := h.characters.control.ClaimCharacterControl(ctx, campaignID, characterID, userID); err != nil {
		h.WriteMutationError(w, r, err, "error.web.message.failed_to_claim_character_control", redirectURL)
		return
	}
	h.WriteMutationSuccess(w, r, "web.campaigns.notice_character_control_claimed", redirectURL)
}

// HandleCharacterControlRelease releases the current participant's control.
func (h Handler) HandleCharacterControlRelease(w http.ResponseWriter, r *http.Request, campaignID, characterID string) {
	redirectURL := routepath.AppCampaignCharacter(campaignID, characterID)
	if !httpx.ParseFormOrRedirectErrorNotice(w, r, "error.web.message.failed_to_parse_character_controller_form", redirectURL) {
		return
	}
	ctx, userID := h.RequestContextAndUserID(r)
	if err := h.characters.control.ReleaseCharacterControl(ctx, campaignID, characterID, userID); err != nil {
		h.WriteMutationError(w, r, err, "error.web.message.failed_to_release_character_control", redirectURL)
		return
	}
	h.WriteMutationSuccess(w, r, "web.campaigns.notice_character_control_released", redirectURL)
}

// HandleCharacterDelete removes a character from the campaign.
func (h Handler) HandleCharacterDelete(w http.ResponseWriter, r *http.Request, campaignID, characterID string) {
	redirectURL := routepath.AppCampaignCharacter(campaignID, characterID)
	if !httpx.ParseFormOrRedirectErrorNotice(w, r, "error.web.message.failed_to_parse_character_delete_form", redirectURL) {
		return
	}
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.characters.mutation.DeleteCharacter(ctx, campaignID, characterID); err != nil {
		h.WriteMutationError(w, r, err, "error.web.message.failed_to_delete_character", redirectURL)
		return
	}
	h.WriteMutationSuccess(w, r, "web.campaigns.notice_character_deleted", routepath.AppCampaignCharacters(campaignID))
}

// HandleCharacterCreationPage renders the dedicated character creation page
// with a full-width layout.
func (h Handler) HandleCharacterCreationPage(w http.ResponseWriter, r *http.Request, campaignID, characterID string) {
	ctx, page, err := h.LoadCampaignPage(w, r, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	if !h.characters.creationPages.Enabled(page.Workspace.System) {
		h.WriteNotFound(w, r)
		return
	}

	creationPage, err := h.characters.creationPages.LoadPage(ctx, campaignID, characterID, page.Locale, page.Workspace.System)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	characterName := creationPage.CharacterName
	if characterName == "" {
		characterName = webtemplates.T(page.Loc, "game.character_detail.title")
	}

	crumbs := campaigndetail.CampaignBreadcrumbs(campaignID, page.Workspace.Name, page.Loc,
		sharedtemplates.BreadcrumbItem{Label: webtemplates.T(page.Loc, "game.characters.title"), URL: routepath.AppCampaignCharacters(campaignID)},
		sharedtemplates.BreadcrumbItem{Label: characterName, URL: routepath.AppCampaignCharacter(campaignID, characterID)},
		sharedtemplates.BreadcrumbItem{Label: webtemplates.T(page.Loc, "game.character_creation.title")},
	)
	header := &webtemplates.AppMainHeader{
		Title:       webtemplates.T(page.Loc, "game.character_creation.title"),
		Breadcrumbs: crumbs,
	}
	layout := webtemplates.AppMainLayoutOptions{
		Metadata: webtemplates.AppMainLayoutMetadata{
			RouteArea: webtemplates.RouteAreaCampaignWorkspace,
		},
	}

	h.WritePage(w, r, webtemplates.T(page.Loc, "game.character_creation.title"), http.StatusOK,
		header, layout, campaignrender.CharacterCreationPage(campaignrender.NewCharacterCreationPageView(campaignID, characterID, creationPage), page.Loc))
}

// HandleCharacterCreationStep applies the next character creation workflow
// step.
func (h Handler) HandleCharacterCreationStep(w http.ResponseWriter, r *http.Request, campaignID, characterID string) {
	if !httpx.ParseFormOrRedirectErrorNotice(w, r, "error.web.message.failed_to_parse_character_creation_form", routepath.AppCampaignCharacterCreation(campaignID, characterID)) {
		return
	}

	ctx, _ := h.RequestContextAndUserID(r)
	workspace, err := h.Pages.Workspace.CampaignWorkspace(ctx, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	if !h.characters.creationPages.Enabled(workspace.System) {
		h.WriteNotFound(w, r)
		return
	}
	if err := h.characters.creationMutation.ApplyStep(ctx, campaignID, characterID, workspace.System, r.Form); err != nil {
		h.writeCreationStepError(w, r, err, campaignID, characterID)
		return
	}

	httpx.WriteRedirect(w, r, routepath.AppCampaignCharacterCreation(campaignID, characterID))
}

// HandleCharacterCreationReset resets the character creation workflow.
func (h Handler) HandleCharacterCreationReset(w http.ResponseWriter, r *http.Request, campaignID, characterID string) {
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.characters.creationMutation.Reset(ctx, campaignID, characterID); err != nil {
		h.writeCreationStepError(w, r, err, campaignID, characterID)
		return
	}
	httpx.WriteRedirect(w, r, routepath.AppCampaignCharacterCreation(campaignID, characterID))
}

// writeCreationStepError keeps character-creation failures in the same workflow page.
func (h Handler) writeCreationStepError(w http.ResponseWriter, r *http.Request, err error, campaignID, characterID string) {
	key := apperrors.LocalizationKey(err)
	if key == "" {
		key = "error.web.message.character_creation_step_failed"
	}
	flash.Write(w, r, flash.Notice{Kind: flash.KindError, Key: key})
	httpx.WriteRedirect(w, r, routepath.AppCampaignCharacterCreation(campaignID, characterID))
}
