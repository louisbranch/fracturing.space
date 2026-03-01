package campaigns

import (
	"context"
	"net/http"
	"strings"

	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
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
	workspace, err := h.service.CampaignWorkspace(ctx, campaignID)
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
		SideMenu:               campaignWorkspaceMenu(p.workspace, currentPath, p.loc),
		MainBackgroundImageURL: strings.TrimSpace(p.workspace.CoverImageURL),
		MainClass:              campaignMainClass(p.workspace.CoverImageURL),
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
