package campaigns

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestRegisterRoutesHandlesNilMux(t *testing.T) {
	t.Parallel()

	registerStableRoutes(
		nil,
		newHandlers(
			newService(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "Campaign"}}}),
			modulehandler.NewTestBase(),
			"",
		),
	)
}

func TestRegisterRoutesCampaignsPathAndMethodContracts(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	registerStableRoutes(
		mux,
		newHandlers(
			newService(fakeGateway{
				items:        []CampaignSummary{{ID: "c1", Name: "Campaign"}},
				participants: []CampaignParticipant{{ID: "p-manager", UserID: "user-123", CampaignAccess: "Manager"}},
			}),
			modulehandler.NewBase(func(*http.Request) string { return "user-123" }, nil, nil),
			"",
		),
	)

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantAllow  string
		wantLoc    string
	}{
		{name: "campaigns root", method: http.MethodGet, path: routepath.AppCampaigns, wantStatus: http.StatusOK},
		{name: "campaigns slash root", method: http.MethodGet, path: routepath.CampaignsPrefix, wantStatus: http.StatusOK},
		{name: "campaign new get", method: http.MethodGet, path: routepath.AppCampaignsNew, wantStatus: http.StatusOK},
		{name: "campaign create get", method: http.MethodGet, path: routepath.AppCampaignsCreate, wantStatus: http.StatusOK},
		{name: "campaign overview head", method: http.MethodHead, path: routepath.AppCampaign("c1"), wantStatus: http.StatusOK},
		{name: "campaign overview post rejected", method: http.MethodPost, path: routepath.AppCampaign("c1"), wantStatus: http.StatusMethodNotAllowed, wantAllow: http.MethodGet + ", HEAD"},
		{name: "campaign session start post", method: http.MethodPost, path: routepath.AppCampaignSessionStart("c1"), wantStatus: http.StatusNotFound},
		{name: "campaign unknown subpath", method: http.MethodGet, path: routepath.AppCampaign("c1") + "/unknown", wantStatus: http.StatusNotFound},
	}

	for _, tc := range tests {
		tc := tc
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
			if tc.wantLoc != "" {
				if got := rr.Header().Get("Location"); got != tc.wantLoc {
					t.Fatalf("Location = %q, want %q", got, tc.wantLoc)
				}
			}
		})
	}
}

func TestRegisterExperimentalRoutesExposeExperimentalWorkspaceRoutes(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	registerExperimentalRoutes(
		mux,
		newHandlers(
			newService(fakeGateway{
				items:    []CampaignSummary{{ID: "c1", Name: "Campaign"}},
				sessions: []CampaignSession{{ID: "sess-1", Name: "Session One"}},
				invites:  []CampaignInvite{{ID: "inv-1", ParticipantID: "p-1", RecipientUserID: "user-123", Status: "Pending"}},
			}),
			modulehandler.NewBase(func(*http.Request) string { return "user-123" }, nil, nil),
			"",
		),
	)

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{name: "sessions route", method: http.MethodGet, path: routepath.AppCampaignSessions("c1"), wantStatus: http.StatusOK},
		{name: "session detail route", method: http.MethodGet, path: routepath.AppCampaignSession("c1", "sess-1"), wantStatus: http.StatusOK},
		{name: "game route", method: http.MethodGet, path: routepath.AppCampaignGame("c1"), wantStatus: http.StatusOK},
		{name: "invites route", method: http.MethodGet, path: routepath.AppCampaignInvites("c1"), wantStatus: http.StatusOK},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
		})
	}
}

func TestRegisterStableRoutesExposeStableWorkspaceRoutesAndHideExperimentalRoutes(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	registerStableRoutes(
		mux,
		newHandlers(
			newService(fakeGateway{
				items: []CampaignSummary{{ID: "c1", Name: "Campaign"}},
				characterCreationProgress: CampaignCharacterCreationProgress{
					Steps:    []CampaignCharacterCreationStep{{Step: 1, Key: "class_subclass", Complete: false}},
					NextStep: 1,
				},
			}),
			modulehandler.NewBase(func(*http.Request) string { return "user-123" }, nil, nil),
			"",
		),
	)

	for _, tc := range []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{name: "overview", method: http.MethodGet, path: routepath.AppCampaign("c1"), wantStatus: http.StatusOK},
		{name: "participants", method: http.MethodGet, path: routepath.AppCampaignParticipants("c1"), wantStatus: http.StatusOK},
		{name: "characters", method: http.MethodGet, path: routepath.AppCampaignCharacters("c1"), wantStatus: http.StatusOK},
		{name: "character detail", method: http.MethodGet, path: routepath.AppCampaignCharacter("c1", "char-1"), wantStatus: http.StatusOK},
		{name: "rest route", method: http.MethodGet, path: routepath.AppCampaign("c1") + "/rest", wantStatus: http.StatusNotFound},
		{name: "sessions", method: http.MethodGet, path: routepath.AppCampaignSessions("c1"), wantStatus: http.StatusNotFound},
		{name: "session detail", method: http.MethodGet, path: routepath.AppCampaignSession("c1", "sess-1"), wantStatus: http.StatusNotFound},
		{name: "invites", method: http.MethodGet, path: routepath.AppCampaignInvites("c1"), wantStatus: http.StatusNotFound},
		{name: "game", method: http.MethodGet, path: routepath.AppCampaignGame("c1"), wantStatus: http.StatusNotFound},
		{name: "session start", method: http.MethodPost, path: routepath.AppCampaignSessionStart("c1"), wantStatus: http.StatusNotFound},
		{name: "session end", method: http.MethodPost, path: routepath.AppCampaignSessionEnd("c1"), wantStatus: http.StatusNotFound},
		{name: "participant update", method: http.MethodPost, path: routepath.AppCampaignParticipantUpdate("c1"), wantStatus: http.StatusNotFound},
		{name: "character update", method: http.MethodPost, path: routepath.AppCampaignCharacterUpdate("c1"), wantStatus: http.StatusNotFound},
		{name: "character control", method: http.MethodPost, path: routepath.AppCampaignCharacterControl("c1"), wantStatus: http.StatusNotFound},
		{name: "invite create", method: http.MethodPost, path: routepath.AppCampaignInviteCreate("c1"), wantStatus: http.StatusNotFound},
		{name: "invite revoke", method: http.MethodPost, path: routepath.AppCampaignInviteRevoke("c1"), wantStatus: http.StatusNotFound},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Fatalf("path %q status = %d, want %d", tc.path, rr.Code, tc.wantStatus)
			}
		})
	}
}
