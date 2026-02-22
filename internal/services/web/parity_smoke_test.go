package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
)

func TestAppSignedInWorkspaceJourneySmoke(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/introspect" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/introspect")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(introspectResponse{
			Active:        true,
			UserID:        "user-123",
			ParticipantID: "part-manager",
		})
	}))
	t.Cleanup(authServer.Close)

	participantClient := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants: []*statev1.Participant{
					{
						Id:             "part-manager",
						CampaignId:     "camp-123",
						UserId:         "user-123",
						Name:           "Alice",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER,
					},
					{
						Id:             "part-member",
						CampaignId:     "camp-123",
						UserId:         "user-456",
						Name:           "Bob",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER,
					},
				},
			},
		},
	}
	campaignClient := &fakeWebCampaignClient{
		response: &statev1.ListCampaignsResponse{
			Campaigns: []*statev1.Campaign{
				{Id: "camp-123", Name: "Skyfall"},
			},
		},
		getResponse: &statev1.GetCampaignResponse{
			Campaign: &statev1.Campaign{Id: "camp-123", Name: "Skyfall"},
		},
	}
	sessionClient := &fakeWebSessionClient{
		response: &statev1.ListSessionsResponse{
			Sessions: []*statev1.Session{
				{Id: "sess-1", CampaignId: "camp-123", Name: "Session One"},
			},
		},
	}
	characterClient := &fakeWebCharacterClient{
		response: &statev1.ListCharactersResponse{
			Characters: []*statev1.Character{
				{Id: "char-1", CampaignId: "camp-123", Name: "Mira"},
			},
		},
	}
	inviteClient := &fakeWebInviteClient{
		response: &statev1.ListPendingInvitesForUserResponse{
			Invites: []*statev1.PendingUserInvite{
				{
					Campaign:    &statev1.Campaign{Id: "camp-123", Name: "Skyfall"},
					Participant: &statev1.Participant{Id: "part-member", Name: "Bob"},
					Invite:      &statev1.Invite{Id: "inv-user-1", CampaignId: "camp-123"},
				},
			},
		},
		listInvitesResp: &statev1.ListInvitesResponse{
			Invites: []*statev1.Invite{
				{Id: "inv-1", CampaignId: "camp-123", RecipientUserId: "user-999"},
			},
		},
	}
	h := &handler{
		config: Config{
			AuthBaseURL:         authServer.URL,
			OAuthResourceSecret: "secret-1",
		},
		sessions:          newSessionStore(),
		pendingFlows:      newPendingFlowStore(),
		campaignClient:    campaignClient,
		sessionClient:     sessionClient,
		participantClient: participantClient,
		characterClient:   characterClient,
		inviteClient:      inviteClient,
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))

	appReq := httptest.NewRequest(http.MethodGet, "/app", nil)
	appReq.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	appResp := httptest.NewRecorder()
	h.handleAppHome(appResp, appReq)
	if appResp.Code != http.StatusFound {
		t.Fatalf("/app status = %d, want %d", appResp.Code, http.StatusFound)
	}
	if location := appResp.Header().Get("Location"); location != "/app/campaigns" {
		t.Fatalf("/app location = %q, want %q", location, "/app/campaigns")
	}

	campaignsReq := httptest.NewRequest(http.MethodGet, "/app/campaigns", nil)
	campaignsReq.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	campaignsResp := httptest.NewRecorder()
	h.handleAppCampaigns(campaignsResp, campaignsReq)
	if campaignsResp.Code != http.StatusOK {
		t.Fatalf("/app/campaigns status = %d, want %d", campaignsResp.Code, http.StatusOK)
	}
	if body := campaignsResp.Body.String(); !strings.Contains(body, "/app/campaigns/camp-123") {
		t.Fatalf("expected campaign detail link on /campaigns")
	}
	if body := campaignsResp.Body.String(); !strings.Contains(body, `data-layout="game"`) {
		t.Fatalf("expected game layout marker on /campaigns")
	}

	createReq := httptest.NewRequest(http.MethodGet, "/app/campaigns/create", nil)
	createReq.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	createResp := httptest.NewRecorder()
	h.handleAppCampaignCreate(createResp, createReq)
	if createResp.Code != http.StatusOK {
		t.Fatalf("/app/campaigns/create status = %d, want %d", createResp.Code, http.StatusOK)
	}
	if body := createResp.Body.String(); !strings.Contains(body, `data-layout="game"`) {
		t.Fatalf("expected game layout marker on /campaigns/create")
	}

	overviewReq := httptest.NewRequest(http.MethodGet, "/app/campaigns/camp-123", nil)
	overviewReq.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	overviewResp := httptest.NewRecorder()
	h.handleAppCampaignDetail(overviewResp, overviewReq)
	if overviewResp.Code != http.StatusOK {
		t.Fatalf("/app/campaigns/camp-123 status = %d, want %d", overviewResp.Code, http.StatusOK)
	}
	overviewBody := overviewResp.Body.String()
	for _, link := range []string{
		"/app/campaigns/camp-123/sessions",
		"/app/campaigns/camp-123/participants",
		"/app/campaigns/camp-123/characters",
		"/app/campaigns/camp-123/invites",
	} {
		if !strings.Contains(overviewBody, link) {
			t.Fatalf("expected overview link %q", link)
		}
	}
	if !strings.Contains(overviewBody, `data-layout="game"`) {
		t.Fatalf("expected game layout marker on /campaigns/camp-123")
	}

	invitesReq := httptest.NewRequest(http.MethodGet, "/app/invites", nil)
	invitesReq.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	invitesResp := httptest.NewRecorder()
	h.handleAppInvites(invitesResp, invitesReq)
	if invitesResp.Code != http.StatusOK {
		t.Fatalf("/app/invites status = %d, want %d", invitesResp.Code, http.StatusOK)
	}
	if body := invitesResp.Body.String(); !strings.Contains(body, `data-layout="game"`) {
		t.Fatalf("expected game layout marker on /invites")
	}

	paths := map[string]string{
		"/app/campaigns/camp-123/sessions":     "Sessions",
		"/app/campaigns/camp-123/participants": "Participants",
		"/app/campaigns/camp-123/characters":   "Characters",
		"/app/campaigns/camp-123/invites":      "Campaign Invites",
	}
	for path, marker := range paths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
		w := httptest.NewRecorder()
		h.handleAppCampaignDetail(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want %d", path, w.Code, http.StatusOK)
		}
		if body := w.Body.String(); !strings.Contains(body, marker) {
			t.Fatalf("expected %q in response body for %s", marker, path)
		}
		if body := w.Body.String(); !strings.Contains(body, `data-layout="game"`) {
			t.Fatalf("expected game layout marker for %s", path)
		}
	}

	legacyReq := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123", nil)
	legacyResp := httptest.NewRecorder()
	NewHandler(Config{AuthBaseURL: authServer.URL}, nil).ServeHTTP(legacyResp, legacyReq)
	if legacyResp.Code != http.StatusNotFound {
		t.Fatalf("/campaigns/camp-123 status = %d, want %d", legacyResp.Code, http.StatusNotFound)
	}
}
