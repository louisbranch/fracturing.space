package campaigns

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/pagerender"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/webctx"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/weberror"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
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

func campaignsListHeader(loc webtemplates.Localizer) *webtemplates.AppMainHeader {
	return &webtemplates.AppMainHeader{
		Title: webtemplates.T(loc, "game.campaigns.title"),
		Action: &webtemplates.AppMainHeaderAction{
			Label: webtemplates.T(loc, "game.campaigns.start_new"),
			URL:   routepath.AppCampaignsNew,
		},
	}
}

func campaignStartHeader(loc webtemplates.Localizer) *webtemplates.AppMainHeader {
	return &webtemplates.AppMainHeader{
		Title: webtemplates.T(loc, "game.campaigns.new.title"),
		Breadcrumbs: []sharedtemplates.BreadcrumbItem{
			{Label: webtemplates.T(loc, "game.campaigns.title"), URL: routepath.AppCampaigns},
			{Label: webtemplates.T(loc, "game.campaigns.new.title")},
		},
	}
}

func campaignCreateHeader(loc webtemplates.Localizer) *webtemplates.AppMainHeader {
	return &webtemplates.AppMainHeader{
		Title: webtemplates.T(loc, "game.create.title"),
		Breadcrumbs: []sharedtemplates.BreadcrumbItem{
			{Label: webtemplates.T(loc, "game.campaigns.title"), URL: routepath.AppCampaigns},
			{Label: webtemplates.T(loc, "game.create.title")},
		},
	}
}

func campaignDetailHeader(route detailRoute, campaignLabel string, loc webtemplates.Localizer) *webtemplates.AppMainHeader {
	return &webtemplates.AppMainHeader{
		Title:       webtemplates.T(loc, "game.campaign.title"),
		Breadcrumbs: campaignDetailBreadcrumbs(route, campaignLabel, loc),
	}
}

func campaignChatView(route detailRoute, workspace CampaignWorkspace, chatFallbackPort string) webtemplates.CampaignChatView {
	return webtemplates.CampaignChatView{
		CampaignID:       route.campaignID,
		CampaignName:     workspace.Name,
		BackURL:          routepath.AppCampaign(route.campaignID),
		ChatFallbackPort: strings.TrimSpace(chatFallbackPort),
	}
}

func campaignDetailBreadcrumbs(route detailRoute, campaignLabel string, loc webtemplates.Localizer) []sharedtemplates.BreadcrumbItem {
	campaignLabel = strings.TrimSpace(campaignLabel)
	if campaignLabel == "" {
		campaignLabel = route.campaignID
	}
	breadcrumbs := []sharedtemplates.BreadcrumbItem{
		{Label: webtemplates.T(loc, "game.campaigns.title"), URL: routepath.AppCampaigns},
		{Label: campaignLabel},
	}

	switch route.kind {
	case detailSessions:
		breadcrumbs[1].URL = routepath.AppCampaign(route.campaignID)
		return append(breadcrumbs, sharedtemplates.BreadcrumbItem{Label: webtemplates.T(loc, "game.sessions.title")})
	case detailSession:
		breadcrumbs[1].URL = routepath.AppCampaign(route.campaignID)
		return append(breadcrumbs,
			sharedtemplates.BreadcrumbItem{Label: webtemplates.T(loc, "game.sessions.title"), URL: routepath.AppCampaignSessions(route.campaignID)},
			sharedtemplates.BreadcrumbItem{Label: route.sessionID},
		)
	case detailParticipants:
		breadcrumbs[1].URL = routepath.AppCampaign(route.campaignID)
		return append(breadcrumbs, sharedtemplates.BreadcrumbItem{Label: webtemplates.T(loc, "game.participants.title")})
	case detailCharacters:
		breadcrumbs[1].URL = routepath.AppCampaign(route.campaignID)
		return append(breadcrumbs, sharedtemplates.BreadcrumbItem{Label: webtemplates.T(loc, "game.characters.title")})
	case detailCharacter:
		breadcrumbs[1].URL = routepath.AppCampaign(route.campaignID)
		return append(breadcrumbs,
			sharedtemplates.BreadcrumbItem{Label: webtemplates.T(loc, "game.characters.title"), URL: routepath.AppCampaignCharacters(route.campaignID)},
			sharedtemplates.BreadcrumbItem{Label: route.characterID},
		)
	case detailInvites:
		breadcrumbs[1].URL = routepath.AppCampaign(route.campaignID)
		return append(breadcrumbs, sharedtemplates.BreadcrumbItem{Label: webtemplates.T(loc, "game.campaign_invites.title")})
	default:
		return breadcrumbs
	}
}

