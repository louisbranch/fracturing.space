package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAppCampaignSessionsPageRedirectsUnauthenticatedToLogin(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/sessions", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/auth/login" {
		t.Fatalf("location = %q, want %q", location, "/auth/login")
	}
}

func TestAppCampaignSessionDetailRedirectsUnauthenticatedToLogin(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/sessions/sess-1", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/auth/login" {
		t.Fatalf("location = %q, want %q", location, "/auth/login")
	}
}

func TestAppCampaignSessionStartRedirectsUnauthenticatedToLogin(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/campaigns/camp-123/sessions/start", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/auth/login" {
		t.Fatalf("location = %q, want %q", location, "/auth/login")
	}
}

func TestAppCampaignSessionEndRedirectsUnauthenticatedToLogin(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/campaigns/camp-123/sessions/end", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/auth/login" {
		t.Fatalf("location = %q, want %q", location, "/auth/login")
	}
}

type fakeWebSessionClient struct {
	response *statev1.ListSessionsResponse
	lastReq  *statev1.ListSessionsRequest
	getReq   *statev1.GetSessionRequest
	getRes   *statev1.GetSessionResponse
	startReq *statev1.StartSessionRequest
	startMD  metadata.MD
	startRes *statev1.StartSessionResponse
	startErr error
	endReq   *statev1.EndSessionRequest
	endMD    metadata.MD
	endRes   *statev1.EndSessionResponse
	endErr   error
}

