package campaigns

import (
	"context"
	"net/http"
	"strings"

	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/routeparam"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"golang.org/x/text/language"
)

// Detail view markers. The template uses these to select which section to render.
const (
	markerOverview          = "campaign-overview"
	markerCampaignEdit      = "campaign-edit"
	markerSessions          = "campaign-sessions"
	markerSessionDetail     = "campaign-session-detail"
	markerParticipants      = "campaign-participants"
	markerParticipantCreate = "campaign-participant-create"
	markerParticipantEdit   = "campaign-participant-edit"
	markerCharacters        = "campaign-characters"
	markerCharacterCreate   = "campaign-character-create"
	markerCharacterEdit     = "campaign-character-edit"
	markerCharacterDetail   = "campaign-character-detail"
	markerInvites           = "campaign-invites"
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

// campaignMainClass centralizes this web behavior in one helper seam.
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
	workspace        campaignapp.CampaignWorkspace
	sessions         []campaignapp.CampaignSession
	canManageInvites bool
	loc              webtemplates.Localizer
	lang             string
	locale           language.Tag
}

// loadCampaignPage loads the package state needed for this request path.
func (h handlers) loadCampaignPage(w http.ResponseWriter, r *http.Request, campaignID string) (context.Context, *campaignPageContext, error) {
	loc, lang := h.PageLocalizer(w, r)
	ctx, _ := h.RequestContextAndUserID(r)
	workspace, err := h.service.CampaignWorkspace(ctx, campaignID)
	if err != nil {
		return nil, nil, err
	}
	sessions, err := h.service.CampaignSessions(ctx, campaignID)
	if err != nil {
		return nil, nil, err
	}
	canManageInvites := h.service.RequireManageInvites(ctx, campaignID) == nil
	return ctx, &campaignPageContext{
		workspace:        workspace,
		sessions:         sessions,
		canManageInvites: canManageInvites,
		loc:              loc,
		lang:             lang,
		locale:           h.RequestLocaleTag(r),
	}, nil
}

// layout centralizes this web behavior in one helper seam.
func (p *campaignPageContext) layout(campaignID, currentPath string) webtemplates.AppMainLayoutOptions {
	return webtemplates.AppMainLayoutOptions{
		SideMenu: campaignWorkspaceMenu(p.workspace, currentPath, p.sessions, p.canManageInvites, p.loc),
		MainBackground: &webtemplates.AppBackgroundImage{
			PreviewURL: strings.TrimSpace(p.workspace.CoverPreviewURL),
			FullURL:    strings.TrimSpace(p.workspace.CoverImageURL),
		},
		MainClass: campaignMainClass(p.workspace.CoverImageURL),
		Metadata: webtemplates.AppMainLayoutMetadata{
			RouteArea: webtemplates.RouteAreaCampaignWorkspace,
		},
	}
}

// detailView returns a module-owned detail render model pre-filled with
// workspace fields. Callers set sub-page-specific fields before rendering.
func (p *campaignPageContext) detailView(campaignID, marker string) campaignrender.DetailView {
	return campaignrender.DetailView{
		Marker:           marker,
		CampaignID:       campaignID,
		Name:             p.workspace.Name,
		Theme:            p.workspace.Theme,
		System:           p.workspace.System,
		GMMode:           p.workspace.GMMode,
		Status:           p.workspace.Status,
		Locale:           p.workspace.Locale,
		LocaleValue:      campaignWorkspaceLocaleFormValue(p.workspace.Locale),
		Intent:           p.workspace.Intent,
		AccessPolicy:     p.workspace.AccessPolicy,
		ActionsLocked:    p.outOfGameActionsLocked(),
		CanManageInvites: p.canManageInvites,
	}
}

// outOfGameActionsLocked reports whether session state should disable campaign
// metadata, participant, invite, or character UI actions.
func (p *campaignPageContext) outOfGameActionsLocked() bool {
	for _, session := range p.sessions {
		if campaignSessionMenuIsActive(session) {
			return true
		}
	}
	return false
}

// title centralizes this web behavior in one helper seam.
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

// routeCampaignID extracts the canonical campaign route parameter.
func (h handlers) routeCampaignID(r *http.Request) (string, bool) {
	return routeparam.Read(r, "campaignID")
}

// routeCharacterID centralizes this web behavior in one helper seam.
func (h handlers) routeCharacterID(r *http.Request) (string, bool) {
	return routeparam.Read(r, "characterID")
}

// routeParticipantID centralizes this web behavior in one helper seam.
func (h handlers) routeParticipantID(r *http.Request) (string, bool) {
	return routeparam.Read(r, "participantID")
}

// routeSessionID centralizes this web behavior in one helper seam.
func (h handlers) routeSessionID(r *http.Request) (string, bool) {
	return routeparam.Read(r, "sessionID")
}

// parseFormOrWriteError parses form data and writes a localized invalid-input
// error response when parsing fails.
func (h handlers) parseFormOrWriteError(w http.ResponseWriter, r *http.Request, localizationKey string, message string) bool {
	if err := r.ParseForm(); err != nil {
		h.WriteError(w, r, apperrors.EK(apperrors.KindInvalidInput, localizationKey, message))
		return false
	}
	return true
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

// withCampaignAndParticipantID extracts campaign/participant IDs and delegates
// to fn, returning 404 when either route parameter is missing.
func (h handlers) withCampaignAndParticipantID(fn func(http.ResponseWriter, *http.Request, string, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		campaignID, ok := h.routeCampaignID(r)
		if !ok {
			h.WriteNotFound(w, r)
			return
		}
		participantID, ok := h.routeParticipantID(r)
		if !ok {
			h.WriteNotFound(w, r)
			return
		}
		fn(w, r, campaignID, participantID)
	}
}

// withCampaignAndCharacterID extracts campaign/character IDs and delegates to
// fn, returning 404 when either route parameter is missing.
func (h handlers) withCampaignAndCharacterID(fn func(http.ResponseWriter, *http.Request, string, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		fn(w, r, campaignID, characterID)
	}
}

// withCampaignAndSessionID extracts campaign/session IDs and delegates to fn,
// returning 404 when either route parameter is missing.
func (h handlers) withCampaignAndSessionID(fn func(http.ResponseWriter, *http.Request, string, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		campaignID, ok := h.routeCampaignID(r)
		if !ok {
			h.WriteNotFound(w, r)
			return
		}
		sessionID, ok := h.routeSessionID(r)
		if !ok {
			h.WriteNotFound(w, r)
			return
		}
		fn(w, r, campaignID, sessionID)
	}
}
