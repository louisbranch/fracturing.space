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

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type fakeWebCampaignClient struct {
	response       *statev1.ListCampaignsResponse
	listCalls      int
	listMetadata   metadata.MD
	createReq      *statev1.CreateCampaignRequest
	createMetadata metadata.MD
	createResp     *statev1.CreateCampaignResponse
}

func (f *fakeWebCampaignClient) ListCampaigns(ctx context.Context, _ *statev1.ListCampaignsRequest, _ ...grpc.CallOption) (*statev1.ListCampaignsResponse, error) {
	f.listCalls++
	md, _ := metadata.FromOutgoingContext(ctx)
	f.listMetadata = md
	if f.response != nil {
		return f.response, nil
	}
	return &statev1.ListCampaignsResponse{}, nil
}

func (f *fakeWebCampaignClient) CreateCampaign(ctx context.Context, req *statev1.CreateCampaignRequest, _ ...grpc.CallOption) (*statev1.CreateCampaignResponse, error) {
	md, _ := metadata.FromOutgoingContext(ctx)
	f.createMetadata = md
	f.createReq = req
	if f.createResp != nil {
		return f.createResp, nil
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (*fakeWebCampaignClient) GetCampaign(context.Context, *statev1.GetCampaignRequest, ...grpc.CallOption) (*statev1.GetCampaignResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (*fakeWebCampaignClient) EndCampaign(context.Context, *statev1.EndCampaignRequest, ...grpc.CallOption) (*statev1.EndCampaignResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (*fakeWebCampaignClient) ArchiveCampaign(context.Context, *statev1.ArchiveCampaignRequest, ...grpc.CallOption) (*statev1.ArchiveCampaignResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (*fakeWebCampaignClient) RestoreCampaign(context.Context, *statev1.RestoreCampaignRequest, ...grpc.CallOption) (*statev1.RestoreCampaignResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func TestAppCampaignsPageRedirectsToLoginWhenUnauthenticated(t *testing.T) {
	h := &handler{
		config:       Config{AuthBaseURL: "http://auth.local"},
		sessions:     newSessionStore(),
		pendingFlows: newPendingFlowStore(),
	}
	req := httptest.NewRequest(http.MethodGet, "/campaigns", nil)
	w := httptest.NewRecorder()

	h.handleAppCampaigns(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/auth/login" {
		t.Fatalf("location = %q, want %q", location, "/auth/login")
	}
}

func TestAppCampaignCreateRedirectsToLoginWhenUnauthenticated(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/campaigns/create", strings.NewReader("name=New+Campaign"))
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

func TestAppCampaignCreateGetRendersFormWithoutListingCampaigns(t *testing.T) {
	introspectCalls := 0
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		introspectCalls++
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

	campaignClient := &fakeWebCampaignClient{
		response: &statev1.ListCampaignsResponse{
			Campaigns: []*statev1.Campaign{
				{Id: "camp-1", Name: "Campaign One"},
			},
		},
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
	req := httptest.NewRequest(http.MethodGet, "/campaigns/create", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignCreate(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Create Campaign") {
		t.Fatalf("expected create campaign control in response")
	}
	if campaignClient.listCalls != 0 {
		t.Fatalf("list calls = %d, want %d", campaignClient.listCalls, 0)
	}
	if introspectCalls != 0 {
		t.Fatalf("introspect calls = %d, want %d", introspectCalls, 0)
	}
}

func TestAppCampaignCreateGetRendersFormWithoutUserLookup(t *testing.T) {
	campaignClient := &fakeWebCampaignClient{}
	h := &handler{
		config: Config{
			AuthBaseURL: "http://auth.local",
		},
		sessions:       newSessionStore(),
		pendingFlows:   newPendingFlowStore(),
		campaignClient: campaignClient,
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/campaigns/create", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignCreate(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Create Campaign") {
		t.Fatalf("expected create campaign control in response")
	}
	if campaignClient.listCalls != 0 {
		t.Fatalf("list calls = %d, want %d", campaignClient.listCalls, 0)
	}
}

func TestAppCampaignCreateGetUsesDashboardShell(t *testing.T) {
	campaignClient := &fakeWebCampaignClient{}
	h := &handler{
		config: Config{
			AuthBaseURL: "http://auth.local",
		},
		sessions:       newSessionStore(),
		pendingFlows:   newPendingFlowStore(),
		campaignClient: campaignClient,
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/campaigns/create", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignCreate(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Create Campaign") {
		t.Fatalf("expected create campaign control in response")
	}
	if !strings.Contains(body, "<nav class=\"navbar") {
		t.Fatalf("expected dashboard navbar shell in response")
	}
}

func TestAppCampaignCreateGetUsesConfiguredAppNameInShell(t *testing.T) {
	campaignClient := &fakeWebCampaignClient{}
	h := &handler{
		config: Config{
			AuthBaseURL: "http://auth.local",
			AppName:     "Custom Realm",
		},
		sessions:       newSessionStore(),
		pendingFlows:   newPendingFlowStore(),
		campaignClient: campaignClient,
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/campaigns/create", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignCreate(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Custom Realm") {
		t.Fatalf("expected configured app name in game shell")
	}
}

func TestAppCampaignsPageRendersUserScopedCampaigns(t *testing.T) {
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

	campaignClient := &fakeWebCampaignClient{
		response: &statev1.ListCampaignsResponse{
			Campaigns: []*statev1.Campaign{
				{Id: "camp-1", Name: "Campaign One"},
				{Id: "camp-2", Name: "Campaign Two"},
			},
		},
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
	req := httptest.NewRequest(http.MethodGet, "/campaigns", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaigns(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Campaign One") {
		t.Fatalf("expected campaign one in response")
	}
	if !strings.Contains(body, "Campaign Two") {
		t.Fatalf("expected campaign two in response")
	}
	if !strings.Contains(body, "Create Campaign") {
		t.Fatalf("expected create campaign control in response")
	}
	if !strings.Contains(body, "/campaigns/camp-1") {
		t.Fatalf("expected campaign detail link for camp-1 in response")
	}
	userIDs := campaignClient.listMetadata.Get(grpcmeta.UserIDHeader)
	if len(userIDs) != 1 || userIDs[0] != "user-123" {
		t.Fatalf("metadata %s = %v, want [user-123]", grpcmeta.UserIDHeader, userIDs)
	}
}

func TestAppCampaignCreateCallsCreateCampaignAndRedirects(t *testing.T) {
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

	campaignClient := &fakeWebCampaignClient{
		createResp: &statev1.CreateCampaignResponse{
			Campaign:         &statev1.Campaign{Id: "camp-777", Name: "New Campaign"},
			OwnerParticipant: &statev1.Participant{Id: "part-1"},
		},
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
	form := url.Values{
		"name":                 {"New Campaign"},
		"system":               {"daggerheart"},
		"gm_mode":              {"ai"},
		"theme_prompt":         {"Misty marshes"},
		"creator_display_name": {"Game Owner"},
		"user_id":              {"ignored-user-id"},
	}
	req := httptest.NewRequest(http.MethodPost, "/campaigns/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignCreate(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/campaigns/camp-777" {
		t.Fatalf("location = %q, want %q", location, "/campaigns/camp-777")
	}
	if campaignClient.createReq == nil {
		t.Fatalf("expected CreateCampaign request to be captured")
	}
	if campaignClient.createReq.GetName() != "New Campaign" {
		t.Fatalf("name = %q, want %q", campaignClient.createReq.GetName(), "New Campaign")
	}
	if campaignClient.createReq.GetSystem() != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		t.Fatalf("system = %v, want %v", campaignClient.createReq.GetSystem(), commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART)
	}
	if campaignClient.createReq.GetGmMode() != statev1.GmMode_AI {
		t.Fatalf("gm_mode = %v, want %v", campaignClient.createReq.GetGmMode(), statev1.GmMode_AI)
	}
	if campaignClient.createReq.GetThemePrompt() != "Misty marshes" {
		t.Fatalf("theme_prompt = %q, want %q", campaignClient.createReq.GetThemePrompt(), "Misty marshes")
	}
	if campaignClient.createReq.GetCreatorDisplayName() != "Game Owner" {
		t.Fatalf("creator_display_name = %q, want %q", campaignClient.createReq.GetCreatorDisplayName(), "Game Owner")
	}
	userIDs := campaignClient.createMetadata.Get(grpcmeta.UserIDHeader)
	if len(userIDs) != 1 || userIDs[0] != "user-123" {
		t.Fatalf("metadata %s = %v, want [user-123]", grpcmeta.UserIDHeader, userIDs)
	}
}

func TestAppCampaignCreateRejectsEmptyName(t *testing.T) {
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

	campaignClient := &fakeWebCampaignClient{}
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
	req := httptest.NewRequest(http.MethodPost, "/campaigns/create", strings.NewReader("name=   "))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignCreate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if campaignClient.createReq != nil {
		t.Fatalf("expected CreateCampaign not to be called for empty name")
	}
}

func TestAppCampaignCreateErrorPageUsesGameLayout(t *testing.T) {
	campaignClient := &fakeWebCampaignClient{}
	h := &handler{
		config: Config{
			AuthBaseURL: "http://auth.local",
		},
		sessions:       newSessionStore(),
		pendingFlows:   newPendingFlowStore(),
		campaignClient: campaignClient,
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	form := url.Values{
		"name":    {"New Campaign"},
		"system":  {"daggerheart"},
		"gm_mode": {"human"},
	}
	req := httptest.NewRequest(http.MethodPost, "/campaigns/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignCreate(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadGateway)
	}
	if !strings.Contains(w.Body.String(), "Campaign create unavailable") {
		t.Fatalf("expected campaign create error page title")
	}
	if !strings.Contains(w.Body.String(), `data-layout="game"`) {
		t.Fatalf("expected game layout marker in campaign create error page")
	}
}
