package campaigns

import (
	"net/http"
	"strings"

	"github.com/a-h/templ"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	module "github.com/louisbranch/fracturing.space/internal/services/web2/module"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web2/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web2/platform/httpx"
	web2i18n "github.com/louisbranch/fracturing.space/internal/services/web2/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web2/platform/pagerender"
	"github.com/louisbranch/fracturing.space/internal/services/web2/platform/webctx"
	"github.com/louisbranch/fracturing.space/internal/services/web2/platform/weberror"
	"github.com/louisbranch/fracturing.space/internal/services/web2/routepath"
	web2templates "github.com/louisbranch/fracturing.space/internal/services/web2/templates"
)

type handlers struct {
	service service
	deps    runtimeDependencies
}

type runtimeDependencies struct {
	resolveUserID    module.ResolveUserID
	resolveLanguage  module.ResolveLanguage
	resolveViewer    module.ResolveViewer
	chatFallbackPort string
}

func newRuntimeDependencies(deps module.Dependencies) runtimeDependencies {
	return runtimeDependencies{
		resolveUserID:    deps.ResolveUserID,
		resolveLanguage:  deps.ResolveLanguage,
		resolveViewer:    deps.ResolveViewer,
		chatFallbackPort: deps.ChatFallbackPort,
	}
}

func (d runtimeDependencies) moduleDependencies() module.Dependencies {
	return module.Dependencies{
		ResolveViewer:   d.resolveViewer,
		ResolveLanguage: d.resolveLanguage,
	}
}

func campaignsListHeader(loc web2templates.Localizer) *web2templates.AppMainHeader {
	return &web2templates.AppMainHeader{
		Title: web2templates.T(loc, "game.campaigns.title"),
		Action: &web2templates.AppMainHeaderAction{
			Label: web2templates.T(loc, "game.campaigns.start_new"),
			URL:   routepath.AppCampaignsCreate,
		},
	}
}

func campaignCreateHeader(loc web2templates.Localizer) *web2templates.AppMainHeader {
	return &web2templates.AppMainHeader{
		Title: web2templates.T(loc, "game.create.title"),
		Breadcrumbs: []sharedtemplates.BreadcrumbItem{
			{Label: web2templates.T(loc, "game.campaigns.title"), URL: routepath.AppCampaigns},
			{Label: web2templates.T(loc, "game.create.title")},
		},
	}
}

func campaignDetailHeader(route detailRoute, campaignLabel string, loc web2templates.Localizer) *web2templates.AppMainHeader {
	return &web2templates.AppMainHeader{
		Title:       web2templates.T(loc, "game.campaign.title"),
		Breadcrumbs: campaignDetailBreadcrumbs(route, campaignLabel, loc),
	}
}

func campaignChatView(route detailRoute, workspace CampaignWorkspace, chatFallbackPort string) web2templates.CampaignChatView {
	return web2templates.CampaignChatView{
		CampaignID:       route.campaignID,
		CampaignName:     workspace.Name,
		BackURL:          routepath.AppCampaign(route.campaignID),
		ChatFallbackPort: strings.TrimSpace(chatFallbackPort),
	}
}

func campaignDetailBreadcrumbs(route detailRoute, campaignLabel string, loc web2templates.Localizer) []sharedtemplates.BreadcrumbItem {
	campaignLabel = strings.TrimSpace(campaignLabel)
	if campaignLabel == "" {
		campaignLabel = route.campaignID
	}
	breadcrumbs := []sharedtemplates.BreadcrumbItem{
		{Label: web2templates.T(loc, "game.campaigns.title"), URL: routepath.AppCampaigns},
		{Label: campaignLabel},
	}

	switch route.kind {
	case detailSessions:
		breadcrumbs[1].URL = routepath.AppCampaign(route.campaignID)
		return append(breadcrumbs, sharedtemplates.BreadcrumbItem{Label: web2templates.T(loc, "game.sessions.title")})
	case detailSession:
		breadcrumbs[1].URL = routepath.AppCampaign(route.campaignID)
		return append(breadcrumbs,
			sharedtemplates.BreadcrumbItem{Label: web2templates.T(loc, "game.sessions.title"), URL: routepath.AppCampaignSessions(route.campaignID)},
			sharedtemplates.BreadcrumbItem{Label: route.sessionID},
		)
	case detailParticipants:
		breadcrumbs[1].URL = routepath.AppCampaign(route.campaignID)
		return append(breadcrumbs, sharedtemplates.BreadcrumbItem{Label: web2templates.T(loc, "game.participants.title")})
	case detailCharacters:
		breadcrumbs[1].URL = routepath.AppCampaign(route.campaignID)
		return append(breadcrumbs, sharedtemplates.BreadcrumbItem{Label: web2templates.T(loc, "game.characters.title")})
	case detailCharacter:
		breadcrumbs[1].URL = routepath.AppCampaign(route.campaignID)
		return append(breadcrumbs,
			sharedtemplates.BreadcrumbItem{Label: web2templates.T(loc, "game.characters.title"), URL: routepath.AppCampaignCharacters(route.campaignID)},
			sharedtemplates.BreadcrumbItem{Label: route.characterID},
		)
	case detailInvites:
		breadcrumbs[1].URL = routepath.AppCampaign(route.campaignID)
		return append(breadcrumbs, sharedtemplates.BreadcrumbItem{Label: web2templates.T(loc, "game.campaign_invites.title")})
	default:
		return breadcrumbs
	}
}

