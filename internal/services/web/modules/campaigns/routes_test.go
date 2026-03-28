package campaigns

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler/modulehandlertest"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func newRouteTestServiceConfig(gw fakeGateway) serviceConfigs {
	return serviceConfigsWithGateway(gw)
}

func newRouteHandlers(gw fakeGateway, base modulehandler.Base) handlers {
	handlerSet, err := newHandlers(handlersConfig{
		Services:        newHandlerServices(newRouteTestServiceConfig(gw)),
		Base:            base,
		PlayLaunchGrant: fakePlayLaunchGrantConfig(),
	})
	if err != nil {
		panic(err)
	}
	return handlerSet
}

func TestRegisterRoutesHandlesNilMux(t *testing.T) {
	t.Parallel()

	registerStableRoutes(
		nil,
		newRouteHandlers(fakeGateway{items: []campaignapp.CampaignSummary{{ID: "c1", Name: "Campaign"}}}, modulehandlertest.NewBase()),
	)
}

func TestStableRouteSurfacesOwnExpectedRouteGroups(t *testing.T) {
	t.Parallel()

	surfaces := stableRouteSurfaces()
	if len(surfaces) != 6 {
		t.Fatalf("len(stableRouteSurfaces()) = %d, want 6", len(surfaces))
	}
	wantIDs := []string{
		"stable-overview",
		"stable-starters",
		"stable-participants",
		"stable-characters",
		"stable-sessions-game",
		"stable-invites",
	}
	for idx, want := range wantIDs {
		if surfaces[idx].id != want {
			t.Fatalf("stable surface[%d] id = %q, want %q", idx, surfaces[idx].id, want)
		}
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
		newRouteHandlers(fakeGateway{
			items:        []campaignapp.CampaignSummary{{ID: "c1", Name: "Campaign"}},
			participants: []campaignapp.CampaignParticipant{{ID: "p-manager", UserID: "user-123", CampaignAccess: "Manager"}},
			characters:   []campaignapp.CampaignCharacter{{ID: "char-1", Name: "Hero", Kind: "PC", Controller: "user-123"}},
			sessions:     []campaignapp.CampaignSession{{ID: "start", Name: "Session Start", Status: "Active"}},
		}, modulehandler.NewBase(func(*http.Request) string { return "user-123" }, nil, nil)),
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
		{name: "campaign new get", method: http.MethodGet, path: routepath.AppCampaignsNew, wantStatus: http.StatusOK},
		{name: "campaign create get", method: http.MethodGet, path: routepath.AppCampaignsCreate, wantStatus: http.StatusOK},
		{name: "campaign overview head", method: http.MethodHead, path: routepath.AppCampaign("c1"), wantStatus: http.StatusOK},
		{name: "campaign overview post rejected", method: http.MethodPost, path: routepath.AppCampaign("c1"), wantStatus: http.StatusMethodNotAllowed, wantAllow: http.MethodGet + ", HEAD"},
		{name: "campaign edit get", method: http.MethodGet, path: routepath.AppCampaignEdit("c1"), wantStatus: http.StatusOK},
		{name: "campaign edit post", method: http.MethodPost, path: routepath.AppCampaignEdit("c1"), body: "name=Updated&theme_prompt=Theme&locale=en-US", wantStatus: http.StatusFound, wantLoc: routepath.AppCampaign("c1")},
		{name: "campaign ai binding get", method: http.MethodGet, path: routepath.AppCampaignAIBinding("c1"), wantStatus: http.StatusForbidden},
		{name: "campaign ai binding post", method: http.MethodPost, path: routepath.AppCampaignAIBinding("c1"), body: "ai_agent_id=agent-1", wantStatus: http.StatusFound},
		{name: "campaign session create get", method: http.MethodGet, path: routepath.AppCampaignSessionCreate("c1"), wantStatus: http.StatusOK},
		{name: "campaign session create post", method: http.MethodPost, path: routepath.AppCampaignSessionCreate("c1"), body: "name=Session+One", wantStatus: http.StatusFound, wantLoc: routepath.AppCampaignSessions("c1")},
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
		newRouteHandlers(fakeGateway{
			items:        []campaignapp.CampaignSummary{{ID: "c1", Name: "Campaign"}},
			participants: []campaignapp.CampaignParticipant{{ID: "p-1", Name: "Owner", Role: "GM", CampaignAccess: "Owner"}},
			participant:  campaignapp.CampaignParticipant{ID: "p-1", Name: "Owner", Role: "GM", CampaignAccess: "Owner"},
			sessions:     []campaignapp.CampaignSession{{ID: "sess-1", Name: "Session One"}},
			invites:      []campaignapp.CampaignInvite{{ID: "inv-1", ParticipantID: "p-1", RecipientUserID: "user-123", Status: "Pending"}},
			characterCreationProgress: campaignapp.CampaignCharacterCreationProgress{
				Steps:    []campaignapp.CampaignCharacterCreationStep{{Step: 1, Key: "class_subclass", Complete: false}},
				NextStep: 1,
			},
			authorizationDecision: campaignapp.AuthorizationDecision{
				Evaluated:           true,
				Allowed:             true,
				ActorCampaignAccess: "Owner",
			},
		}, modulehandler.NewBase(func(*http.Request) string { return "user-123" }, nil, nil)),
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
		{name: "participant create", method: http.MethodGet, path: routepath.AppCampaignParticipantCreate("c1"), wantStatus: http.StatusOK},
		{name: "participant edit", method: http.MethodGet, path: routepath.AppCampaignParticipantEdit("c1", "p-1"), wantStatus: http.StatusOK},
		{name: "characters", method: http.MethodGet, path: routepath.AppCampaignCharacters("c1"), wantStatus: http.StatusOK},
		{name: "character create", method: http.MethodGet, path: routepath.AppCampaignCharacterCreate("c1"), wantStatus: http.StatusOK},
		{name: "character detail", method: http.MethodGet, path: routepath.AppCampaignCharacter("c1", "char-1"), wantStatus: http.StatusOK},
		{name: "character edit", method: http.MethodGet, path: routepath.AppCampaignCharacterEdit("c1", "char-1"), wantStatus: http.StatusOK},
		{name: "character control set", method: http.MethodPost, path: routepath.AppCampaignCharacterControl("c1", "char-1"), body: "participant_id=p-1", wantStatus: http.StatusFound},
		{name: "character control claim", method: http.MethodPost, path: routepath.AppCampaignCharacterControlClaim("c1", "char-1"), body: "", wantStatus: http.StatusFound},
		{name: "character control release", method: http.MethodPost, path: routepath.AppCampaignCharacterControlRelease("c1", "char-1"), body: "", wantStatus: http.StatusFound},
		{name: "character delete", method: http.MethodPost, path: routepath.AppCampaignCharacterDelete("c1", "char-1"), body: "", wantStatus: http.StatusFound},
		{name: "sessions", method: http.MethodGet, path: routepath.AppCampaignSessions("c1"), wantStatus: http.StatusOK},
		{name: "session create", method: http.MethodGet, path: routepath.AppCampaignSessionCreate("c1"), wantStatus: http.StatusOK},
		{name: "session detail", method: http.MethodGet, path: routepath.AppCampaignSession("c1", "sess-1"), wantStatus: http.StatusOK},
		{name: "invites", method: http.MethodGet, path: routepath.AppCampaignInvites("c1"), wantStatus: http.StatusOK},
		{name: "invite search", method: http.MethodPost, path: routepath.AppCampaignInviteSearch("c1"), body: `{"query":"al"}`, wantStatus: http.StatusOK},
		{name: "game", method: http.MethodGet, path: routepath.AppCampaignGame("c1"), wantStatus: http.StatusSeeOther},
		{name: "participant update", method: http.MethodPost, path: routepath.AppCampaignParticipantEdit("c1", "p-1"), body: "name=Owner&role=gm&pronouns=they%2Fthem", wantStatus: http.StatusFound},
		{name: "participant create submit", method: http.MethodPost, path: routepath.AppCampaignParticipantCreate("c1"), body: "name=Pending+Seat&role=player&campaign_access=member", wantStatus: http.StatusFound},
		{name: "character update", method: http.MethodPost, path: routepath.AppCampaignCharacterEdit("c1", "char-1"), body: "name=Hero&pronouns=they%2Fthem", wantStatus: http.StatusFound},
		{name: "campaign ai binding", method: http.MethodPost, path: routepath.AppCampaignAIBinding("c1"), body: "ai_agent_id=agent-1", wantStatus: http.StatusFound},
		{name: "campaign update", method: http.MethodPost, path: routepath.AppCampaignEdit("c1"), body: "name=Updated&theme_prompt=Theme&locale=en-US", wantStatus: http.StatusFound},
		{name: "session create submit", method: http.MethodPost, path: routepath.AppCampaignSessionCreate("c1"), body: "name=Session+Two", wantStatus: http.StatusFound},
		{name: "session end", method: http.MethodPost, path: routepath.AppCampaignSessionEnd("c1"), body: "session_id=sess-1", wantStatus: http.StatusFound},
		{name: "invite create", method: http.MethodPost, path: routepath.AppCampaignInviteCreate("c1"), body: "participant_id=p-1&username=alice", wantStatus: http.StatusFound},
		{name: "invite revoke", method: http.MethodPost, path: routepath.AppCampaignInviteRevoke("c1"), body: "invite_id=inv-1", wantStatus: http.StatusFound},
		{name: "rest route", method: http.MethodGet, path: routepath.AppCampaign("c1") + "/rest", wantStatus: http.StatusNotFound},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			if tc.method == http.MethodPost {
				if strings.HasPrefix(tc.body, "{") {
					req.Header.Set("Content-Type", "application/json")
				} else {
					req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				}
			}
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Fatalf("path %q status = %d, want %d", tc.path, rr.Code, tc.wantStatus)
			}
		})
	}
}
