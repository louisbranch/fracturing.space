package campaigns

import (
	"context"
	"net/http"
	"strings"

	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"golang.org/x/text/language"
)

// Detail view markers. The template uses these to select which section to render.
const (
	markerOverview        = "campaign-overview"
	markerSessions        = "campaign-sessions"
	markerSessionDetail   = "campaign-session-detail"
	markerParticipants    = "campaign-participants"
	markerCharacters      = "campaign-characters"
	markerCharacterDetail = "campaign-character-detail"
	markerInvites         = "campaign-invites"
)

// --- Breadcrumbs and layout ---

// campaignBreadcrumbs builds breadcrumbs for a campaign page. When extra
// items are provided, the campaign label becomes a link to the campaign
// overview page.
func campaignBreadcrumbs(campaignID, campaignLabel string, loc webtemplates.Localizer, extra ...sharedtemplates.BreadcrumbItem) []sharedtemplates.BreadcrumbItem {
	campaignLabel = strings.TrimSpace(campaignLabel)
	if campaignLabel == "" {
		campaignLabel = campaignID
	}
	campaign := sharedtemplates.BreadcrumbItem{Label: campaignLabel}
	if len(extra) > 0 {
		campaign.URL = routepath.AppCampaign(campaignID)
	}
	result := []sharedtemplates.BreadcrumbItem{
		{Label: webtemplates.T(loc, "game.campaigns.title"), URL: routepath.AppCampaigns},
		campaign,
	}
	return append(result, extra...)
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

// --- Campaign page context ---

// campaignPageContext holds the shared state loaded for any campaign detail page.
type campaignPageContext struct {
	workspace CampaignWorkspace
	loc       webtemplates.Localizer
	lang      string
	locale    language.Tag
}

func (h handlers) loadCampaignPage(w http.ResponseWriter, r *http.Request, campaignID string) (context.Context, *campaignPageContext, error) {
	loc, lang := h.PageLocalizer(w, r)
	ctx, _ := h.RequestContextAndUserID(r)
	workspace, err := h.service.campaignWorkspace(ctx, campaignID)
	if err != nil {
		return nil, nil, err
	}
	return ctx, &campaignPageContext{
		workspace: workspace,
		loc:       loc,
		lang:      lang,
		locale:    h.RequestLocaleTag(r),
	}, nil
}

func (p *campaignPageContext) layout(campaignID, currentPath string) webtemplates.AppMainLayoutOptions {
	return webtemplates.AppMainLayoutOptions{
		SideMenu:  campaignWorkspaceMenu(p.workspace, currentPath, p.loc),
		MainStyle: campaignMainStyle(p.workspace.CoverImageURL),
		MainClass: campaignMainClass(p.workspace.CoverImageURL),
		Metadata: webtemplates.AppMainLayoutMetadata{
			RouteArea: webtemplates.RouteAreaCampaignWorkspace,
		},
	}
}

// detailView returns a CampaignDetailView pre-filled with workspace fields.
// Callers set sub-page-specific fields on the returned value before rendering.
func (p *campaignPageContext) detailView(campaignID, marker string) webtemplates.CampaignDetailView {
	return webtemplates.CampaignDetailView{
		Marker:       marker,
		CampaignID:   campaignID,
		Name:         p.workspace.Name,
		Theme:        p.workspace.Theme,
		System:       p.workspace.System,
		GMMode:       p.workspace.GMMode,
		Status:       p.workspace.Status,
		Locale:       p.workspace.Locale,
		Intent:       p.workspace.Intent,
		AccessPolicy: p.workspace.AccessPolicy,
	}
}

func (p *campaignPageContext) title(campaignID string) string {
	name := strings.TrimSpace(p.workspace.Name)
	if name != "" {
		return name
	}
	if id := strings.TrimSpace(p.workspace.ID); id != "" {
		return id
	}
	if id := strings.TrimSpace(campaignID); id != "" {
		return id
	}
	return webtemplates.T(p.loc, "game.campaign.title")
}

// header returns a campaign detail page header with the given breadcrumbs.
func (p *campaignPageContext) header(campaignID string, breadcrumbs []sharedtemplates.BreadcrumbItem) *webtemplates.AppMainHeader {
	return &webtemplates.AppMainHeader{
		Title:       p.title(campaignID),
		Breadcrumbs: breadcrumbs,
	}
}

// --- Route param extractors ---

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

// withCampaignID extracts the campaign ID path param and delegates to fn,
// returning 404 when the param is missing.
func (h handlers) withCampaignID(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		campaignID, ok := h.routeCampaignID(r)
		if !ok {
			h.WriteNotFound(w, r)
			return
		}
		fn(w, r, campaignID)
	}
}

