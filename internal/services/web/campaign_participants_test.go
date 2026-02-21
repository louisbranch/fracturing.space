package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
)

func TestAppCampaignParticipantsPageRedirectsUnauthenticatedToLogin(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/participants", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/auth/login" {
		t.Fatalf("location = %q, want %q", location, "/auth/login")
	}
}

func TestAppCampaignParticipantUpdateRedirectsUnauthenticatedToLogin(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/campaigns/camp-123/participants/update", strings.NewReader("participant_id=part-1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/auth/login" {
		t.Fatalf("location = %q, want %q", location, "/auth/login")
	}
}

func TestAppCampaignParticipantsPageParticipantRendersParticipants(t *testing.T) {
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
					{Id: "part-1", CampaignId: "camp-123", UserId: "Alice", Name: "Alice"},
					{Id: "part-2", CampaignId: "camp-123", UserId: "Bob", Name: "Bob"},
				},
			},
		},
	}
	h := &handler{
		config: Config{
			AuthBaseURL:         authServer.URL,
			OAuthResourceSecret: "secret-1",
		},
		sessions:     newSessionStore(),
		pendingFlows: newPendingFlowStore(),
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
		participantClient: participantClient,
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/participants", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if len(participantClient.calls) != 2 {
		t.Fatalf("ListParticipants calls = %d, want 2", len(participantClient.calls))
	}
	if participantClient.calls[0].GetCampaignId() != "camp-123" {
		t.Fatalf("campaign_id = %q, want %q", participantClient.calls[0].GetCampaignId(), "camp-123")
	}
	body := w.Body.String()
	if !strings.Contains(body, "Alice") {
		t.Fatalf("expected Alice in response body")
	}
	if !strings.Contains(body, "Bob") {
		t.Fatalf("expected Bob in response body")
	}
}

func TestAppCampaignParticipantUpdateManagerCallsUpdateParticipant(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/introspect" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/introspect")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(introspectResponse{
			Active: true,
			UserID: "user-123",
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
					{Id: "part-2", CampaignId: "camp-123", UserId: "user-456", Name: "Bob"},
				},
			},
		},
		updateResp: &statev1.UpdateParticipantResponse{
			Participant: &statev1.Participant{
				Id:             "part-2",
				CampaignId:     "camp-123",
				Role:           statev1.ParticipantRole_PLAYER,
				CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER,
				Controller:     statev1.Controller_CONTROLLER_AI,
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
	form := url.Values{
		"participant_id":  {"part-2"},
		"campaign_access": {"manager"},
		"role":            {"player"},
		"controller":      {"ai"},
	}
	req := httptest.NewRequest(http.MethodPost, "/campaigns/camp-123/participants/update", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/campaigns/camp-123/participants" {
		t.Fatalf("location = %q, want %q", location, "/campaigns/camp-123/participants")
	}
	if participantClient.updateReq == nil {
		t.Fatalf("expected UpdateParticipant request to be captured")
	}
	if participantClient.updateReq.GetCampaignId() != "camp-123" {
		t.Fatalf("campaign_id = %q, want %q", participantClient.updateReq.GetCampaignId(), "camp-123")
	}
	if participantClient.updateReq.GetParticipantId() != "part-2" {
		t.Fatalf("participant_id = %q, want %q", participantClient.updateReq.GetParticipantId(), "part-2")
	}
	if participantClient.updateReq.GetCampaignAccess() != statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER {
		t.Fatalf("campaign_access = %v, want %v", participantClient.updateReq.GetCampaignAccess(), statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER)
	}
	if participantClient.updateReq.GetRole() != statev1.ParticipantRole_PLAYER {
		t.Fatalf("role = %v, want %v", participantClient.updateReq.GetRole(), statev1.ParticipantRole_PLAYER)
	}
	if participantClient.updateReq.GetController() != statev1.Controller_CONTROLLER_AI {
		t.Fatalf("controller = %v, want %v", participantClient.updateReq.GetController(), statev1.Controller_CONTROLLER_AI)
	}
	participantIDs := participantClient.updateMD.Get(grpcmeta.ParticipantIDHeader)
	if len(participantIDs) != 1 || participantIDs[0] != "part-manager" {
		t.Fatalf("metadata %s = %v, want [part-manager]", grpcmeta.ParticipantIDHeader, participantIDs)
	}
}

func TestAppCampaignParticipantUpdateMemberForbidden(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/introspect" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/introspect")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(introspectResponse{
			Active: true,
			UserID: "user-123",
		})
	}))
	t.Cleanup(authServer.Close)

	participantClient := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants: []*statev1.Participant{
					{
						Id:             "part-member",
						CampaignId:     "camp-123",
						UserId:         "user-123",
						Name:           "Alice",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER,
					},
					{Id: "part-2", CampaignId: "camp-123", UserId: "user-456", Name: "Bob"},
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
	form := url.Values{
		"participant_id":  {"part-2"},
		"campaign_access": {"manager"},
		"role":            {"player"},
		"controller":      {"ai"},
	}
	req := httptest.NewRequest(http.MethodPost, "/campaigns/camp-123/participants/update", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
	if participantClient.updateReq != nil {
		t.Fatalf("expected UpdateParticipant not to be called for member access")
	}
}

func TestAppCampaignParticipantsPageManagerShowsUpdateControls(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/introspect" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/introspect")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(introspectResponse{
			Active: true,
			UserID: "user-123",
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
					{Id: "part-2", CampaignId: "camp-123", UserId: "user-456", Name: "Bob"},
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
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/participants", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Update Access") {
		t.Fatalf("expected update access control in response body")
	}
	if !strings.Contains(body, "name=\"role\"") {
		t.Fatalf("expected role control in response body")
	}
	if !strings.Contains(body, "name=\"controller\"") {
		t.Fatalf("expected controller control in response body")
	}
}

func TestAppCampaignParticipantsPageMemberHidesUpdateControls(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/introspect" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/introspect")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(introspectResponse{
			Active: true,
			UserID: "user-123",
		})
	}))
	t.Cleanup(authServer.Close)

	participantClient := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants: []*statev1.Participant{
					{
						Id:             "part-member",
						CampaignId:     "camp-123",
						UserId:         "user-123",
						Name:           "Alice",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER,
					},
					{Id: "part-2", CampaignId: "camp-123", UserId: "user-456", Name: "Bob"},
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
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/participants", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if strings.Contains(body, "Update Access") {
		t.Fatalf("did not expect update access control in response body")
	}
	if strings.Contains(body, "name=\"role\"") {
		t.Fatalf("did not expect role control in response body")
	}
	if strings.Contains(body, "name=\"controller\"") {
		t.Fatalf("did not expect controller control in response body")
	}
}