func campaignWorkspaceMenu(campaignID string, currentPath string, loc web2templates.Localizer) *web2templates.AppSideMenu {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return nil
	}
	overviewURL := routepath.AppCampaign(campaignID)
	participantsURL := routepath.AppCampaignParticipants(campaignID)
	charactersURL := routepath.AppCampaignCharacters(campaignID)
	return &web2templates.AppSideMenu{
		CurrentPath: strings.TrimSpace(currentPath),
		Items: []web2templates.AppSideMenuItem{
			{
				Label:       web2templates.T(loc, "game.campaign.menu.overview"),
				URL:         overviewURL,
				MatchPrefix: overviewURL,
				MatchExact:  true,
				IconID:      commonv1.IconId_ICON_ID_CAMPAIGN,
			},
			{
				Label:       web2templates.T(loc, "game.participants.title"),
				URL:         participantsURL,
				MatchPrefix: participantsURL,
				IconID:      commonv1.IconId_ICON_ID_PARTICIPANT,
			},
			{
				Label:       web2templates.T(loc, "game.characters.title"),
				URL:         charactersURL,
				MatchPrefix: charactersURL,
				IconID:      commonv1.IconId_ICON_ID_CHARACTER,
			},
		},
	}
}

func campaignMainStyle(coverImageURL string) string {
	coverImageURL = strings.TrimSpace(coverImageURL)
	if coverImageURL == "" {
		return ""
	}
	// TODO(web2-hardening): move cover image styling to template attributes or CSS classes so URL handling is not composed into inline style strings.
	safeCoverImageURL := strings.ReplaceAll(coverImageURL, "\"", "\\\"")
	return "background-image: linear-gradient(to bottom, rgba(0, 0, 0, 0.45), rgba(0, 0, 0, 0.55)), url(\"" + safeCoverImageURL + "\"); background-size: cover; background-position: center; background-repeat: no-repeat;"
}

func campaignMainClass(coverImageURL string) string {
	coverImageURL = strings.TrimSpace(coverImageURL)
	if coverImageURL == "" {
		return "max-w-none"
	}
	return "px-4"
}

func newHandlers(s service, deps module.Dependencies) handlers {
	return handlers{service: s, deps: newRuntimeDependencies(deps)}
}

func (h handlers) routeCampaignID(r *http.Request) (string, bool) {
	campaignID := strings.TrimSpace(r.PathValue("campaignID"))
	if campaignID == "" {
		return "", false
	}
	return campaignID, true
}

func (h handlers) handleOverviewRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.handleNotFound(w, r)
		return
	}
	h.handleDetail(w, r, detailRoute{campaignID: campaignID, kind: detailOverview})
}

func (h handlers) handleOverviewMethodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	httpx.MethodNotAllowed(http.MethodGet+", HEAD")(w, nil)
}

func (h handlers) handleSessionsRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.handleNotFound(w, r)
		return
	}
	h.handleDetail(w, r, detailRoute{campaignID: campaignID, kind: detailSessions})
}

func (h handlers) handleSessionDetailRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.handleNotFound(w, r)
		return
	}
	sessionID := strings.TrimSpace(r.PathValue("sessionID"))
	if sessionID == "" {
		h.handleNotFound(w, r)
		return
	}
	h.handleDetail(w, r, detailRoute{campaignID: campaignID, kind: detailSession, sessionID: sessionID})
}

func (h handlers) handleSessionStartRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.handleNotFound(w, r)
		return
	}
	h.handleMutation(w, r, detailRoute{campaignID: campaignID, kind: detailSessionStart})
}

func (h handlers) handleSessionEndRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.handleNotFound(w, r)
		return
	}
	h.handleMutation(w, r, detailRoute{campaignID: campaignID, kind: detailSessionEnd})
}

