package campaigns

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestMountCampaignGameRouteRedirectsToPlayHost(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{items: []campaignapp.CampaignSummary{{
		ID:            "c1",
		Name:          "The Guildhouse",
		CoverImageURL: "/static/campaign-covers/abandoned_castle_courtyard.png",
	}}}, modulehandler.NewBase(func(*http.Request) string { return "user-1" }, nil, nil), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignGame("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusSeeOther)
	}
	location := rr.Header().Get("Location")
	if !strings.HasPrefix(location, "http://play.example.com/campaigns/c1?launch=") {
		t.Fatalf("Location = %q, want play host handoff", location)
	}
}

func TestMountCampaignGameRouteRedirectsToPlayPortForLoopbackHosts(t *testing.T) {
	t.Parallel()

	cfg := configWithGateway(
		fakeGateway{items: []campaignapp.CampaignSummary{{ID: "c1", Name: "The Guildhouse"}}},
		modulehandler.NewBase(func(*http.Request) string { return "user-1" }, nil, nil),
		nil,
	)
	cfg.PlayFallbackPort = "8094"
	m := New(cfg)

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080"+routepath.AppCampaignGame("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusSeeOther)
	}
	location := rr.Header().Get("Location")
	if !strings.HasPrefix(location, "http://localhost:8094/campaigns/c1?launch=") {
		t.Fatalf("Location = %q, want localhost play-port handoff", location)
	}
}

func TestMountCampaignGameRouteHTMXRedirectsToPlayHost(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{items: []campaignapp.CampaignSummary{{ID: "c1", Name: "The Guildhouse"}}}, modulehandler.NewBase(func(*http.Request) string { return "user-1" }, nil, nil), nil))
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignGame("c1"), nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("HX-Redirect"); !strings.HasPrefix(got, "http://play.example.com/campaigns/c1?launch=") {
		t.Fatalf("HX-Redirect = %q, want play host handoff", got)
	}
}

func TestMountCampaignGameRouteHandlesLaunchGrantFailure(t *testing.T) {
	t.Parallel()

	cfg := configWithGateway(
		fakeGateway{items: []campaignapp.CampaignSummary{{ID: "c1", Name: "The Guildhouse"}}},
		modulehandler.NewBase(func(*http.Request) string { return "user-1" }, nil, nil),
		nil,
	)
	cfg.PlayLaunchGrant = playlaunchgrant.Config{}
	m := New(cfg)
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignGame("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}
