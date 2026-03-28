package detail

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

type detailLocalizer map[string]string

func (l detailLocalizer) Sprintf(key message.Reference, args ...any) string {
	ref := fmt.Sprint(key)
	if value, ok := l[ref]; ok {
		return value
	}
	return ref
}

type detailWorkspaceService struct {
	workspace campaignapp.CampaignWorkspace
	err       error
}

func (s detailWorkspaceService) CampaignName(context.Context, string) string { return s.workspace.Name }
func (s detailWorkspaceService) CampaignWorkspace(context.Context, string) (campaignapp.CampaignWorkspace, error) {
	return s.workspace, s.err
}

type detailSessionReads struct {
	sessions []campaignapp.CampaignSession
	err      error
}

func (r detailSessionReads) CampaignSessions(context.Context, string) ([]campaignapp.CampaignSession, error) {
	return append([]campaignapp.CampaignSession(nil), r.sessions...), r.err
}
func (detailSessionReads) CampaignSessionReadiness(context.Context, string, language.Tag) (campaignapp.CampaignSessionReadiness, error) {
	return campaignapp.CampaignSessionReadiness{}, nil
}

type detailAuthorization struct {
	manageCampaignErr     error
	manageSessionErr      error
	manageParticipantsErr error
	manageInvitesErr      error
	mutateCharactersErr   error
}

func (a detailAuthorization) RequireManageCampaign(context.Context, string) error {
	return a.manageCampaignErr
}
func (a detailAuthorization) RequireManageSession(context.Context, string) error {
	return a.manageSessionErr
}
func (a detailAuthorization) RequireManageParticipants(context.Context, string) error {
	return a.manageParticipantsErr
}
func (a detailAuthorization) RequireManageInvites(context.Context, string) error {
	return a.manageInvitesErr
}
func (a detailAuthorization) RequireMutateCharacters(context.Context, string) error {
	return a.mutateCharactersErr
}

func TestSortedActiveSessionsPrefersNewestStartedAt(t *testing.T) {
	t.Parallel()

	sessions := SortedActiveSessions([]campaignapp.CampaignSession{
		{ID: "old", Status: "active", StartedAt: "2026-03-20 10:00 UTC"},
		{ID: "new", Status: "active", StartedAt: "2026-03-21 10:00 UTC"},
		{ID: "draft", Status: "draft", StartedAt: "2026-03-22 10:00 UTC"},
	})
	if len(sessions) != 2 || sessions[0].ID != "new" || sessions[1].ID != "old" {
		t.Fatalf("SortedActiveSessions() = %#v", sessions)
	}
}

func TestCampaignSessionMenuStartTimeRejectsInvalidTimestamp(t *testing.T) {
	t.Parallel()

	if _, ok := campaignSessionMenuStartTime(campaignapp.CampaignSession{StartedAt: "not-a-time"}); ok {
		t.Fatal("campaignSessionMenuStartTime() unexpectedly parsed invalid input")
	}
	if _, ok := campaignSessionMenuStartTime(campaignapp.CampaignSession{StartedAt: "2026-03-21 10:00 UTC"}); !ok {
		t.Fatal("campaignSessionMenuStartTime() did not parse valid input")
	}
}

func TestLoadCampaignPageBuildsSharedWorkspaceContext(t *testing.T) {
	t.Parallel()

	h := NewHandler(
		NewSupport(
			modulehandler.NewBase(
				func(*http.Request) string { return "user-1" },
				func(*http.Request) string { return "pt-BR" },
				func(*http.Request) module.Viewer { return module.Viewer{} },
			),
			requestmeta.SchemePolicy{},
			nil,
		),
		PageServices{
			Workspace: detailWorkspaceService{
				workspace: campaignapp.CampaignWorkspace{
					ID:               "camp-1",
					Name:             "The Guildhouse",
					System:           "Daggerheart",
					CoverPreviewURL:  "/preview.jpg",
					CoverImageURL:    "/cover.jpg",
					ParticipantCount: "3",
					CharacterCount:   "5",
				},
			},
			SessionReads: detailSessionReads{
				sessions: []campaignapp.CampaignSession{
					{ID: "sess-1", Name: "Session One", Status: "Active", StartedAt: "2026-03-21 10:00 UTC"},
				},
			},
			Authorization: detailAuthorization{
				manageInvitesErr: errors.New("forbidden"),
			},
		},
	)

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaign("camp-1"), nil)
	rr := httptest.NewRecorder()

	ctx, page, err := h.LoadCampaignPage(rr, req, "camp-1")

	if err != nil {
		t.Fatalf("LoadCampaignPage() error = %v", err)
	}
	if ctx == nil || page == nil {
		t.Fatalf("LoadCampaignPage() returned nil values: ctx=%v page=%#v", ctx, page)
	}
	if page.Workspace.Name != "The Guildhouse" || len(page.Sessions) != 1 {
		t.Fatalf("page = %#v", page)
	}
	if !page.CanManageSession || page.CanManageInvites {
		t.Fatalf("page permissions = %#v", page)
	}
	if page.Lang != "pt-BR" || page.Locale != language.MustParse("pt-BR") {
		t.Fatalf("page locale = lang:%q locale:%q", page.Lang, page.Locale)
	}
}