func campaignWorkspaceMenu(campaignID string, currentPath string, loc webtemplates.Localizer) *webtemplates.AppSideMenu {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return nil
	}
	overviewURL := routepath.AppCampaign(campaignID)
	participantsURL := routepath.AppCampaignParticipants(campaignID)
	charactersURL := routepath.AppCampaignCharacters(campaignID)
	return &webtemplates.AppSideMenu{
		CurrentPath: strings.TrimSpace(currentPath),
		Items: []webtemplates.AppSideMenuItem{
			{
				Label:       webtemplates.T(loc, "game.campaign.menu.overview"),
				URL:         overviewURL,
				MatchPrefix: overviewURL,
				MatchExact:  true,
				IconID:      commonv1.IconId_ICON_ID_CAMPAIGN,
			},
			{
				Label:       webtemplates.T(loc, "game.participants.title"),
				URL:         participantsURL,
				MatchPrefix: participantsURL,
				IconID:      commonv1.IconId_ICON_ID_PARTICIPANT,
			},
			{
				Label:       webtemplates.T(loc, "game.characters.title"),
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
	// TODO(web-hardening): move cover image styling to template attributes or CSS classes so URL handling is not composed into inline style strings.
	safeCoverImageURL := strings.ReplaceAll(coverImageURL, "\"", "\\\"")
	return "background-image: url(\"" + safeCoverImageURL + "\"); background-size: cover; background-position: center; background-repeat: no-repeat;"
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

func (h handlers) routeCharacterID(r *http.Request) (string, bool) {
	characterID := strings.TrimSpace(r.PathValue("characterID"))
	if characterID == "" {
		return "", false
	}
	return characterID, true
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

func (h handlers) handleCharacterCreationStepRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.handleNotFound(w, r)
		return
	}
	characterID, ok := h.routeCharacterID(r)
	if !ok {
		h.handleNotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		h.writeError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.failed_to_parse_character_creation_form", "failed to parse character creation form"))
		return
	}

	ctx := webctx.WithResolvedUserID(r, h.deps.resolveUserID)
	progress, err := h.service.campaignCharacterCreationProgress(ctx, campaignID, characterID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if progress.Ready {
		h.writeError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_already_complete", "character creation workflow is already complete"))
		return
	}

	stepInput, err := daggerheartStepInputFromForm(r, progress.NextStep)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if err := h.service.applyCharacterCreationStep(ctx, campaignID, characterID, stepInput); err != nil {
		h.writeError(w, r, err)
		return
	}

	httpx.WriteRedirect(w, r, routepath.AppCampaignCharacter(campaignID, characterID))
}

func (h handlers) handleCharacterCreationResetRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.handleNotFound(w, r)
		return
	}
	characterID, ok := h.routeCharacterID(r)
	if !ok {
		h.handleNotFound(w, r)
		return
	}
	if err := h.service.resetCharacterCreationWorkflow(webctx.WithResolvedUserID(r, h.deps.resolveUserID), campaignID, characterID); err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteRedirect(w, r, routepath.AppCampaignCharacter(campaignID, characterID))
}

func (h handlers) handleCharacterCreateRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.handleNotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		h.writeError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.failed_to_parse_character_create_form", "failed to parse character create form"))
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		h.writeError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_name_is_required", "character name is required"))
		return
	}

	kindValue := strings.TrimSpace(r.FormValue("kind"))
	if kindValue == "" {
		kindValue = "pc"
	}
	kind, ok := parseAppCharacterKind(kindValue)
	if !ok {
		h.writeError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_kind_value_is_invalid", "character kind value is invalid"))
		return
	}

	created, err := h.service.createCharacter(webctx.WithResolvedUserID(r, h.deps.resolveUserID), campaignID, CreateCharacterInput{
		Name: name,
		Kind: kind,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	httpx.WriteRedirect(w, r, routepath.AppCampaignCharacter(campaignID, created.CharacterID))
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

func (h handlers) handleStartNewCampaign(w http.ResponseWriter, r *http.Request) {
	loc, _ := h.pageLocalizer(w, r)
	h.writeCampaignHTML(
		w,
		r,
		webtemplates.T(loc, "game.campaigns.new.title"),
		campaignStartHeader(loc),
		webtemplates.AppMainLayoutOptions{},
		webtemplates.CampaignStartFragment(loc),
	)
}

func (h handlers) handleCreateCampaign(w http.ResponseWriter, r *http.Request) {
	loc, _ := h.pageLocalizer(w, r)
	h.writeCampaignHTML(
		w,
		r,
		webtemplates.T(loc, "game.create.title"),
		campaignCreateHeader(loc),
		webtemplates.AppMainLayoutOptions{},
		webtemplates.CampaignCreateFragment(webtemplates.CampaignCreateFormValues{}, loc),
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
	resolvedLocale := platformi18n.LocaleForTag(webi18n.ResolveTag(r, h.deps.resolveLanguage))

	created, err := h.service.createCampaign(webctx.WithResolvedUserID(r, h.deps.resolveUserID), CreateCampaignInput{
		Name:        name,
		System:      system,
		GMMode:      gmMode,
		ThemePrompt: themePrompt,
		Locale:      resolvedLocale,
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
	list := make([]webtemplates.CampaignListItem, 0, len(items))
	for _, item := range items {
		list = append(list, webtemplates.CampaignListItem{
			ID:               item.ID,
			Name:             item.Name,
			Theme:            item.Theme,
			CoverImageURL:    item.CoverImageURL,
			ParticipantCount: item.ParticipantCount,
			CharacterCount:   item.CharacterCount,
		})
	}
	h.writeCampaignHTML(w, r, webtemplates.T(loc, "game.campaigns.title"), campaignsListHeader(loc), webtemplates.AppMainLayoutOptions{}, webtemplates.CampaignListFragment(list, loc))
}

func (h handlers) handleDetail(w http.ResponseWriter, r *http.Request, route detailRoute) {
	loc, lang := h.pageLocalizer(w, r)
	resolvedLocale := platformi18n.LocaleForTag(webi18n.ResolveTag(r, h.deps.resolveLanguage))
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
	participants := []webtemplates.CampaignParticipantView{}
	characters := []webtemplates.CampaignCharacterView{}
	sessions := []webtemplates.CampaignSessionView{}
	invites := []webtemplates.CampaignInviteView{}
	characterCreation := webtemplates.CampaignCharacterCreationView{}
	characterCreationEnabled := false
	if route.kind == detailParticipants {
		participantItems, participantsErr := h.service.campaignParticipants(ctx, route.campaignID)
		if participantsErr != nil {
			h.writeError(w, r, participantsErr)
			return
		}
		participants = make([]webtemplates.CampaignParticipantView, 0, len(participantItems))
		for _, participant := range participantItems {
			participants = append(participants, webtemplates.CampaignParticipantView{
				ID:             participant.ID,
				Name:           participant.Name,
				Role:           participant.Role,
				CampaignAccess: participant.CampaignAccess,
				Controller:     participant.Controller,
				Pronouns:       participant.Pronouns,
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
		characters = make([]webtemplates.CampaignCharacterView, 0, len(characterItems))
		for _, character := range characterItems {
			characters = append(characters, webtemplates.CampaignCharacterView{
				ID:             character.ID,
				Name:           character.Name,
				Kind:           character.Kind,
				Controller:     character.Controller,
				Pronouns:       character.Pronouns,
				Aliases:        append([]string(nil), character.Aliases...),
				AvatarURL:      character.AvatarURL,
				CanEdit:        character.CanEdit,
				EditReasonCode: character.EditReasonCode,
			})
		}
	}
	if route.kind == detailCharacter {
		characterCreationEnabled = isDaggerheartCampaignSystem(workspace.System)
		if characterCreationEnabled {
			creation, creationErr := h.service.campaignCharacterCreation(ctx, route.campaignID, route.characterID, resolvedLocale)
			if creationErr != nil {
				h.writeError(w, r, creationErr)
				return
			}
			characterCreation = campaignCharacterCreationView(creation)
		}
	}
	if route.kind == detailSessions || route.kind == detailSession {
		sessionItems, sessionsErr := h.service.campaignSessions(ctx, route.campaignID)
		if sessionsErr != nil {
			h.writeError(w, r, sessionsErr)
			return
		}
		sessions = make([]webtemplates.CampaignSessionView, 0, len(sessionItems))
		for _, session := range sessionItems {
			sessions = append(sessions, webtemplates.CampaignSessionView{
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
		invites = make([]webtemplates.CampaignInviteView, 0, len(inviteItems))
		for _, invite := range inviteItems {
			invites = append(invites, webtemplates.CampaignInviteView{
				ID:              invite.ID,
				ParticipantID:   invite.ParticipantID,
				RecipientUserID: invite.RecipientUserID,
				Status:          invite.Status,
			})
		}
	}
	layout := webtemplates.AppMainLayoutOptions{
		SideMenu:  campaignWorkspaceMenu(route.campaignID, r.URL.Path, loc),
		MainStyle: campaignMainStyle(workspace.CoverImageURL),
		MainClass: campaignMainClass(workspace.CoverImageURL),
	}
	h.writeCampaignHTML(w, r, webtemplates.T(loc, "game.campaign.title"), campaignDetailHeader(route, workspace.Name, loc), layout, webtemplates.CampaignDetailFragment(webtemplates.CampaignDetailView{
		Marker:                   route.kind.marker(),
		CampaignID:               route.campaignID,
		SessionID:                route.sessionID,
		CharacterID:              route.characterID,
		Name:                     workspace.Name,
		Theme:                    workspace.Theme,
		System:                   workspace.System,
		GMMode:                   workspace.GMMode,
		Participants:             participants,
		Characters:               characters,
		Sessions:                 sessions,
		Invites:                  invites,
		CharacterCreationEnabled: characterCreationEnabled,
		CharacterCreation:        characterCreation,
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
	header *webtemplates.AppMainHeader,
	layout webtemplates.AppMainLayoutOptions,
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
	view webtemplates.CampaignChatView,
	lang string,
	loc webtemplates.Localizer,
) {
	if httpx.IsHTMXRequest(r) {
		httpx.WriteHXRedirect(w, routepath.AppCampaignGame(view.CampaignID))
		return
	}
	if err := webtemplates.CampaignChatPage(view, lang, loc).Render(r.Context(), w); err != nil {
		h.writeError(w, r, err)
	}
}

func (h handlers) pageLocalizer(w http.ResponseWriter, r *http.Request) (webtemplates.Localizer, string) {
	loc, lang := webi18n.ResolveLocalizer(w, r, h.deps.resolveLanguage)
	return loc, lang
}

func (h handlers) writeError(w http.ResponseWriter, r *http.Request, err error) {
	weberror.WriteModuleError(w, r, err, h.deps.moduleDependencies())
}

func campaignCharacterCreationView(creation CampaignCharacterCreation) webtemplates.CampaignCharacterCreationView {
	view := webtemplates.CampaignCharacterCreationView{
		Ready:              creation.Progress.Ready,
		NextStep:           creation.Progress.NextStep,
		UnmetReasons:       append([]string(nil), creation.Progress.UnmetReasons...),
		ClassID:            strings.TrimSpace(creation.Profile.ClassID),
		SubclassID:         strings.TrimSpace(creation.Profile.SubclassID),
		AncestryID:         strings.TrimSpace(creation.Profile.AncestryID),
		CommunityID:        strings.TrimSpace(creation.Profile.CommunityID),
		Agility:            strings.TrimSpace(creation.Profile.Agility),
		Strength:           strings.TrimSpace(creation.Profile.Strength),
		Finesse:            strings.TrimSpace(creation.Profile.Finesse),
		Instinct:           strings.TrimSpace(creation.Profile.Instinct),
		Presence:           strings.TrimSpace(creation.Profile.Presence),
		Knowledge:          strings.TrimSpace(creation.Profile.Knowledge),
		PrimaryWeaponID:    strings.TrimSpace(creation.Profile.PrimaryWeaponID),
		SecondaryWeaponID:  strings.TrimSpace(creation.Profile.SecondaryWeaponID),
		ArmorID:            strings.TrimSpace(creation.Profile.ArmorID),
		PotionItemID:       strings.TrimSpace(creation.Profile.PotionItemID),
		Background:         strings.TrimSpace(creation.Profile.Background),
		ExperienceName:     strings.TrimSpace(creation.Profile.ExperienceName),
		ExperienceModifier: strings.TrimSpace(creation.Profile.ExperienceModifier),
		DomainCardIDs:      append([]string(nil), creation.Profile.DomainCardIDs...),
		Connections:        strings.TrimSpace(creation.Profile.Connections),
		Steps:              []webtemplates.CampaignCharacterCreationStepView{},
		Classes:            []webtemplates.CampaignCreationClassView{},
		Subclasses:         []webtemplates.CampaignCreationSubclassView{},
		Ancestries:         []webtemplates.CampaignCreationHeritageView{},
		Communities:        []webtemplates.CampaignCreationHeritageView{},
		PrimaryWeapons:     []webtemplates.CampaignCreationWeaponView{},
		SecondaryWeapons:   []webtemplates.CampaignCreationWeaponView{},
		Armor:              []webtemplates.CampaignCreationArmorView{},
		PotionItems:        []webtemplates.CampaignCreationItemView{},
		DomainCards:        []webtemplates.CampaignCreationDomainCardView{},
	}
	for _, step := range creation.Progress.Steps {
		view.Steps = append(view.Steps, webtemplates.CampaignCharacterCreationStepView{
			Step:     step.Step,
			Key:      strings.TrimSpace(step.Key),
			Complete: step.Complete,
		})
	}
	for _, class := range creation.Classes {
		view.Classes = append(view.Classes, webtemplates.CampaignCreationClassView{
			ID:   strings.TrimSpace(class.ID),
			Name: strings.TrimSpace(class.Name),
		})
	}
	for _, subclass := range creation.Subclasses {
		view.Subclasses = append(view.Subclasses, webtemplates.CampaignCreationSubclassView{
			ID:      strings.TrimSpace(subclass.ID),
			Name:    strings.TrimSpace(subclass.Name),
			ClassID: strings.TrimSpace(subclass.ClassID),
		})
	}
	for _, ancestry := range creation.Ancestries {
		view.Ancestries = append(view.Ancestries, webtemplates.CampaignCreationHeritageView{
			ID:   strings.TrimSpace(ancestry.ID),
			Name: strings.TrimSpace(ancestry.Name),
		})
	}
	for _, community := range creation.Communities {
		view.Communities = append(view.Communities, webtemplates.CampaignCreationHeritageView{
			ID:   strings.TrimSpace(community.ID),
			Name: strings.TrimSpace(community.Name),
		})
	}
	for _, weapon := range creation.PrimaryWeapons {
		view.PrimaryWeapons = append(view.PrimaryWeapons, webtemplates.CampaignCreationWeaponView{
			ID:   strings.TrimSpace(weapon.ID),
			Name: strings.TrimSpace(weapon.Name),
		})
	}
	for _, weapon := range creation.SecondaryWeapons {
		view.SecondaryWeapons = append(view.SecondaryWeapons, webtemplates.CampaignCreationWeaponView{
			ID:   strings.TrimSpace(weapon.ID),
			Name: strings.TrimSpace(weapon.Name),
		})
	}
	for _, armor := range creation.Armor {
		view.Armor = append(view.Armor, webtemplates.CampaignCreationArmorView{
			ID:   strings.TrimSpace(armor.ID),
			Name: strings.TrimSpace(armor.Name),
		})
	}
	for _, item := range creation.PotionItems {
		view.PotionItems = append(view.PotionItems, webtemplates.CampaignCreationItemView{
			ID:   strings.TrimSpace(item.ID),
			Name: strings.TrimSpace(item.Name),
		})
	}
	for _, card := range creation.DomainCards {
		view.DomainCards = append(view.DomainCards, webtemplates.CampaignCreationDomainCardView{
			ID:       strings.TrimSpace(card.ID),
			Name:     strings.TrimSpace(card.Name),
			DomainID: strings.TrimSpace(card.DomainID),
			Level:    card.Level,
		})
	}
	return view
}

func daggerheartStepInputFromForm(r *http.Request, nextStep int32) (*daggerheartv1.DaggerheartCreationStepInput, error) {
	switch nextStep {
	case 1:
		classID := strings.TrimSpace(r.FormValue("class_id"))
		subclassID := strings.TrimSpace(r.FormValue("subclass_id"))
		if classID == "" || subclassID == "" {
			return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_class_and_subclass_are_required", "class and subclass are required")
		}
		return &daggerheartv1.DaggerheartCreationStepInput{Step: &daggerheartv1.DaggerheartCreationStepInput_ClassSubclassInput{ClassSubclassInput: &daggerheartv1.DaggerheartCreationStepClassSubclassInput{ClassId: classID, SubclassId: subclassID}}}, nil
	case 2:
		ancestryID := strings.TrimSpace(r.FormValue("ancestry_id"))
		communityID := strings.TrimSpace(r.FormValue("community_id"))
		if ancestryID == "" || communityID == "" {
			return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_ancestry_and_community_are_required", "ancestry and community are required")
		}
		return &daggerheartv1.DaggerheartCreationStepInput{Step: &daggerheartv1.DaggerheartCreationStepInput_HeritageInput{HeritageInput: &daggerheartv1.DaggerheartCreationStepHeritageInput{AncestryId: ancestryID, CommunityId: communityID}}}, nil
	case 3:
		agility, err := parseRequiredInt32(r.FormValue("agility"), "agility")
		if err != nil {
			return nil, err
		}
		strength, err := parseRequiredInt32(r.FormValue("strength"), "strength")
		if err != nil {
			return nil, err
		}
		finesse, err := parseRequiredInt32(r.FormValue("finesse"), "finesse")
		if err != nil {
			return nil, err
		}
		instinct, err := parseRequiredInt32(r.FormValue("instinct"), "instinct")
		if err != nil {
			return nil, err
		}
		presence, err := parseRequiredInt32(r.FormValue("presence"), "presence")
		if err != nil {
			return nil, err
		}
		knowledge, err := parseRequiredInt32(r.FormValue("knowledge"), "knowledge")
		if err != nil {
			return nil, err
		}
		return &daggerheartv1.DaggerheartCreationStepInput{Step: &daggerheartv1.DaggerheartCreationStepInput_TraitsInput{TraitsInput: &daggerheartv1.DaggerheartCreationStepTraitsInput{Agility: agility, Strength: strength, Finesse: finesse, Instinct: instinct, Presence: presence, Knowledge: knowledge}}}, nil
	case 4:
		return &daggerheartv1.DaggerheartCreationStepInput{Step: &daggerheartv1.DaggerheartCreationStepInput_DetailsInput{DetailsInput: &daggerheartv1.DaggerheartCreationStepDetailsInput{}}}, nil
	case 5:
		primaryWeaponID := strings.TrimSpace(r.FormValue("weapon_primary_id"))
		secondaryWeaponID := strings.TrimSpace(r.FormValue("weapon_secondary_id"))
		armorID := strings.TrimSpace(r.FormValue("armor_id"))
		potionItemID := strings.TrimSpace(r.FormValue("potion_item_id"))
		if primaryWeaponID == "" || armorID == "" || potionItemID == "" {
			return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_primary_weapon_armor_and_potion_are_required", "primary weapon, armor, and potion are required")
		}
		weaponIDs := []string{primaryWeaponID}
		if secondaryWeaponID != "" {
			weaponIDs = append(weaponIDs, secondaryWeaponID)
		}
		return &daggerheartv1.DaggerheartCreationStepInput{Step: &daggerheartv1.DaggerheartCreationStepInput_EquipmentInput{EquipmentInput: &daggerheartv1.DaggerheartCreationStepEquipmentInput{WeaponIds: weaponIDs, ArmorId: armorID, PotionItemId: potionItemID}}}, nil
	case 6:
		background := strings.TrimSpace(r.FormValue("background"))
		if background == "" {
			return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_background_is_required", "background is required")
		}
		return &daggerheartv1.DaggerheartCreationStepInput{Step: &daggerheartv1.DaggerheartCreationStepInput_BackgroundInput{BackgroundInput: &daggerheartv1.DaggerheartCreationStepBackgroundInput{Background: background}}}, nil
	case 7:
		experienceName := strings.TrimSpace(r.FormValue("experience_name"))
		if experienceName == "" {
			return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_experience_name_is_required", "experience name is required")
		}
		experienceModifier, err := parseOptionalInt32(r.FormValue("experience_modifier"))
		if err != nil {
			return nil, err
		}
		return &daggerheartv1.DaggerheartCreationStepInput{Step: &daggerheartv1.DaggerheartCreationStepInput_ExperiencesInput{ExperiencesInput: &daggerheartv1.DaggerheartCreationStepExperiencesInput{Experiences: []*daggerheartv1.DaggerheartExperience{{Name: experienceName, Modifier: experienceModifier}}}}}, nil
	case 8:
		rawDomainCardIDs := r.Form["domain_card_id"]
		domainCardIDs := make([]string, 0, len(rawDomainCardIDs))
		seen := map[string]struct{}{}
		for _, rawDomainCardID := range rawDomainCardIDs {
			trimmedDomainCardID := strings.TrimSpace(rawDomainCardID)
			if trimmedDomainCardID == "" {
				continue
			}
			if _, ok := seen[trimmedDomainCardID]; ok {
				continue
			}
			seen[trimmedDomainCardID] = struct{}{}
			domainCardIDs = append(domainCardIDs, trimmedDomainCardID)
		}
		if len(domainCardIDs) == 0 {
			return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_at_least_one_domain_card_is_required", "at least one domain card is required")
		}
		return &daggerheartv1.DaggerheartCreationStepInput{Step: &daggerheartv1.DaggerheartCreationStepInput_DomainCardsInput{DomainCardsInput: &daggerheartv1.DaggerheartCreationStepDomainCardsInput{DomainCardIds: domainCardIDs}}}, nil
	case 9:
		connections := strings.TrimSpace(r.FormValue("connections"))
		if connections == "" {
			return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_connections_are_required", "connections are required")
		}
		return &daggerheartv1.DaggerheartCreationStepInput{Step: &daggerheartv1.DaggerheartCreationStepInput_ConnectionsInput{ConnectionsInput: &daggerheartv1.DaggerheartCreationStepConnectionsInput{Connections: connections}}}, nil
	default:
		return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_step_is_not_available", "character creation step is not available")
	}
}

func parseRequiredInt32(raw string, field string) (int32, error) {
	trimmedRaw := strings.TrimSpace(raw)
	if trimmedRaw == "" {
		return 0, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_numeric_field_is_required", field+" is required")
	}
	value, err := strconv.Atoi(trimmedRaw)
	if err != nil {
		return 0, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_numeric_field_must_be_valid_integer", field+" must be a valid integer")
	}
	return int32(value), nil
}

func parseOptionalInt32(raw string) (int32, error) {
	trimmedRaw := strings.TrimSpace(raw)
	if trimmedRaw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(trimmedRaw)
	if err != nil {
		return 0, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_modifier_must_be_valid_integer", "modifier must be a valid integer")
	}
	return int32(value), nil
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

func parseAppCharacterKind(value string) (statev1.CharacterKind, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "pc", "character_kind_pc":
		return statev1.CharacterKind_PC, true
	case "npc", "character_kind_npc":
		return statev1.CharacterKind_NPC, true
	default:
		return statev1.CharacterKind_CHARACTER_KIND_UNSPECIFIED, false
	}
}

func isDaggerheartCampaignSystem(system string) bool {
	return strings.EqualFold(strings.TrimSpace(system), "Daggerheart")
}