func (h handlers) handleParticipantsRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.handleNotFound(w, r)
		return
	}
	h.handleDetail(w, r, detailRoute{campaignID: campaignID, kind: detailParticipants})
}

func (h handlers) handleParticipantUpdateRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.handleNotFound(w, r)
		return
	}
	h.handleMutation(w, r, detailRoute{campaignID: campaignID, kind: detailParticipantUpdate})
}

func (h handlers) handleCharactersRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.handleNotFound(w, r)
		return
	}
	h.handleDetail(w, r, detailRoute{campaignID: campaignID, kind: detailCharacters})
}

func (h handlers) handleCharacterDetailRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.handleNotFound(w, r)
		return
	}
	characterID := strings.TrimSpace(r.PathValue("characterID"))
	if characterID == "" {
		h.handleNotFound(w, r)
		return
	}
	h.handleDetail(w, r, detailRoute{campaignID: campaignID, kind: detailCharacter, characterID: characterID})
}

func (h handlers) handleCharacterCreateRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.handleNotFound(w, r)
		return
	}
	h.handleMutation(w, r, detailRoute{campaignID: campaignID, kind: detailCharacterCreate})
}

func (h handlers) handleCharacterUpdateRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.handleNotFound(w, r)
		return
	}
	h.handleMutation(w, r, detailRoute{campaignID: campaignID, kind: detailCharacterUpdate})
}

func (h handlers) handleCharacterControlRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.handleNotFound(w, r)
		return
	}
	h.handleMutation(w, r, detailRoute{campaignID: campaignID, kind: detailCharacterControl})
}

func (h handlers) handleGameRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.handleNotFound(w, r)
		return
	}
	h.handleDetail(w, r, detailRoute{campaignID: campaignID, kind: detailGame})
}

func (h handlers) handleInvitesRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.handleNotFound(w, r)
		return
	}
	h.handleDetail(w, r, detailRoute{campaignID: campaignID, kind: detailInvites})
}

func (h handlers) handleInviteCreateRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.handleNotFound(w, r)
		return
	}
	h.handleMutation(w, r, detailRoute{campaignID: campaignID, kind: detailInviteCreate})
}

func (h handlers) handleInviteRevokeRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.handleNotFound(w, r)
		return
	}
	h.handleMutation(w, r, detailRoute{campaignID: campaignID, kind: detailInviteRevoke})
}

func (h handlers) handleNotFound(w http.ResponseWriter, r *http.Request) {
	weberror.WriteAppError(w, r, http.StatusNotFound, h.deps.moduleDependencies())
}

func (h handlers) handleCreateCampaign(w http.ResponseWriter, r *http.Request) {
	loc, _ := h.pageLocalizer(w, r)
	h.writeCampaignHTML(
		w,
		r,
		web2templates.T(loc, "game.create.title"),
		campaignCreateHeader(loc),
		web2templates.AppMainLayoutOptions{},
		web2templates.CampaignCreateFragment(web2templates.CampaignCreateFormValues{}, loc),
	)
}

