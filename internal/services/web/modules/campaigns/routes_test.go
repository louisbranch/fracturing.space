package campaigns

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
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

func TestStableRouteSurfacesOwnExpectedRouteGroups(t *testing.T) {
	t.Parallel()

	surfaces := stableRouteSurfaces()
	if len(surfaces) != 3 {
		t.Fatalf("len(stableRouteSurfaces()) = %d, want 3", len(surfaces))
	}
	if surfaces[0].id != "stable-core" {
		t.Fatalf("stable surface[0] id = %q, want %q", surfaces[0].id, "stable-core")
	}
	if surfaces[1].id != "stable-workflow" {
		t.Fatalf("stable surface[1] id = %q, want %q", surfaces[1].id, "stable-workflow")
	}
	if surfaces[2].id != "stable-mutations" {
		t.Fatalf("stable surface[2] id = %q, want %q", surfaces[2].id, "stable-mutations")
	}
	for _, surface := range surfaces {
		if surface.register == nil {
			t.Fatalf("surface %q missing register fn", surface.id)
		}
	}
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
		body       string
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
		{name: "campaign edit get", method: http.MethodGet, path: routepath.AppCampaignEdit("c1"), wantStatus: http.StatusOK},
		{name: "campaign edit post", method: http.MethodPost, path: routepath.AppCampaignEdit("c1"), body: "name=Updated&theme_prompt=Theme&locale=en-US", wantStatus: http.StatusFound, wantLoc: routepath.AppCampaign("c1")},
		{name: "campaign ai binding post", method: http.MethodPost, path: routepath.AppCampaignAIBinding("c1"), body: "participant_id=p-manager&ai_agent_id=agent-1", wantStatus: http.StatusForbidden},
		{name: "campaign session start get resolves session detail route", method: http.MethodGet, path: routepath.AppCampaignSessionStart("c1"), wantStatus: http.StatusOK},
		{name: "campaign session start post", method: http.MethodPost, path: routepath.AppCampaignSessionStart("c1"), body: "name=Session+One", wantStatus: http.StatusFound, wantLoc: routepath.AppCampaignSessions("c1")},
		{name: "campaign unknown subpath", method: http.MethodGet, path: routepath.AppCampaign("c1") + "/unknown", wantStatus: http.StatusNotFound},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			if tc.method == http.MethodPost {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
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

func TestRegisterStableRoutesExposeWorkspaceAndMutationRoutes(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	registerStableRoutes(
		mux,
		newHandlers(
			newService(fakeGateway{
				items:        []CampaignSummary{{ID: "c1", Name: "Campaign"}},
				participants: []CampaignParticipant{{ID: "p-1", Name: "Owner", Role: "GM", CampaignAccess: "Owner"}},
				participant:  CampaignParticipant{ID: "p-1", Name: "Owner", Role: "GM", CampaignAccess: "Owner"},
				sessions:     []CampaignSession{{ID: "sess-1", Name: "Session One"}},
				invites:      []CampaignInvite{{ID: "inv-1", ParticipantID: "p-1", RecipientUserID: "user-123", Status: "Pending"}},
				characterCreationProgress: CampaignCharacterCreationProgress{
					Steps:    []CampaignCharacterCreationStep{{Step: 1, Key: "class_subclass", Complete: false}},
					NextStep: 1,
				},
				authorizationDecision: campaignapp.AuthorizationDecision{
					Evaluated:           true,
					Allowed:             true,
					ActorCampaignAccess: "Owner",
				},
			}),
			modulehandler.NewBase(func(*http.Request) string { return "user-123" }, nil, nil),
			"",
		),
	)

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{name: "overview", method: http.MethodGet, path: routepath.AppCampaign("c1"), wantStatus: http.StatusOK},
		{name: "campaign edit", method: http.MethodGet, path: routepath.AppCampaignEdit("c1"), wantStatus: http.StatusOK},
		{name: "participants", method: http.MethodGet, path: routepath.AppCampaignParticipants("c1"), wantStatus: http.StatusOK},
		{name: "participant edit", method: http.MethodGet, path: routepath.AppCampaignParticipantEdit("c1", "p-1"), wantStatus: http.StatusOK},
		{name: "characters", method: http.MethodGet, path: routepath.AppCampaignCharacters("c1"), wantStatus: http.StatusOK},
		{name: "character detail", method: http.MethodGet, path: routepath.AppCampaignCharacter("c1", "char-1"), wantStatus: http.StatusOK},
		{name: "sessions", method: http.MethodGet, path: routepath.AppCampaignSessions("c1"), wantStatus: http.StatusOK},
		{name: "session detail", method: http.MethodGet, path: routepath.AppCampaignSession("c1", "sess-1"), wantStatus: http.StatusOK},
		{name: "invites", method: http.MethodGet, path: routepath.AppCampaignInvites("c1"), wantStatus: http.StatusOK},
		{name: "game", method: http.MethodGet, path: routepath.AppCampaignGame("c1"), wantStatus: http.StatusOK},
		{name: "participant update", method: http.MethodPost, path: routepath.AppCampaignParticipantEdit("c1", "p-1"), body: "name=Owner&role=gm&pronouns=they%2Fthem", wantStatus: http.StatusFound},
		{name: "campaign ai binding", method: http.MethodPost, path: routepath.AppCampaignAIBinding("c1"), body: "participant_id=p-1&ai_agent_id=agent-1", wantStatus: http.StatusFound},
		{name: "campaign update", method: http.MethodPost, path: routepath.AppCampaignEdit("c1"), body: "name=Updated&theme_prompt=Theme&locale=en-US", wantStatus: http.StatusFound},
		{name: "session start", method: http.MethodPost, path: routepath.AppCampaignSessionStart("c1"), body: "name=Session+Two", wantStatus: http.StatusFound},
		{name: "session end", method: http.MethodPost, path: routepath.AppCampaignSessionEnd("c1"), body: "session_id=sess-1", wantStatus: http.StatusFound},
		{name: "invite create", method: http.MethodPost, path: routepath.AppCampaignInviteCreate("c1"), body: "participant_id=p-1&recipient_user_id=user-123", wantStatus: http.StatusFound},
		{name: "invite revoke", method: http.MethodPost, path: routepath.AppCampaignInviteRevoke("c1"), body: "invite_id=inv-1", wantStatus: http.StatusFound},
		{name: "rest route", method: http.MethodGet, path: routepath.AppCampaign("c1") + "/rest", wantStatus: http.StatusNotFound},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			if tc.method == http.MethodPost {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Fatalf("path %q status = %d, want %d", tc.path, rr.Code, tc.wantStatus)
			}
		})
	}
}
