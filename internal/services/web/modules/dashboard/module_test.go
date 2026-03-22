package dashboard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
	dashboardapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard/app"
	dashboardgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard/gateway"
	modulehandler "github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	"google.golang.org/grpc"
)

func TestModuleIDReturnsDashboard(t *testing.T) {
	t.Parallel()

	if got := New(Config{}).ID(); got != "dashboard" {
		t.Fatalf("ID() = %q, want %q", got, "dashboard")
	}
}

func TestMountServesDashboardGet(t *testing.T) {
	t.Parallel()

	m := New(Config{})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppDashboard, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("content-type = %q, want %q", got, "text/html; charset=utf-8")
	}
	body := rr.Body.String()
	if !strings.Contains(body, "dashboard-root") {
		t.Fatalf("body = %q, want dashboard marker", body)
	}
	if !strings.Contains(body, `id="dashboard-root" hx-history="false"`) {
		t.Fatalf("body = %q, want dashboard history opt-out", body)
	}
	// Invariant: default dashboard should not render profile-pending block when userhub state is absent.
	if strings.Contains(body, `data-dashboard-block="profile-pending"`) {
		t.Fatalf("body = %q, want no pending-profile block", body)
	}
	// Invariant: default dashboard should not render campaign-adventure block when userhub state is absent.
	if strings.Contains(body, `data-dashboard-block="campaign-adventure"`) {
		t.Fatalf("body = %q, want no campaign-adventure block", body)
	}
}

func TestMountServesDashboardHead(t *testing.T) {
	t.Parallel()

	m := New(Config{})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodHead, routepath.AppDashboard, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestMountDashboardHTMXReturnsFragmentWithoutDocumentWrapper(t *testing.T) {
	t.Parallel()

	m := New(Config{})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppDashboard, nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "dashboard-root") {
		t.Fatalf("body = %q, want dashboard marker", body)
	}
	if !strings.Contains(body, `id="dashboard-root" hx-history="false"`) {
		t.Fatalf("body = %q, want dashboard history opt-out", body)
	}
	if strings.Contains(strings.ToLower(body), "<!doctype html") || strings.Contains(strings.ToLower(body), "<html") {
		t.Fatalf("expected htmx fragment without document wrapper")
	}
}

func TestMountRendersUnavailableStatusNoticeWhenDashboardDataFails(t *testing.T) {
	t.Parallel()

	base := modulehandler.NewBase(
		func(*http.Request) string { return "user-1" },
		func(*http.Request) string { return "en-US" },
		nil,
	)
	m := New(Config{Base: base})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppDashboard, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), `data-dashboard-status="unavailable"`) {
		t.Fatalf("body = %q, want unavailable dashboard status notice", rr.Body.String())
	}
}

func TestMountRendersPendingProfileBlockFromUserHubState(t *testing.T) {
	t.Parallel()

	base := modulehandler.NewBase(
		func(*http.Request) string { return "user-1" },
		func(*http.Request) string { return "pt-BR" },
		nil,
	)
	gateway := &fakeGateway{snapshot: dashboardapp.DashboardSnapshot{NeedsProfileCompletion: true}}
	m := New(Config{
		Service: dashboardapp.NewService(gateway, nil, nil),
		Base:    base,
	})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppDashboard, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `data-dashboard-block="profile-pending"`) {
		t.Fatalf("body = %q, want pending-profile block", body)
	}
	if !strings.Contains(body, routepath.AppSettingsProfile) {
		t.Fatalf("body = %q, want settings profile CTA", body)
	}
}

