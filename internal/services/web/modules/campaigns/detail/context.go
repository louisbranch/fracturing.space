package detail

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"golang.org/x/sync/errgroup"
	"golang.org/x/text/language"
)

// PageContext holds the shared state loaded for campaign workspace pages that
// render the common campaign shell.
type PageContext struct {
	Workspace        campaignapp.CampaignWorkspace
	Sessions         []campaignapp.CampaignSession
	CanManageSession bool
	CanManageInvites bool
	Loc              webtemplates.Localizer
	Lang             string
	Locale           language.Tag
}

// LoadCampaignPage loads the shared workspace page state needed by campaign
// detail, chat, and creation routes. The four backend calls run concurrently.
func (h Handler) LoadCampaignPage(w http.ResponseWriter, r *http.Request, campaignID string) (context.Context, *PageContext, error) {
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
		workspace, err = h.Pages.Workspace.CampaignWorkspace(gctx, campaignID)
		return err
	})
	g.Go(func() error {
		var err error
		sessions, err = h.Pages.SessionReads.CampaignSessions(gctx, campaignID)
		return err
	})
	g.Go(func() error {
		canManageSession = h.Pages.Authorization.RequireManageSession(gctx, campaignID) == nil
		return nil
	})
	g.Go(func() error {
		canManageInvites = h.Pages.Authorization.RequireManageInvites(gctx, campaignID) == nil
		return nil
	})
	if err := g.Wait(); err != nil {
		return nil, nil, err
	}

	return ctx, &PageContext{
		Workspace:        workspace,
		Sessions:         sessions,
		CanManageSession: canManageSession,
		CanManageInvites: canManageInvites,
		Loc:              loc,
		Lang:             lang,
		Locale:           h.RequestLocaleTag(r),
	}, nil
}

// CampaignBreadcrumbs builds breadcrumbs for one campaign workspace page.
func CampaignBreadcrumbs(campaignID, campaignLabel string, loc webtemplates.Localizer, extra ...sharedtemplates.BreadcrumbItem) []sharedtemplates.BreadcrumbItem {
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

// CampaignMainClass centralizes the campaign workspace shell treatment in one
// seam so detail/chat/creation pages reuse the same cover-image layout rule.
func CampaignMainClass(coverImageURL string) string {
	coverImageURL = strings.TrimSpace(coverImageURL)
	if coverImageURL == "" {
		return "max-w-none"
	}
	return "px-4"
}

// Layout builds the common campaign workspace shell for one page.
func (p *PageContext) Layout(campaignID, currentPath string) webtemplates.AppMainLayoutOptions {
	return webtemplates.AppMainLayoutOptions{
		SideMenu: CampaignWorkspaceMenu(p.Workspace, currentPath, p.Sessions, p.CanManageInvites, p.Loc),
		MainBackground: &webtemplates.AppBackgroundImage{
			PreviewURL: strings.TrimSpace(p.Workspace.CoverPreviewURL),
			FullURL:    strings.TrimSpace(p.Workspace.CoverImageURL),
		},
		MainClass: CampaignMainClass(p.Workspace.CoverImageURL),
		Metadata: webtemplates.AppMainLayoutMetadata{
			RouteArea: webtemplates.RouteAreaCampaignWorkspace,
		},
	}
}

// OutOfGameActionsLocked reports whether active session state should disable
// campaign metadata, participant, invite, or character UI actions.
func (p *PageContext) OutOfGameActionsLocked() bool {
	for _, session := range p.Sessions {
		if CampaignSessionMenuIsActive(session) {
			return true
		}
	}
	return false
}

// Title resolves the page title for one campaign workspace page.
func (p *PageContext) Title(campaignID string) string {
	name := strings.TrimSpace(p.Workspace.Name)
	if name != "" {
		return name
	}
	if id := strings.TrimSpace(p.Workspace.ID); id != "" {
		return id
	}
	if id := strings.TrimSpace(campaignID); id != "" {
		return id
	}
	return webtemplates.T(p.Loc, "game.campaign.title")
}

// Header builds the shared campaign workspace page header.
func (p *PageContext) Header(campaignID string, breadcrumbs []sharedtemplates.BreadcrumbItem) *webtemplates.AppMainHeader {
	return p.HeaderWithAction(campaignID, breadcrumbs, nil)
}

// HeaderWithAction builds the shared campaign workspace page header with an
// optional primary action.
func (p *PageContext) HeaderWithAction(
	campaignID string,
	breadcrumbs []sharedtemplates.BreadcrumbItem,
	action *webtemplates.AppMainHeaderAction,
) *webtemplates.AppMainHeader {
	return &webtemplates.AppMainHeader{
		Title:       p.Title(campaignID),
		Action:      action,
		Breadcrumbs: breadcrumbs,
	}
}

// BaseDetailView maps shared campaign workspace state into the base detail
// render model consumed by section-owned detail view builders.
func (p *PageContext) BaseDetailView(campaignID string) campaignrender.CampaignDetailBaseView {
	return campaignrender.CampaignDetailBaseView{
		CampaignID:       campaignID,
		Name:             p.Workspace.Name,
		Theme:            p.Workspace.Theme,
		System:           p.Workspace.System,
		GMMode:           p.Workspace.GMMode,
		Status:           p.Workspace.Status,
		Locale:           p.Workspace.Locale,
		LocaleValue:      CampaignWorkspaceLocaleFormValue(p.Workspace.Locale),
		Intent:           p.Workspace.Intent,
		AccessPolicy:     p.Workspace.AccessPolicy,
		ActionsLocked:    p.OutOfGameActionsLocked(),
		CanManageSession: p.CanManageSession,
		CanManageInvites: p.CanManageInvites,
	}
}

// CampaignWorkspaceMenu builds the side navigation menu for a campaign
// workspace page.
func CampaignWorkspaceMenu(
	workspace campaignapp.CampaignWorkspace,
	currentPath string,
	sessions []campaignapp.CampaignSession,
	canManageInvites bool,
	loc webtemplates.Localizer,
) *webtemplates.AppSideMenu {
	campaignID := strings.TrimSpace(workspace.ID)
	if campaignID == "" {
		return nil
	}
	sessionSubItems := campaignSessionMenuSubItems(campaignID, sessions, loc)
	participantCount := strings.TrimSpace(workspace.ParticipantCount)
	if participantCount == "" {
		participantCount = "0"
	}
	characterCount := strings.TrimSpace(workspace.CharacterCount)
	if characterCount == "" {
		characterCount = "0"
	}
	overviewURL := routepath.AppCampaign(campaignID)
	sessionsURL := routepath.AppCampaignSessions(campaignID)
	participantsURL := routepath.AppCampaignParticipants(campaignID)
	invitesURL := routepath.AppCampaignInvites(campaignID)
	charactersURL := routepath.AppCampaignCharacters(campaignID)
	items := []webtemplates.AppSideMenuItem{
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
			Badge:       participantCount,
			IconID:      commonv1.IconId_ICON_ID_PARTICIPANT,
		},
		{
			Label:       webtemplates.T(loc, "game.characters.title"),
			URL:         charactersURL,
			MatchPrefix: charactersURL,
			Badge:       characterCount,
			IconID:      commonv1.IconId_ICON_ID_CHARACTER,
		},
		{
			Label:       webtemplates.T(loc, "game.sessions.title"),
			URL:         sessionsURL,
			MatchPrefix: sessionsURL,
			Badge:       strconv.Itoa(campaignSessionMenuCount(sessions)),
			IconID:      commonv1.IconId_ICON_ID_SESSION,
			SubItems:    sessionSubItems,
		},
	}
	if canManageInvites {
		items = append(items, webtemplates.AppSideMenuItem{
			Label:       webtemplates.T(loc, "game.campaign_invites.title"),
			URL:         invitesURL,
			MatchPrefix: invitesURL,
			IconID:      commonv1.IconId_ICON_ID_INVITES,
		})
	}
	return &webtemplates.AppSideMenu{
		CurrentPath: strings.TrimSpace(currentPath),
		Items:       items,
	}
}

