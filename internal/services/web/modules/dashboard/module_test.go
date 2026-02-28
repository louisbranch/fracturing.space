package dashboard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
	modulehandler "github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	"google.golang.org/grpc"
)

func TestModuleIDReturnsDashboard(t *testing.T) {
	t.Parallel()

	if got := New().ID(); got != "dashboard" {
		t.Fatalf("ID() = %q, want %q", got, "dashboard")
	}
}

func TestMountServesDashboardGet(t *testing.T) {
	t.Parallel()

	m := New()
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.DashboardPrefix, nil)
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

	m := New()
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

	m := New()
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.DashboardPrefix, nil)
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
	if strings.Contains(strings.ToLower(body), "<!doctype html") || strings.Contains(strings.ToLower(body), "<html") {
		t.Fatalf("expected htmx fragment without document wrapper")
	}
}

func TestMountRendersPendingProfileBlockFromUserHubState(t *testing.T) {
	t.Parallel()

	client := dashboardUserHubClientStub{resp: &userhubv1.GetDashboardResponse{
		User: &userhubv1.UserSummary{NeedsProfileCompletion: true},
	}}
	base := modulehandler.NewBase(
		func(*http.Request) string { return "user-1" },
		func(*http.Request) string { return "pt-BR" },
		nil,
	)
	m := NewWithGateway(NewGRPCGateway(client), base, nil)
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.DashboardPrefix, nil)
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

	client := dashboardUserHubClientStub{resp: &userhubv1.GetDashboardResponse{
		User:     &userhubv1.UserSummary{NeedsProfileCompletion: true},
		Metadata: &userhubv1.DashboardMetadata{DegradedDependencies: []string{"social.profile"}},
	}}
	base := modulehandler.NewBase(
		func(*http.Request) string { return "user-1" },
		func(*http.Request) string { return "en-US" },
		nil,
	)
	m := NewWithGateway(NewGRPCGateway(client), base, nil)
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.DashboardPrefix, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	// Invariant: degraded social profile state must suppress the pending-profile block.
	if strings.Contains(rr.Body.String(), `data-dashboard-block="profile-pending"`) {
		t.Fatalf("body = %q, want no pending-profile block when social profile is degraded", rr.Body.String())
	}
	// Invariant: degraded social profile state suppresses dashboard nudges derived from userhub state.
	if strings.Contains(rr.Body.String(), `data-dashboard-block="campaign-adventure"`) {
		t.Fatalf("body = %q, want no campaign-adventure block when social profile is degraded", rr.Body.String())
	}
}

func TestMountRendersCampaignAdventureBlockWhenNoDraftOrActiveCampaignExists(t *testing.T) {
	t.Parallel()

	client := dashboardUserHubClientStub{resp: &userhubv1.GetDashboardResponse{
		Campaigns: &userhubv1.CampaignSummary{
			HasMore: false,
			Campaigns: []*userhubv1.CampaignPreview{
				{CampaignId: "camp-1", Status: userhubv1.CampaignStatus_CAMPAIGN_STATUS_COMPLETED},
			},
		},
	}}
	base := modulehandler.NewBase(
		func(*http.Request) string { return "user-1" },
		func(*http.Request) string { return "en-US" },
		nil,
	)
	m := NewWithGateway(NewGRPCGateway(client), base, nil)
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.DashboardPrefix, nil)
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

	client := dashboardUserHubClientStub{resp: &userhubv1.GetDashboardResponse{
		Campaigns: &userhubv1.CampaignSummary{
			HasMore: false,
			Campaigns: []*userhubv1.CampaignPreview{
				{CampaignId: "camp-1", Status: userhubv1.CampaignStatus_CAMPAIGN_STATUS_DRAFT},
			},
		},
	}}
	base := modulehandler.NewBase(
		func(*http.Request) string { return "user-1" },
		func(*http.Request) string { return "en-US" },
		nil,
	)
	m := NewWithGateway(NewGRPCGateway(client), base, nil)
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.DashboardPrefix, nil)
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

	client := dashboardUserHubClientStub{resp: &userhubv1.GetDashboardResponse{
		Metadata: &userhubv1.DashboardMetadata{DegradedDependencies: []string{"game.campaigns"}},
		Campaigns: &userhubv1.CampaignSummary{
			HasMore: false,
			Campaigns: []*userhubv1.CampaignPreview{
				{CampaignId: "camp-1", Status: userhubv1.CampaignStatus_CAMPAIGN_STATUS_COMPLETED},
			},
		},
	}}
	base := modulehandler.NewBase(
		func(*http.Request) string { return "user-1" },
		func(*http.Request) string { return "en-US" },
		nil,
	)
	m := NewWithGateway(NewGRPCGateway(client), base, nil)
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.DashboardPrefix, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	// Invariant: degraded campaign dependency must suppress campaign-adventure block.
	if strings.Contains(rr.Body.String(), `data-dashboard-block="campaign-adventure"`) {
		t.Fatalf("body = %q, want no campaign-adventure block when campaign state is degraded", rr.Body.String())
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