// --- Campaign detail route handlers ---

func (h handlers) handleOverviewMethodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	httpx.MethodNotAllowed(http.MethodGet+", HEAD")(w, nil)
}

func (h handlers) handleCharacterDetailRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.WriteNotFound(w, r)
		return
	}
	characterID, ok := h.routeCharacterID(r)
	if !ok {
		h.WriteNotFound(w, r)
		return
	}
	h.handleCharacterDetail(w, r, campaignID, characterID)
}

func (h handlers) handleSessionDetailRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.WriteNotFound(w, r)
		return
	}
	sessionID := strings.TrimSpace(r.PathValue("sessionID"))
	if sessionID == "" {
		h.WriteNotFound(w, r)
		return
	}
	h.handleSessionDetail(w, r, campaignID, sessionID)
}

// --- Campaign detail scaffold ---

// campaignDetailSpec describes one campaign sub-page. The scaffold loads the
// campaign workspace, calls loadData to populate view-specific fields, builds
// breadcrumbs, and renders the detail fragment.
type campaignDetailSpec struct {
	marker   string
	extra    func(loc webtemplates.Localizer) []sharedtemplates.BreadcrumbItem
	loadData func(ctx context.Context, campaignID string, page *campaignPageContext, view *webtemplates.CampaignDetailView) error
}