func (h handlers) handleCreateCampaignSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.writeError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.failed_to_parse_campaign_create_form", "failed to parse campaign create form"))
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		h.writeError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.campaign_name_is_required", "campaign name is required"))
		return
	}

	systemValue := strings.TrimSpace(r.FormValue("system"))
	if systemValue == "" {
		systemValue = "daggerheart"
	}
	system, ok := parseAppGameSystem(systemValue)
	if !ok {
		h.writeError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.campaign_system_is_invalid", "campaign system is invalid"))
		return
	}

	gmModeValue := strings.TrimSpace(r.FormValue("gm_mode"))
	if gmModeValue == "" {
		gmModeValue = "human"
	}
	gmMode, ok := parseAppGmMode(gmModeValue)
	if !ok {
		h.writeError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.campaign_gm_mode_is_invalid", "campaign gm mode is invalid"))
		return
	}

	themePrompt := strings.TrimSpace(r.FormValue("theme_prompt"))
	creatorDisplayName := strings.TrimSpace(r.FormValue("creator_display_name"))
	resolvedLocale := platformi18n.LocaleForTag(web2i18n.ResolveTag(r, h.deps.resolveLanguage))
	if creatorDisplayName == "" && h.deps.resolveViewer != nil {
		creatorDisplayName = strings.TrimSpace(h.deps.resolveViewer(r).DisplayName)
	}

	created, err := h.service.createCampaign(webctx.WithResolvedUserID(r, h.deps.resolveUserID), CreateCampaignInput{
		Name:               name,
		System:             system,
		GMMode:             gmMode,
		ThemePrompt:        themePrompt,
		CreatorDisplayName: creatorDisplayName,
		Locale:             resolvedLocale,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	location := routepath.AppCampaign(created.CampaignID)
	if httpx.IsHTMXRequest(r) {
		httpx.WriteHXRedirect(w, location)
		return
	}
	http.Redirect(w, r, location, http.StatusFound)
}

func (h handlers) handleIndex(w http.ResponseWriter, r *http.Request) {
	loc, _ := h.pageLocalizer(w, r)
	items, err := h.service.listCampaigns(webctx.WithResolvedUserID(r, h.deps.resolveUserID))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	list := make([]web2templates.CampaignListItem, 0, len(items))
	for _, item := range items {
		list = append(list, web2templates.CampaignListItem{
			ID:               item.ID,
			Name:             item.Name,
			Theme:            item.Theme,
			CoverImageURL:    item.CoverImageURL,
			ParticipantCount: item.ParticipantCount,
			CharacterCount:   item.CharacterCount,
		})
	}
	h.writeCampaignHTML(w, r, web2templates.T(loc, "game.campaigns.title"), campaignsListHeader(loc), web2templates.AppMainLayoutOptions{}, web2templates.CampaignListFragment(list, loc))
}

func (h handlers) handleDetail(w http.ResponseWriter, r *http.Request, route detailRoute) {
	loc, lang := h.pageLocalizer(w, r)
	ctx := webctx.WithResolvedUserID(r, h.deps.resolveUserID)
	workspace, err := h.service.campaignWorkspace(ctx, route.campaignID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if route.kind == detailGame {
		h.writeCampaignChatHTML(w, r, campaignChatView(route, workspace, h.deps.chatFallbackPort), lang, loc)
		return
	}
	participants := []web2templates.CampaignParticipantView{}
	characters := []web2templates.CampaignCharacterView{}
	sessions := []web2templates.CampaignSessionView{}
	invites := []web2templates.CampaignInviteView{}
	if route.kind == detailParticipants {
		participantItems, participantsErr := h.service.campaignParticipants(ctx, route.campaignID)
		if participantsErr != nil {
			h.writeError(w, r, participantsErr)
			return
		}
		participants = make([]web2templates.CampaignParticipantView, 0, len(participantItems))
		for _, participant := range participantItems {
			participants = append(participants, web2templates.CampaignParticipantView{
				ID:             participant.ID,
				Name:           participant.Name,
				Role:           participant.Role,
				CampaignAccess: participant.CampaignAccess,
				Controller:     participant.Controller,
				AvatarURL:      participant.AvatarURL,
			})
		}
	}
	if route.kind == detailCharacters || route.kind == detailCharacter {
		characterItems, charactersErr := h.service.campaignCharacters(ctx, route.campaignID)
		if charactersErr != nil {
			h.writeError(w, r, charactersErr)
			return
		}
		characters = make([]web2templates.CampaignCharacterView, 0, len(characterItems))
		for _, character := range characterItems {
			characters = append(characters, web2templates.CampaignCharacterView{
				ID:         character.ID,
				Name:       character.Name,
				Kind:       character.Kind,
				Controller: character.Controller,
				AvatarURL:  character.AvatarURL,
			})
		}
	}
	if route.kind == detailSessions || route.kind == detailSession {
		sessionItems, sessionsErr := h.service.campaignSessions(ctx, route.campaignID)
		if sessionsErr != nil {
			h.writeError(w, r, sessionsErr)
			return
		}
		sessions = make([]web2templates.CampaignSessionView, 0, len(sessionItems))
		for _, session := range sessionItems {
			sessions = append(sessions, web2templates.CampaignSessionView{
				ID:        session.ID,
				Name:      session.Name,
				Status:    session.Status,
				StartedAt: session.StartedAt,
				UpdatedAt: session.UpdatedAt,
				EndedAt:   session.EndedAt,
			})
		}
	}
	if route.kind == detailInvites {
		inviteItems, invitesErr := h.service.campaignInvites(ctx, route.campaignID)
		if invitesErr != nil {
			h.writeError(w, r, invitesErr)
			return
		}
		invites = make([]web2templates.CampaignInviteView, 0, len(inviteItems))
		for _, invite := range inviteItems {
			invites = append(invites, web2templates.CampaignInviteView{
				ID:              invite.ID,
				ParticipantID:   invite.ParticipantID,
				RecipientUserID: invite.RecipientUserID,
				Status:          invite.Status,
			})
		}
	}
	layout := web2templates.AppMainLayoutOptions{
		SideMenu:  campaignWorkspaceMenu(route.campaignID, r.URL.Path, loc),
		MainStyle: campaignMainStyle(workspace.CoverImageURL),
		MainClass: campaignMainClass(workspace.CoverImageURL),
	}
	h.writeCampaignHTML(w, r, web2templates.T(loc, "game.campaign.title"), campaignDetailHeader(route, workspace.Name, loc), layout, web2templates.CampaignDetailFragment(web2templates.CampaignDetailView{
		Marker:       route.kind.marker(),
		CampaignID:   route.campaignID,
		SessionID:    route.sessionID,
		CharacterID:  route.characterID,
		Name:         workspace.Name,
		Theme:        workspace.Theme,
		System:       workspace.System,
		GMMode:       workspace.GMMode,
		Participants: participants,
		Characters:   characters,
		Sessions:     sessions,
		Invites:      invites,
	}, loc))
}

func (h handlers) handleMutation(w http.ResponseWriter, r *http.Request, route detailRoute) {
	ctx := webctx.WithResolvedUserID(r, h.deps.resolveUserID)
	var err error
	var redirect string
	switch route.kind {
	case detailSessionStart:
		err = h.service.startSession(ctx, route.campaignID)
		redirect = routepath.AppCampaignSessions(route.campaignID)
	case detailSessionEnd:
		err = h.service.endSession(ctx, route.campaignID)
		redirect = routepath.AppCampaignSessions(route.campaignID)
	case detailParticipantUpdate:
		err = h.service.updateParticipants(ctx, route.campaignID)
		redirect = routepath.AppCampaignParticipants(route.campaignID)
	case detailCharacterCreate:
		err = h.service.createCharacter(ctx, route.campaignID)
		redirect = routepath.AppCampaignCharacters(route.campaignID)
	case detailCharacterUpdate:
		err = h.service.updateCharacter(ctx, route.campaignID)
		redirect = routepath.AppCampaignCharacters(route.campaignID)
	case detailCharacterControl:
		err = h.service.controlCharacter(ctx, route.campaignID)
		redirect = routepath.AppCampaignCharacters(route.campaignID)
	case detailInviteCreate:
		err = h.service.createInvite(ctx, route.campaignID)
		redirect = routepath.AppCampaignInvites(route.campaignID)
	case detailInviteRevoke:
		err = h.service.revokeInvite(ctx, route.campaignID)
		redirect = routepath.AppCampaignInvites(route.campaignID)
	default:
		weberror.WriteAppError(w, r, http.StatusNotFound, h.deps.moduleDependencies())
		return
	}
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteRedirect(w, r, redirect)
}

func (h handlers) writeCampaignHTML(
	w http.ResponseWriter,
	r *http.Request,
	title string,
	header *web2templates.AppMainHeader,
	layout web2templates.AppMainLayoutOptions,
	fragment templ.Component,
) {
	if err := pagerender.WriteModulePage(w, r, h.deps.moduleDependencies(), pagerender.ModulePage{
		Title:    title,
		Header:   header,
		Layout:   layout,
		Fragment: fragment,
	}); err != nil {
		h.writeError(w, r, err)
	}
}

func (h handlers) writeCampaignChatHTML(
	w http.ResponseWriter,
	r *http.Request,
	view web2templates.CampaignChatView,
	lang string,
	loc web2templates.Localizer,
) {
	if httpx.IsHTMXRequest(r) {
		httpx.WriteHXRedirect(w, routepath.AppCampaignGame(view.CampaignID))
		return
	}
	if err := web2templates.CampaignChatPage(view, lang, loc).Render(r.Context(), w); err != nil {
		h.writeError(w, r, err)
	}
}

func (h handlers) pageLocalizer(w http.ResponseWriter, r *http.Request) (web2templates.Localizer, string) {
	loc, lang := web2i18n.ResolveLocalizer(w, r, h.deps.resolveLanguage)
	return loc, lang
}

func (h handlers) writeError(w http.ResponseWriter, r *http.Request, err error) {
	weberror.WriteModuleError(w, r, err, h.deps.moduleDependencies())
}

func parseAppGameSystem(value string) (commonv1.GameSystem, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "daggerheart", "game_system_daggerheart":
		return commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, true
	default:
		return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED, false
	}
}

func parseAppGmMode(value string) (statev1.GmMode, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "human":
		return statev1.GmMode_HUMAN, true
	case "ai":
		return statev1.GmMode_AI, true
	case "hybrid":
		return statev1.GmMode_HYBRID, true
	default:
		return statev1.GmMode_GM_MODE_UNSPECIFIED, false
	}
}