func (f *fakeWebSessionClient) StartSession(ctx context.Context, req *statev1.StartSessionRequest, _ ...grpc.CallOption) (*statev1.StartSessionResponse, error) {
	md, _ := metadata.FromOutgoingContext(ctx)
	f.startMD = md
	f.startReq = req
	if f.startErr != nil {
		return nil, f.startErr
	}
	if f.startRes != nil {
		return f.startRes, nil
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeWebSessionClient) ListSessions(_ context.Context, req *statev1.ListSessionsRequest, _ ...grpc.CallOption) (*statev1.ListSessionsResponse, error) {
	f.lastReq = req
	if f.response != nil {
		return f.response, nil
	}
	return &statev1.ListSessionsResponse{}, nil
}

func (f *fakeWebSessionClient) GetSession(_ context.Context, req *statev1.GetSessionRequest, _ ...grpc.CallOption) (*statev1.GetSessionResponse, error) {
	f.getReq = req
	if f.getRes != nil {
		return f.getRes, nil
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeWebSessionClient) EndSession(ctx context.Context, req *statev1.EndSessionRequest, _ ...grpc.CallOption) (*statev1.EndSessionResponse, error) {
	md, _ := metadata.FromOutgoingContext(ctx)
	f.endMD = md
	f.endReq = req
	if f.endErr != nil {
		return nil, f.endErr
	}
	if f.endRes != nil {
		return f.endRes, nil
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeWebSessionClient) OpenSessionGate(context.Context, *statev1.OpenSessionGateRequest, ...grpc.CallOption) (*statev1.OpenSessionGateResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeWebSessionClient) ResolveSessionGate(context.Context, *statev1.ResolveSessionGateRequest, ...grpc.CallOption) (*statev1.ResolveSessionGateResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeWebSessionClient) AbandonSessionGate(context.Context, *statev1.AbandonSessionGateRequest, ...grpc.CallOption) (*statev1.AbandonSessionGateResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeWebSessionClient) GetSessionSpotlight(context.Context, *statev1.GetSessionSpotlightRequest, ...grpc.CallOption) (*statev1.GetSessionSpotlightResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeWebSessionClient) SetSessionSpotlight(context.Context, *statev1.SetSessionSpotlightRequest, ...grpc.CallOption) (*statev1.SetSessionSpotlightResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeWebSessionClient) ClearSessionSpotlight(context.Context, *statev1.ClearSessionSpotlightRequest, ...grpc.CallOption) (*statev1.ClearSessionSpotlightResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func TestAppCampaignSessionsPageParticipantRendersSessions(t *testing.T) {
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
						Id:             "part-gm",
						CampaignId:     "camp-123",
						UserId:         "Alice",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER,
					},
				},
			},
		},
	}
	sessionClient := &fakeWebSessionClient{
		response: &statev1.ListSessionsResponse{
			Sessions: []*statev1.Session{
				{Id: "sess-1", CampaignId: "camp-123", Name: "Session One"},
				{Id: "sess-2", CampaignId: "camp-123", Name: "Session Two"},
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
		sessionClient: sessionClient,
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/sessions", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if sessionClient.lastReq == nil {
		t.Fatalf("expected ListSessions request to be captured")
	}
	if sessionClient.lastReq.GetCampaignId() != "camp-123" {
		t.Fatalf("campaign_id = %q, want %q", sessionClient.lastReq.GetCampaignId(), "camp-123")
	}
	body := w.Body.String()
	if !strings.Contains(body, "Session One") {
		t.Fatalf("expected Session One in response body")
	}
	if !strings.Contains(body, "Session Two") {
		t.Fatalf("expected Session Two in response body")
	}
	if !strings.Contains(body, "/campaigns/camp-123/sessions/sess-1") {
		t.Fatalf("expected session detail link for sess-1")
	}
}

func TestAppCampaignSessionDetailParticipantRendersSession(t *testing.T) {
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
						Id:             "part-gm",
						CampaignId:     "camp-123",
						UserId:         "Alice",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER,
					},
				},
			},
		},
	}
	sessionClient := &fakeWebSessionClient{
		getRes: &statev1.GetSessionResponse{
			Session: &statev1.Session{
				Id:         "sess-1",
				CampaignId: "camp-123",
				Name:       "Session One",
				Status:     statev1.SessionStatus_SESSION_ACTIVE,
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
		sessionClient: sessionClient,
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/sessions/sess-1", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if sessionClient.getReq == nil {
		t.Fatalf("expected GetSession request to be captured")
	}
	if sessionClient.getReq.GetCampaignId() != "camp-123" {
		t.Fatalf("campaign_id = %q, want %q", sessionClient.getReq.GetCampaignId(), "camp-123")
	}
	if sessionClient.getReq.GetSessionId() != "sess-1" {
		t.Fatalf("session_id = %q, want %q", sessionClient.getReq.GetSessionId(), "sess-1")
	}
	body := w.Body.String()
	if !strings.Contains(body, "Session One") {
		t.Fatalf("expected session name in response body")
	}
}

func TestAppCampaignSessionStartManagerCallsStartSession(t *testing.T) {
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
				},
			},
		},
	}
	sessionClient := &fakeWebSessionClient{
		startRes: &statev1.StartSessionResponse{
			Session: &statev1.Session{Id: "sess-new", CampaignId: "camp-123", Name: "Session Three"},
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
		sessionClient:     sessionClient,
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	form := url.Values{"name": {"Session Three"}}
	req := httptest.NewRequest(http.MethodPost, "/campaigns/camp-123/sessions/start", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/campaigns/camp-123/sessions" {
		t.Fatalf("location = %q, want %q", location, "/campaigns/camp-123/sessions")
	}
	if sessionClient.startReq == nil {
		t.Fatalf("expected StartSession request to be captured")
	}
	if sessionClient.startReq.GetCampaignId() != "camp-123" {
		t.Fatalf("campaign_id = %q, want %q", sessionClient.startReq.GetCampaignId(), "camp-123")
	}
	if sessionClient.startReq.GetName() != "Session Three" {
		t.Fatalf("name = %q, want %q", sessionClient.startReq.GetName(), "Session Three")
	}
	participantIDs := sessionClient.startMD.Get(grpcmeta.ParticipantIDHeader)
	if len(participantIDs) != 1 || participantIDs[0] != "part-manager" {
		t.Fatalf("metadata %s = %v, want [part-manager]", grpcmeta.ParticipantIDHeader, participantIDs)
	}
}

func TestAppCampaignSessionEndManagerCallsEndSession(t *testing.T) {
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
				},
			},
		},
	}
	sessionClient := &fakeWebSessionClient{
		endRes: &statev1.EndSessionResponse{
			Session: &statev1.Session{Id: "sess-1", CampaignId: "camp-123", Name: "Session One"},
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
		sessionClient:     sessionClient,
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	form := url.Values{"session_id": {"sess-1"}}
	req := httptest.NewRequest(http.MethodPost, "/campaigns/camp-123/sessions/end", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/campaigns/camp-123/sessions" {
		t.Fatalf("location = %q, want %q", location, "/campaigns/camp-123/sessions")
	}
	if sessionClient.endReq == nil {
		t.Fatalf("expected EndSession request to be captured")
	}
	if sessionClient.endReq.GetCampaignId() != "camp-123" {
		t.Fatalf("campaign_id = %q, want %q", sessionClient.endReq.GetCampaignId(), "camp-123")
	}
	if sessionClient.endReq.GetSessionId() != "sess-1" {
		t.Fatalf("session_id = %q, want %q", sessionClient.endReq.GetSessionId(), "sess-1")
	}
	participantIDs := sessionClient.endMD.Get(grpcmeta.ParticipantIDHeader)
	if len(participantIDs) != 1 || participantIDs[0] != "part-manager" {
		t.Fatalf("metadata %s = %v, want [part-manager]", grpcmeta.ParticipantIDHeader, participantIDs)
	}
}

func TestAppCampaignSessionStartMemberForbidden(t *testing.T) {
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
				},
			},
		},
	}
	sessionClient := &fakeWebSessionClient{}
	h := &handler{
		config: Config{
			AuthBaseURL:         authServer.URL,
			OAuthResourceSecret: "secret-1",
		},
		sessions:          newSessionStore(),
		pendingFlows:      newPendingFlowStore(),
		participantClient: participantClient,
		sessionClient:     sessionClient,
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	form := url.Values{"name": {"Session Three"}}
	req := httptest.NewRequest(http.MethodPost, "/campaigns/camp-123/sessions/start", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
	if sessionClient.startReq != nil {
		t.Fatalf("expected StartSession not to be called for member access")
	}
}

func TestAppCampaignSessionStartRequiresName(t *testing.T) {
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
				},
			},
		},
	}
	sessionClient := &fakeWebSessionClient{}
	h := &handler{
		config: Config{
			AuthBaseURL:         authServer.URL,
			OAuthResourceSecret: "secret-1",
		},
		sessions:          newSessionStore(),
		pendingFlows:      newPendingFlowStore(),
		participantClient: participantClient,
		sessionClient:     sessionClient,
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	form := url.Values{"name": {"   "}}
	req := httptest.NewRequest(http.MethodPost, "/campaigns/camp-123/sessions/start", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if sessionClient.startReq != nil {
		t.Fatalf("expected StartSession not to be called when name is empty")
	}
}

func TestAppCampaignSessionStartMapsInvalidArgumentToBadRequest(t *testing.T) {
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
				},
			},
		},
	}
	sessionClient := &fakeWebSessionClient{
		startErr: status.Error(codes.InvalidArgument, "invalid session"),
	}
	h := &handler{
		config: Config{
			AuthBaseURL:         authServer.URL,
			OAuthResourceSecret: "secret-1",
		},
		sessions:          newSessionStore(),
		pendingFlows:      newPendingFlowStore(),
		participantClient: participantClient,
		sessionClient:     sessionClient,
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	form := url.Values{"name": {"Session Three"}}
	req := httptest.NewRequest(http.MethodPost, "/campaigns/camp-123/sessions/start", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAppCampaignSessionsPageManagerShowsWriteControls(t *testing.T) {
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
				},
			},
		},
	}
	sessionClient := &fakeWebSessionClient{
		response: &statev1.ListSessionsResponse{
			Sessions: []*statev1.Session{
				{
					Id:         "sess-1",
					CampaignId: "camp-123",
					Name:       "Session One",
					Status:     statev1.SessionStatus_SESSION_ACTIVE,
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
		sessionClient:     sessionClient,
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/sessions", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Start Session") {
		t.Fatalf("expected start control in response body")
	}
	if !strings.Contains(body, "End Session") {
		t.Fatalf("expected end control in response body")
	}
}

func TestAppCampaignSessionsPageMemberHidesWriteControls(t *testing.T) {
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
				},
			},
		},
	}
	sessionClient := &fakeWebSessionClient{
		response: &statev1.ListSessionsResponse{
			Sessions: []*statev1.Session{
				{
					Id:         "sess-1",
					CampaignId: "camp-123",
					Name:       "Session One",
					Status:     statev1.SessionStatus_SESSION_ACTIVE,
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
		sessionClient:     sessionClient,
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/sessions", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if strings.Contains(body, "Start Session") {
		t.Fatalf("did not expect start control in response body")
	}
	if strings.Contains(body, "End Session") {
		t.Fatalf("did not expect end control in response body")
	}
}