func (h handlers) renderCampaignDetail(w http.ResponseWriter, r *http.Request, campaignID string, spec campaignDetailSpec) {
	ctx, page, err := h.loadCampaignPage(w, r, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := page.detailView(campaignID, spec.marker)
	if spec.loadData != nil {
		if err := spec.loadData(ctx, campaignID, page, &view); err != nil {
			h.WriteError(w, r, err)
			return
		}
	}
	var crumbs []sharedtemplates.BreadcrumbItem
	if spec.extra != nil {
		crumbs = campaignBreadcrumbs(campaignID, page.workspace.Name, page.loc, spec.extra(page.loc)...)
	} else {
		crumbs = campaignBreadcrumbs(campaignID, page.workspace.Name, page.loc)
	}
	h.WritePage(w, r, page.title(campaignID), http.StatusOK,
		page.header(campaignID, crumbs),
		page.layout(campaignID, r.URL.Path),
		webtemplates.CampaignDetailFragment(view, page.loc))
}

// --- Per-sub-page detail handlers ---

func (h handlers) handleOverview(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.renderCampaignDetail(w, r, campaignID, campaignDetailSpec{marker: markerOverview})
}

func (h handlers) handleParticipants(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.renderCampaignDetail(w, r, campaignID, campaignDetailSpec{
		marker: markerParticipants,
		extra: func(loc webtemplates.Localizer) []sharedtemplates.BreadcrumbItem {
			return []sharedtemplates.BreadcrumbItem{{Label: webtemplates.T(loc, "game.participants.title")}}
		},
		loadData: func(ctx context.Context, campaignID string, _ *campaignPageContext, view *webtemplates.CampaignDetailView) error {
			items, err := h.service.campaignParticipants(ctx, campaignID)
			if err != nil {
				return err
			}
			view.Participants = mapParticipantsView(items)
			return nil
		},
	})
}

func (h handlers) handleCharacters(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.renderCampaignDetail(w, r, campaignID, campaignDetailSpec{
		marker: markerCharacters,
		extra: func(loc webtemplates.Localizer) []sharedtemplates.BreadcrumbItem {
			return []sharedtemplates.BreadcrumbItem{{Label: webtemplates.T(loc, "game.characters.title")}}
		},
		loadData: func(ctx context.Context, campaignID string, _ *campaignPageContext, view *webtemplates.CampaignDetailView) error {
			items, err := h.service.campaignCharacters(ctx, campaignID)
			if err != nil {
				return err
			}
			view.Characters = mapCharactersView(items)
			return nil
		},
	})
}

func (h handlers) handleCharacterDetail(w http.ResponseWriter, r *http.Request, campaignID, characterID string) {
	h.renderCampaignDetail(w, r, campaignID, campaignDetailSpec{
		marker: markerCharacterDetail,
		extra: func(loc webtemplates.Localizer) []sharedtemplates.BreadcrumbItem {
			return []sharedtemplates.BreadcrumbItem{
				{Label: webtemplates.T(loc, "game.characters.title"), URL: routepath.AppCampaignCharacters(campaignID)},
				{Label: characterID},
			}
		},
		loadData: func(ctx context.Context, campaignID string, page *campaignPageContext, view *webtemplates.CampaignDetailView) error {
			characterItems, err := h.service.campaignCharacters(ctx, campaignID)
			if err != nil {
				return err
			}
			view.CharacterID = characterID
			view.Characters = mapCharactersView(characterItems)
			workflow := h.service.resolveWorkflow(page.workspace.System)
			view.CharacterCreationEnabled = workflow != nil
			if view.CharacterCreationEnabled {
				creation, err := h.service.campaignCharacterCreation(ctx, campaignID, characterID, page.locale, workflow)
				if err != nil {
					return err
				}
				view.CharacterCreation = workflow.CreationView(creation)
			}
			return nil
		},
	})
}

func (h handlers) handleSessions(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.renderCampaignDetail(w, r, campaignID, campaignDetailSpec{
		marker: markerSessions,
		extra: func(loc webtemplates.Localizer) []sharedtemplates.BreadcrumbItem {
			return []sharedtemplates.BreadcrumbItem{{Label: webtemplates.T(loc, "game.sessions.title")}}
		},
		loadData: func(ctx context.Context, campaignID string, _ *campaignPageContext, view *webtemplates.CampaignDetailView) error {
			items, err := h.service.campaignSessions(ctx, campaignID)
			if err != nil {
				return err
			}
			view.Sessions = mapSessionsView(items)
			return nil
		},
	})
}

func (h handlers) handleSessionDetail(w http.ResponseWriter, r *http.Request, campaignID, sessionID string) {
	h.renderCampaignDetail(w, r, campaignID, campaignDetailSpec{
		marker: markerSessionDetail,
		extra: func(loc webtemplates.Localizer) []sharedtemplates.BreadcrumbItem {
			return []sharedtemplates.BreadcrumbItem{
				{Label: webtemplates.T(loc, "game.sessions.title"), URL: routepath.AppCampaignSessions(campaignID)},
				{Label: sessionID},
			}
		},
		loadData: func(ctx context.Context, campaignID string, _ *campaignPageContext, view *webtemplates.CampaignDetailView) error {
			items, err := h.service.campaignSessions(ctx, campaignID)
			if err != nil {
				return err
			}
			view.SessionID = sessionID
			view.Sessions = mapSessionsView(items)
			return nil
		},
	})
}

func (h handlers) handleInvites(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.renderCampaignDetail(w, r, campaignID, campaignDetailSpec{
		marker: markerInvites,
		extra: func(loc webtemplates.Localizer) []sharedtemplates.BreadcrumbItem {
			return []sharedtemplates.BreadcrumbItem{{Label: webtemplates.T(loc, "game.campaign_invites.title")}}
		},
		loadData: func(ctx context.Context, campaignID string, _ *campaignPageContext, view *webtemplates.CampaignDetailView) error {
			items, err := h.service.campaignInvites(ctx, campaignID)
			if err != nil {
				return err
			}
			view.Invites = mapInvitesView(items)
			return nil
		},
	})
}
