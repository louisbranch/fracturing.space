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
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/camp-123/participants", nil)
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
	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/camp-123/participants/update", strings.NewReader("participant_id=part-1"))
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
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/camp-123/participants", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if len(participantClient.calls) != 1 {
		t.Fatalf("ListParticipants calls = %d, want 1", len(participantClient.calls))
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

func TestAppCampaignParticipantsPagePropagatesUserMetadataToParticipantReads(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/introspect" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/introspect")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(introspectResponse{Active: true, UserID: "user-123"})
	}))
	t.Cleanup(authServer.Close)
	participantClient := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants: []*statev1.Participant{
					{
						Id:             "part-1",
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
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/camp-123/participants", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if len(participantClient.listMDByCall) == 0 {
		t.Fatalf("expected participant list metadata to be captured")
	}
	for i, md := range participantClient.listMDByCall {
		userIDs := md.Get(grpcmeta.UserIDHeader)
		if len(userIDs) != 1 || userIDs[0] != "user-123" {
			t.Fatalf("call %d metadata %s = %v, want [user-123]", i+1, grpcmeta.UserIDHeader, userIDs)
		}
	}
}

func TestAppCampaignParticipantsPageCachesCampaignParticipants(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/introspect" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/introspect")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(introspectResponse{Active: true, UserID: "Alice"})
	}))
	t.Cleanup(authServer.Close)

	cacheStore := newFakeWebCacheStore()
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
		cacheStore:   cacheStore,
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
		participantClient: participantClient,
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))

	req1 := httptest.NewRequest(http.MethodGet, "/app/campaigns/camp-123/participants", nil)
	req1.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w1 := httptest.NewRecorder()
	h.handleAppCampaignDetail(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("first status = %d, want %d", w1.Code, http.StatusOK)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/app/campaigns/camp-123/participants", nil)
	req2.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w2 := httptest.NewRecorder()
	h.handleAppCampaignDetail(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("second status = %d, want %d", w2.Code, http.StatusOK)
	}
	if !strings.Contains(w2.Body.String(), "Campaign One") {
		t.Fatalf("expected campaign name in cached response body")
	}

	if len(participantClient.calls) != 1 {
		t.Fatalf("ListParticipants calls = %d, want %d", len(participantClient.calls), 1)
	}
	if cacheStore.putCalls == 0 {
		t.Fatalf("expected cache store put calls")
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
	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/camp-123/participants/update", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/app/campaigns/camp-123/participants" {
		t.Fatalf("location = %q, want %q", location, "/app/campaigns/camp-123/participants")
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
	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/camp-123/participants/update", strings.NewReader(form.Encode()))
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
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/camp-123/participants", nil)
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
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/camp-123/participants", nil)
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
