package campaigns

import (
	"context"
	"net/http"
	"strings"

	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"golang.org/x/sync/errgroup"
	"golang.org/x/text/language"
)

// campaignBreadcrumbs builds breadcrumbs for one campaign workspace page.
// When extra items are provided, the campaign label becomes a link back to the
// campaign overview page.
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

// campaignMainClass centralizes the campaign workspace shell treatment in one
// seam so detail/chat/creation pages reuse the same cover-image layout rule.
func campaignMainClass(coverImageURL string) string {
	coverImageURL = strings.TrimSpace(coverImageURL)
	if coverImageURL == "" {
		return "max-w-none"
	}
	return "px-4"
}

// campaignPageContext holds the shared state loaded for campaign workspace
// pages that render the common campaign shell.
type campaignPageContext struct {
	workspace        campaignapp.CampaignWorkspace
	sessions         []campaignapp.CampaignSession
	canManageSession bool
	canManageInvites bool
	loc              webtemplates.Localizer
	lang             string
	locale           language.Tag
}

// loadCampaignPage loads the shared workspace page state needed by campaign
// detail, chat, and creation routes. The four backend calls — workspace fetch,
// session list, and two authorization checks — run concurrently.
func (h campaignDetailHandlers) loadCampaignPage(w http.ResponseWriter, r *http.Request, campaignID string) (context.Context, *campaignPageContext, error) {
	loc, lang := h.PageLocalizer(w, r)
	ctx, _ := h.RequestContextAndUserID(r)

	var (
		workspace        campaignapp.CampaignWorkspace
		sessions         []campaignapp.CampaignSession
		canManageSession bool
		canManageInvites bool
	)

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		workspace, err = h.pages.workspace.CampaignWorkspace(gctx, campaignID)
		return err
	})
	g.Go(func() error {
		var err error
		sessions, err = h.pages.sessionReads.CampaignSessions(gctx, campaignID)
		return err
	})
	g.Go(func() error {
		canManageSession = h.pages.authorization.RequireManageSession(gctx, campaignID) == nil
		return nil
	})
	g.Go(func() error {
		canManageInvites = h.pages.authorization.RequireManageInvites(gctx, campaignID) == nil
		return nil
	})
	if err := g.Wait(); err != nil {
		return nil, nil, err
	}

	return ctx, &campaignPageContext{
		workspace:        workspace,
		sessions:         sessions,
		canManageSession: canManageSession,
		canManageInvites: canManageInvites,
		loc:              loc,
		lang:             lang,
		locale:           h.RequestLocaleTag(r),
	}, nil
}

// layout builds the common campaign workspace shell for one page.
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

// outOfGameActionsLocked reports whether active session state should disable
// campaign metadata, participant, invite, or character UI actions.
func (p *campaignPageContext) outOfGameActionsLocked() bool {
	for _, session := range p.sessions {
		if campaignSessionMenuIsActive(session) {
			return true
		}
	}
	return false
}

// title resolves the page title for one campaign workspace page, preserving
// fallback behavior when the workspace name is missing.
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

// header builds the shared campaign workspace page header.
func (p *campaignPageContext) header(campaignID string, breadcrumbs []sharedtemplates.BreadcrumbItem) *webtemplates.AppMainHeader {
	return p.headerWithAction(campaignID, breadcrumbs, nil)
}

// headerWithAction builds the shared campaign workspace page header with an
// optional primary action.
func (p *campaignPageContext) headerWithAction(
	campaignID string,
	breadcrumbs []sharedtemplates.BreadcrumbItem,
	action *webtemplates.AppMainHeaderAction,
) *webtemplates.AppMainHeader {
	return &webtemplates.AppMainHeader{
		Title:       p.title(campaignID),
		Action:      action,
		Breadcrumbs: breadcrumbs,
	}
}