func TestMountHidesPendingProfileBlockWhenSocialStateIsDegraded(t *testing.T) {
	t.Parallel()

	base := modulehandler.NewBase(
		func(*http.Request) string { return "user-1" },
		func(*http.Request) string { return "en-US" },
		nil,
	)
	gateway := &fakeGateway{snapshot: dashboardapp.DashboardSnapshot{
		NeedsProfileCompletion: true,
		DegradedDependencies:   []string{dashboardapp.DegradedDependencySocialProfile},
	}}
	m := New(Config{
		Service: dashboardapp.NewService(gateway, nil, nil),
		Base:    base,
	})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppDashboard, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	// Invariant: degraded social profile state must suppress the pending-profile block.
	if strings.Contains(rr.Body.String(), `data-dashboard-block="profile-pending"`) {
		t.Fatalf("body = %q, want no pending-profile block when social profile is degraded", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `data-dashboard-status="degraded"`) {
		t.Fatalf("body = %q, want degraded dashboard status notice", rr.Body.String())
	}
	// Invariant: social-profile degradation must not suppress unrelated dashboard sections that still come from userhub.
	if !strings.Contains(rr.Body.String(), `data-dashboard-block="campaign-adventure"`) {
		t.Fatalf("body = %q, want campaign-adventure block when campaign data remains available", rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), `data-dashboard-block="active-sessions"`) {
		t.Fatalf("body = %q, want no active-sessions block when social profile is degraded", rr.Body.String())
	}
}

func TestMountRendersCampaignAdventureBlockWhenNoDraftOrActiveCampaignExists(t *testing.T) {
	t.Parallel()

	base := modulehandler.NewBase(
		func(*http.Request) string { return "user-1" },
		func(*http.Request) string { return "en-US" },
		nil,
	)
	gateway := &fakeGateway{snapshot: dashboardapp.DashboardSnapshot{
		HasDraftOrActiveCampaign: false,
		CampaignsHasMore:         false,
	}}
	m := New(Config{
		Service: dashboardapp.NewService(gateway, nil, nil),
		Base:    base,
	})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppDashboard, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `data-dashboard-block="campaign-adventure"`) {
		t.Fatalf("body = %q, want campaign-adventure block", body)
	}
	if !strings.Contains(body, routepath.AppCampaignsNew) {
		t.Fatalf("body = %q, want campaign start-choice CTA", body)
	}
}

func TestMountHidesCampaignAdventureBlockWhenDraftOrActiveCampaignExists(t *testing.T) {
	t.Parallel()

	base := modulehandler.NewBase(
		func(*http.Request) string { return "user-1" },
		func(*http.Request) string { return "en-US" },
		nil,
	)
	gateway := &fakeGateway{snapshot: dashboardapp.DashboardSnapshot{
		HasDraftOrActiveCampaign: true,
		CampaignsHasMore:         false,
	}}
	m := New(Config{
		Service: dashboardapp.NewService(gateway, nil, nil),
		Base:    base,
	})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppDashboard, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	// Invariant: campaign-adventure prompt must be hidden when at least one draft/active campaign exists.
	if strings.Contains(rr.Body.String(), `data-dashboard-block="campaign-adventure"`) {
		t.Fatalf("body = %q, want no campaign-adventure block", rr.Body.String())
	}
}

func TestMountHidesCampaignAdventureBlockWhenCampaignStateIsDegraded(t *testing.T) {
	t.Parallel()

	base := modulehandler.NewBase(
		func(*http.Request) string { return "user-1" },
		func(*http.Request) string { return "en-US" },
		nil,
	)
	gateway := &fakeGateway{snapshot: dashboardapp.DashboardSnapshot{
		HasDraftOrActiveCampaign: false,
		CampaignsHasMore:         false,
		DegradedDependencies:     []string{dashboardapp.DegradedDependencyGameCampaigns},
	}}
	m := New(Config{
		Service: dashboardapp.NewService(gateway, nil, nil),
		Base:    base,
	})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppDashboard, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	// Invariant: degraded campaign dependency must suppress campaign-adventure block.
	if strings.Contains(rr.Body.String(), `data-dashboard-block="campaign-adventure"`) {
		t.Fatalf("body = %q, want no campaign-adventure block when campaign state is degraded", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `data-dashboard-status="degraded"`) {
		t.Fatalf("body = %q, want degraded dashboard status notice", rr.Body.String())
	}
}

func TestMountRendersActiveSessionsBlockWithMultipleJoinLinks(t *testing.T) {
	t.Parallel()

	client := dashboardUserHubClientStub{resp: &userhubv1.GetDashboardResponse{
		ActiveSessions: &userhubv1.ActiveSessionSummary{
			Available: true,
			Sessions: []*userhubv1.ActiveSessionPreview{
				{CampaignId: "camp-1", CampaignName: "Sunfall", SessionId: "session-1", SessionName: "The Crossing"},
				{CampaignId: "camp-2", CampaignName: "Gloam Tide", SessionId: "session-2", SessionName: "Session 2"},
			},
		},
	}}
	base := modulehandler.NewBase(
		func(*http.Request) string { return "user-1" },
		func(*http.Request) string { return "en-US" },
		nil,
	)
	gateway := dashboardgateway.NewGRPCGateway(client)
	m := New(Config{
		Service: dashboardapp.NewService(gateway, nil, nil),
		Base:    base,
	})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppDashboard, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `data-dashboard-block="active-sessions"`) {
		t.Fatalf("body = %q, want active-sessions block", body)
	}
	if strings.Contains(body, `data-dashboard-block="campaign-adventure"`) {
		t.Fatalf("body = %q, want no campaign-adventure block when active sessions exist", body)
	}
	if strings.Count(body, `>Join Game</a>`) != 2 {
		t.Fatalf("body = %q, want two Join Game CTAs", body)
	}
	if !strings.Contains(body, routepath.AppCampaignGame("camp-1")) || !strings.Contains(body, routepath.AppCampaignGame("camp-2")) {
		t.Fatalf("body = %q, want join links for both campaigns", body)
	}
	if !strings.Contains(body, "Session 2") {
		t.Fatalf("body = %q, want named second session", body)
	}
}