func TestLoadCampaignPageOrWriteErrorWritesTransportError(t *testing.T) {
	t.Parallel()

	h := NewHandler(
		NewSupport(modulehandler.NewBase(nil, nil, nil), requestmeta.SchemePolicy{}, nil),
		PageServices{
			Workspace:     detailWorkspaceService{err: errors.New("boom")},
			SessionReads:  detailSessionReads{},
			Authorization: detailAuthorization{},
		},
	)

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaign("camp-1"), nil)
	rr := httptest.NewRecorder()

	_, _, ok := h.LoadCampaignPageOrWriteError(rr, req, "camp-1")

	if ok {
		t.Fatal("LoadCampaignPageOrWriteError() unexpectedly succeeded")
	}
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestPageContextBuildsWorkspaceShellState(t *testing.T) {
	t.Parallel()

	loc := detailLocalizer{
		"game.campaigns.title":           "Campaigns",
		"game.campaign.menu.overview":    "Overview",
		"game.participants.title":        "Participants",
		"game.characters.title":          "Characters",
		"game.sessions.title":            "Sessions",
		"game.sessions.menu.start":       "Start",
		"game.sessions.action_join_game": "Join Game",
		"game.campaign_invites.title":    "Invites",
		"game.campaign.title":            "Campaign",
	}
	page := &PageContext{
		Workspace: campaignapp.CampaignWorkspace{
			ID:               "camp-1",
			Name:             "The Guildhouse",
			Theme:            "Storm",
			System:           "Daggerheart",
			GMMode:           "Human",
			Status:           "Active",
			Locale:           "Portuguese (Brazil)",
			Intent:           "Standard",
			AccessPolicy:     "invite-only",
			CoverPreviewURL:  "/preview.jpg",
			CoverImageURL:    "/cover.jpg",
			ParticipantCount: "",
			CharacterCount:   "5",
		},
		Sessions: []campaignapp.CampaignSession{
			{ID: "sess-1", Name: "Session One", Status: "Active", StartedAt: "2026-03-21 10:00 UTC"},
			{ID: "sess-2", Name: "Session Zero", Status: "Draft"},
		},
		CanManageSession: true,
		CanManageInvites: true,
		Loc:              loc,
	}

	if got := CampaignMainClass(page.Workspace.CoverImageURL); got != "px-4" {
		t.Fatalf("CampaignMainClass() = %q, want %q", got, "px-4")
	}
	if !page.OutOfGameActionsLocked() {
		t.Fatal("OutOfGameActionsLocked() = false, want true")
	}
	if got := page.Title("camp-1"); got != "The Guildhouse" {
		t.Fatalf("Title() = %q, want %q", got, "The Guildhouse")
	}
	if got := CampaignWorkspaceLocaleFormValue(page.Workspace.Locale); got != "pt-BR" {
		t.Fatalf("CampaignWorkspaceLocaleFormValue() = %q, want %q", got, "pt-BR")
	}

	crumbs := CampaignBreadcrumbs("camp-1", page.Workspace.Name, loc, sharedtemplates.BreadcrumbItem{
		Label: "Sessions",
		URL:   routepath.AppCampaignSessions("camp-1"),
	})
	if len(crumbs) != 3 || crumbs[1].URL != routepath.AppCampaign("camp-1") {
		t.Fatalf("CampaignBreadcrumbs() = %#v", crumbs)
	}

	layout := page.Layout("camp-1", routepath.AppCampaignSessions("camp-1"))
	if layout.MainClass != "px-4" || layout.Metadata.RouteArea != webtemplates.RouteAreaCampaignWorkspace {
		t.Fatalf("Layout() = %#v", layout)
	}
	if layout.SideMenu == nil || len(layout.SideMenu.Items) != 5 {
		t.Fatalf("Layout().SideMenu = %#v", layout.SideMenu)
	}
	if got := layout.SideMenu.Items[3].Badge; got != "2" {
		t.Fatalf("session badge = %q, want %q", got, "2")
	}
	if got := len(layout.SideMenu.Items[3].SubItems); got != 1 {
		t.Fatalf("session subitems = %d, want 1", got)
	}

	header := page.Header("camp-1", crumbs)
	if header.Title != "The Guildhouse" || len(header.Breadcrumbs) != 3 {
		t.Fatalf("Header() = %#v", header)
	}

	base := page.BaseDetailView("camp-1")
	if !base.ActionsLocked || base.LocaleValue != "pt-BR" {
		t.Fatalf("BaseDetailView() = %#v", base)
	}
	if !base.CanManageSession || !base.CanManageInvites {
		t.Fatalf("BaseDetailView() permissions = %#v", base)
	}
}

func TestCampaignBreadcrumbsAndTitleFallbackToIDs(t *testing.T) {
	t.Parallel()

	loc := detailLocalizer{
		"game.campaigns.title": "Campaigns",
		"game.campaign.title":  "Campaign",
	}
	crumbs := CampaignBreadcrumbs("camp-1", "   ", loc)
	if len(crumbs) != 2 || crumbs[1].Label != "camp-1" || crumbs[1].URL != "" {
		t.Fatalf("CampaignBreadcrumbs() = %#v", crumbs)
	}

	page := &PageContext{Workspace: campaignapp.CampaignWorkspace{ID: "workspace-1"}, Loc: loc}
	if got := page.Title("campaign-1"); got != "workspace-1" {
		t.Fatalf("Title() = %q, want %q", got, "workspace-1")
	}
	page.Workspace.ID = ""
	if got := page.Title("campaign-1"); got != "campaign-1" {
		t.Fatalf("Title() = %q, want %q", got, "campaign-1")
	}
	if got := (&PageContext{Loc: loc}).Title(""); got != "Campaign" {
		t.Fatalf("Title() empty fallback = %q, want %q", got, "Campaign")
	}
}
