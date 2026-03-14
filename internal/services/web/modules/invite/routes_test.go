package invite

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	inviteapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/invite/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestresolver"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/sessioncookie"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestRegisterRoutesHandlesNilMux(t *testing.T) {
	t.Parallel()

	registerRoutes(nil, newHandlers(inviteapp.NewService(&routeGatewayStub{}), nil, requestmeta.SchemePolicy{}, nil))
}

func TestRegisterRoutesInviteMethodContracts(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	registerRoutes(mux, newHandlers(inviteapp.NewService(&routeGatewayStub{}), principalWithUserID("user-1"), requestmeta.SchemePolicy{}, nil))

	tests := []struct {
		name         string
		method       string
		path         string
		wantStatus   int
		wantAllow    string
		wantLocation string
	}{
		{name: "invite get", method: http.MethodGet, path: routepath.PublicInvite("inv-1"), wantStatus: http.StatusOK},
		{name: "invite head", method: http.MethodHead, path: routepath.PublicInvite("inv-1"), wantStatus: http.StatusOK},
		{name: "invite root", method: http.MethodGet, path: routepath.InvitePrefix, wantStatus: http.StatusNotFound},
		{name: "invite nested", method: http.MethodGet, path: routepath.PublicInvite("inv-1") + "/other", wantStatus: http.StatusNotFound},
		{name: "invite accept post", method: http.MethodPost, path: routepath.PublicInviteAccept("inv-1"), wantStatus: http.StatusSeeOther, wantLocation: routepath.AppCampaign("camp-1")},
		{name: "invite accept get", method: http.MethodGet, path: routepath.PublicInviteAccept("inv-1"), wantStatus: http.StatusMethodNotAllowed, wantAllow: http.MethodPost},
		{name: "invite decline post", method: http.MethodPost, path: routepath.PublicInviteDecline("inv-1"), wantStatus: http.StatusSeeOther, wantLocation: routepath.AppDashboard},
		{name: "invite decline get", method: http.MethodGet, path: routepath.PublicInviteDecline("inv-1"), wantStatus: http.StatusMethodNotAllowed, wantAllow: http.MethodPost},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
			if tc.wantAllow != "" {
				if got := rr.Header().Get("Allow"); got != tc.wantAllow {
					t.Fatalf("Allow = %q, want %q", got, tc.wantAllow)
				}
			}
			if tc.wantLocation != "" {
				if got := rr.Header().Get("Location"); got != tc.wantLocation {
					t.Fatalf("Location = %q, want %q", got, tc.wantLocation)
				}
			}
		})
	}
}

func TestRegisterRoutesRejectsCookieMutationWithoutSameOriginProof(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	registerRoutes(mux, newHandlers(inviteapp.NewService(&routeGatewayStub{}), principalWithUserID("user-1"), requestmeta.SchemePolicy{}, nil))

	req := httptest.NewRequest(http.MethodPost, routepath.PublicInviteAccept("inv-1"), nil)
	req.AddCookie(&http.Cookie{Name: sessioncookie.Name, Value: "sess-1"})
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestRegisterRoutesAllowsCookieMutationWithSameOriginProof(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	registerRoutes(mux, newHandlers(inviteapp.NewService(&routeGatewayStub{}), principalWithUserID("user-1"), requestmeta.SchemePolicy{}, nil))

	req := httptest.NewRequest(http.MethodPost, "http://app.example.test"+routepath.PublicInviteAccept("inv-1"), nil)
	req.Host = "app.example.test"
	req.Header.Set("Origin", "http://app.example.test")
	req.AddCookie(&http.Cookie{Name: sessioncookie.Name, Value: "sess-1"})
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusSeeOther)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppCampaign("camp-1") {
		t.Fatalf("Location = %q, want %q", got, routepath.AppCampaign("camp-1"))
	}
}

type routeGatewayStub struct{}

func (routeGatewayStub) GetPublicInvite(context.Context, string) (inviteapp.PublicInvite, error) {
	return inviteapp.PublicInvite{
		InviteID:        "inv-1",
		CampaignID:      "camp-1",
		CampaignName:    "Skyfall",
		ParticipantID:   "part-1",
		ParticipantName: "Scout",
		RecipientUserID: "user-1",
		Status:          inviteapp.InviteStatusPending,
	}, nil
}

func (routeGatewayStub) AcceptInvite(context.Context, string, inviteapp.PublicInvite) error {
	return nil
}

func (routeGatewayStub) DeclineInvite(context.Context, string, string) error {
	return nil
}

func principalWithUserID(userID string) requestresolver.Principal {
	return requestresolver.NewPrincipal(
		nil,
		func(*http.Request) bool { return userID != "" },
		func(*http.Request) string { return userID },
		nil,
		nil,
	)
}