func TestMountRendersCampaignStartNudgesBlock(t *testing.T) {
	t.Parallel()

	client := dashboardUserHubClientStub{resp: &userhubv1.GetDashboardResponse{
		CampaignStartNudges: &userhubv1.CampaignStartNudgeSummary{
			Available: true,
			Nudges: []*userhubv1.CampaignStartNudge{{
				CampaignId:        "camp-1",
				CampaignName:      "Sunfall",
				BlockerCode:       "CHARACTER_SYSTEM_REQUIRED",
				BlockerMessage:    "Finish Aria",
				ActionKind:        userhubv1.CampaignStartNudgeActionKind_CAMPAIGN_START_NUDGE_ACTION_KIND_COMPLETE_CHARACTER,
				TargetCharacterId: "char-1",
			}},
		},
	}}
	base := modulehandler.NewBase(
		func(*http.Request) string { return "user-1" },
		func(*http.Request) string { return "en-US" },
		nil,
	)
	gateway := dashboardgateway.NewGRPCGateway(client)
	m := New(Config{
		Service: dashboardapp.NewService(gateway, nil, nil),
		Base:    base,
	})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppDashboard, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `data-dashboard-block="campaign-start-nudges"`) {
		t.Fatalf("body = %q, want campaign-start-nudges block", body)
	}
	if !strings.Contains(body, routepath.AppCampaignCharacter("camp-1", "char-1")) {
		t.Fatalf("body = %q, want character detail CTA", body)
	}
	if !strings.Contains(body, "Finish character") {
		t.Fatalf("body = %q, want finish-character CTA label", body)
	}
}

func TestMountRendersStartSessionNudgeToSessionCreatePage(t *testing.T) {
	t.Parallel()

	client := dashboardUserHubClientStub{resp: &userhubv1.GetDashboardResponse{
		CampaignStartNudges: &userhubv1.CampaignStartNudgeSummary{
			Available: true,
			Nudges: []*userhubv1.CampaignStartNudge{{
				CampaignId:     "camp-1",
				CampaignName:   "Sunfall",
				ActionKind:     userhubv1.CampaignStartNudgeActionKind_CAMPAIGN_START_NUDGE_ACTION_KIND_START_SESSION,
				BlockerCode:    "START_SESSION_STALE",
				BlockerMessage: "Start a new session for this campaign.",
			}},
		},
	}}
	base := modulehandler.NewBase(
		func(*http.Request) string { return "user-1" },
		func(*http.Request) string { return "en-US" },
		nil,
	)
	gateway := dashboardgateway.NewGRPCGateway(client)
	m := New(Config{
		Service: dashboardapp.NewService(gateway, nil, nil),
		Base:    base,
	})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppDashboard, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, routepath.AppCampaignSessionCreate("camp-1")) {
		t.Fatalf("body = %q, want session create CTA", body)
	}
	if !strings.Contains(body, "Start session") {
		t.Fatalf("body = %q, want start-session CTA label", body)
	}
	if !strings.Contains(body, "This campaign is ready for a new session.") {
		t.Fatalf("body = %q, want localized start-session message", body)
	}
}

type dashboardUserHubClientStub struct {
	resp *userhubv1.GetDashboardResponse
	err  error
}

func (s dashboardUserHubClientStub) GetDashboard(_ context.Context, req *userhubv1.GetDashboardRequest, _ ...grpc.CallOption) (*userhubv1.GetDashboardResponse, error) {
	_ = req
	if s.err != nil {
		return nil, s.err
	}
	return s.resp, nil
}
