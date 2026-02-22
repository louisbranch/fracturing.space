package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/branding"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAppCampaignDetailPageRedirectsUnauthenticatedToLogin(t *testing.T) {
	h := &handler{
		config:       Config{AuthBaseURL: "http://auth.local"},
		sessions:     newSessionStore(),
		pendingFlows: newPendingFlowStore(),
	}
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123", nil)
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/auth/login" {
		t.Fatalf("location = %q, want %q", location, "/auth/login")
	}
}

func TestAppCampaignDetailCanonicalizesTrailingSlash(t *testing.T) {
	h := &handler{
		config:       Config{AuthBaseURL: "http://auth.local"},
		sessions:     newSessionStore(),
		pendingFlows: newPendingFlowStore(),
	}
	req := httptest.NewRequest(http.MethodGet, "/campaigns/", nil)
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMovedPermanently)
	}
	if location := w.Header().Get("Location"); location != "/campaigns" {
		t.Fatalf("location = %q, want %q", location, "/campaigns")
	}
}

func TestAppCampaignDetailPageForbiddenForNonParticipant(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/introspect" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/introspect")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(introspectResponse{Active: true, UserID: "Alice"})
	}))
	t.Cleanup(authServer.Close)
	campaignClient := &fakeWebCampaignClient{
		getError: status.Error(codes.PermissionDenied, "participant lacks permission"),
	}
	h := &handler{
		config: Config{
			AuthBaseURL:         authServer.URL,
			OAuthResourceSecret: "secret-1",
		},
		sessions:       newSessionStore(),
		pendingFlows:   newPendingFlowStore(),
		campaignClient: campaignClient,
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestAppCampaignDetailPageParticipantRendersCampaign(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/introspect" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/introspect")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(introspectResponse{Active: true, UserID: "Alice"})
	}))
	t.Cleanup(authServer.Close)
	participantClient := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants: []*statev1.Participant{
					{
						Id:             "participant-1",
						CampaignId:     "camp-123",
						UserId:         "Alice",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER,
					},
				},
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
		participantClient: participantClient,
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "camp-123") {
		t.Fatalf("expected campaign heading in body")
	}
	if !strings.Contains(body, "/campaigns/camp-123/sessions") {
		t.Fatalf("expected sessions link in body")
	}
	if !strings.Contains(body, "/campaigns/camp-123/participants") {
		t.Fatalf("expected participants link in body")
	}
	if !strings.Contains(body, "/campaigns/camp-123/characters") {
		t.Fatalf("expected characters link in body")
	}
	if !strings.Contains(body, "/campaigns/camp-123/invites") {
		t.Fatalf("expected invites link in body")
	}
	chatMenuLink := `<a class="menu-active" href="/campaigns/camp-123" hx-get="/campaigns/camp-123"`
	if !strings.Contains(body, chatMenuLink) {
		t.Fatalf("expected chat menu link to be active on campaign root")
	}
	if strings.Count(body, `class="card-title m-0"`) != 1 {
		t.Fatalf("expected chat page card title once, got %d", strings.Count(body, `class="card-title m-0"`))
	}
}

func TestAppCampaignDetailPagePropagatesUserMetadataToCampaignRead(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/introspect" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/introspect")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(introspectResponse{Active: true, UserID: "Alice"})
	}))
	t.Cleanup(authServer.Close)

	participantClient := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants: []*statev1.Participant{
					{
						Id:             "participant-1",
						CampaignId:     "camp-123",
						UserId:         "Alice",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER,
					},
				},
			},
		},
	}
	campaignClient := &fakeWebCampaignClient{
		getResponse: &statev1.GetCampaignResponse{
			Campaign: &statev1.Campaign{
				Id:   "camp-123",
				Name: "Campaign One",
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
		participantClient: participantClient,
		campaignClient:    campaignClient,
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if campaignClient.getReq == nil {
		t.Fatalf("expected GetCampaign request to be captured")
	}
	if campaignClient.getReq.GetCampaignId() != "camp-123" {
		t.Fatalf("campaign_id = %q, want %q", campaignClient.getReq.GetCampaignId(), "camp-123")
	}
	userIDs := campaignClient.getMetadata.Get(grpcmeta.UserIDHeader)
	if len(userIDs) != 1 || userIDs[0] != "Alice" {
		t.Fatalf("metadata %s = %v, want [Alice]", grpcmeta.UserIDHeader, userIDs)
	}
}

func TestAppCampaignDetailPageUsesCampaignNameFromService(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/introspect" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/introspect")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(introspectResponse{Active: true, UserID: "Alice"})
	}))
	t.Cleanup(authServer.Close)

	participantClient := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants: []*statev1.Participant{
					{
						Id:             "participant-1",
						CampaignId:     "camp-123",
						UserId:         "Alice",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER,
					},
				},
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
		participantClient: participantClient,
		campaignClient: &fakeWebCampaignClient{
			getResponse: &statev1.GetCampaignResponse{
				Campaign: &statev1.Campaign{
					Id:   "camp-123",
					Name: "Campaign One",
				},
			},
		},
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "<title>Campaign One | "+branding.AppName+"</title>") {
		t.Fatalf("expected campaign title in body")
	}
	if !strings.Contains(body, "Campaign One</h1>") {
		t.Fatalf("expected campaign heading in body")
	}
}

func TestAppCampaignDetailPageReturnsBadGatewayOnAccessCheckerError(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/introspect" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/introspect")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(introspectResponse{Active: true, UserID: "Alice"})
	}))
	t.Cleanup(authServer.Close)
	campaignClient := &fakeWebCampaignClient{
		getError: status.Error(codes.Unavailable, "upstream failure"),
	}
	h := &handler{
		config: Config{
			AuthBaseURL:         authServer.URL,
			OAuthResourceSecret: "secret-1",
		},
		sessions:       newSessionStore(),
		pendingFlows:   newPendingFlowStore(),
		campaignClient: campaignClient,
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestAppCampaignDetailPageRejectsInvalidPath(t *testing.T) {
	h := &handler{
		config:       Config{AuthBaseURL: "http://auth.local"},
		sessions:     newSessionStore(),
		pendingFlows: newPendingFlowStore(),
	}
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/extra", nil)
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestAppCampaignDetailPageRejectsNonGET(t *testing.T) {
	h := &handler{
		config:       Config{AuthBaseURL: "http://auth.local"},
		sessions:     newSessionStore(),
		pendingFlows: newPendingFlowStore(),
	}
	req := httptest.NewRequest(http.MethodPost, "/campaigns/camp-123", nil)
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
	if allow := w.Header().Get("Allow"); allow != http.MethodGet {
		t.Fatalf("Allow = %q, want %q", allow, http.MethodGet)
	}
}