const campaignSessionTimestampLayout = "2006-01-02 15:04 UTC"

// campaignSessionMenuSubItems maps active sessions into workspace menu entries.
func campaignSessionMenuSubItems(campaignID string, sessions []campaignapp.CampaignSession, loc webtemplates.Localizer) []webtemplates.AppSideMenuSubItem {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" || len(sessions) == 0 {
		return []webtemplates.AppSideMenuSubItem{}
	}

	startLabel := webtemplates.T(loc, "game.sessions.menu.start")
	result := make([]webtemplates.AppSideMenuSubItem, 0)
	for _, session := range sessions {
		sessionID := strings.TrimSpace(session.ID)
		if sessionID == "" || !CampaignSessionMenuIsActive(session) {
			continue
		}

		startValue := strings.TrimSpace(session.StartedAt)
		if startValue == "" {
			startValue = "-"
		}

		result = append(result, webtemplates.AppSideMenuSubItem{
			Label:         strings.TrimSpace(session.Name),
			URL:           routepath.AppCampaignSession(campaignID, sessionID),
			StartDetail:   startLabel + ": " + startValue,
			ActiveSession: true,
			JoinURL:       routepath.AppCampaignGame(campaignID),
			JoinLabel:     webtemplates.T(loc, "game.sessions.action_join_game"),
		})
	}

	return result
}

// campaignSessionMenuCount counts only renderable session menu rows.
func campaignSessionMenuCount(sessions []campaignapp.CampaignSession) int {
	count := 0
	for _, session := range sessions {
		if strings.TrimSpace(session.ID) == "" {
			continue
		}
		count++
	}
	return count
}

// CampaignSessionMenuIsActive marks session rows that should receive
// active-session styling.
func CampaignSessionMenuIsActive(session campaignapp.CampaignSession) bool {
	return strings.EqualFold(strings.TrimSpace(session.Status), "active")
}

// CampaignWorkspaceLocaleFormValue maps campaign locale labels/tags to form
// values.
func CampaignWorkspaceLocaleFormValue(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "pt", "pt-br", "portuguese (brazil)":
		return "pt-BR"
	default:
		return "en-US"
	}
}

// SortedActiveSessions returns active sessions ordered by most recent start
// time first.
func SortedActiveSessions(items []campaignapp.CampaignSession) []campaignapp.CampaignSession {
	active := make([]campaignapp.CampaignSession, 0, len(items))
	for _, session := range items {
		if !CampaignSessionMenuIsActive(session) {
			continue
		}
		active = append(active, session)
	}
	sort.SliceStable(active, func(i, j int) bool {
		iTime, iOK := campaignSessionMenuStartTime(active[i])
		jTime, jOK := campaignSessionMenuStartTime(active[j])
		switch {
		case iOK && jOK:
			if !iTime.Equal(jTime) {
				return iTime.After(jTime)
			}
		case iOK:
			return true
		case jOK:
			return false
		}
		return strings.TrimSpace(active[i].ID) < strings.TrimSpace(active[j].ID)
	})
	return active
}

// campaignSessionMenuStartTime parses the menu timestamp format into UTC ordering time.
func campaignSessionMenuStartTime(session campaignapp.CampaignSession) (time.Time, bool) {
	startedAt := strings.TrimSpace(session.StartedAt)
	if startedAt == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(campaignSessionTimestampLayout, startedAt)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}
