package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/platform/branding"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	campaignfeature "github.com/louisbranch/fracturing.space/internal/services/web/feature/campaign"
	webstorage "github.com/louisbranch/fracturing.space/internal/services/web/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeWebCampaignClient struct {
	response            *statev1.ListCampaignsResponse
	listCalls           int
	getCalls            int
	listMetadata        metadata.MD
	listMetadataByCall  []metadata.MD
	getReq              *statev1.GetCampaignRequest
	getMetadata         metadata.MD
	listResponsesByCall []*statev1.ListCampaignsResponse
	getResponse         *statev1.GetCampaignResponse
	getError            error
	createReq           *statev1.CreateCampaignRequest
	createMetadata      metadata.MD
	createResp          *statev1.CreateCampaignResponse
}

type fakeWebCacheStore struct {
	mu       sync.Mutex
	entries  map[string]webstorage.CacheEntry
	cursors  map[string]webstorage.CampaignEventCursor
	getCalls int
	putCalls int
}

func newFakeWebCacheStore() *fakeWebCacheStore {
	return &fakeWebCacheStore{
		entries: make(map[string]webstorage.CacheEntry),
		cursors: make(map[string]webstorage.CampaignEventCursor),
	}
}

func (f *fakeWebCacheStore) Close() error {
	return nil
}

func (f *fakeWebCacheStore) GetCacheEntry(_ context.Context, cacheKey string) (webstorage.CacheEntry, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.getCalls++
	entry, ok := f.entries[cacheKey]
	return entry, ok, nil
}

func (f *fakeWebCacheStore) PutCacheEntry(_ context.Context, entry webstorage.CacheEntry) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.putCalls++
	f.entries[entry.CacheKey] = entry
	return nil
}

func (f *fakeWebCacheStore) DeleteCacheEntry(_ context.Context, cacheKey string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.entries, cacheKey)
	return nil
}

func (f *fakeWebCacheStore) ListTrackedCampaignIDs(_ context.Context) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ids := make([]string, 0, len(f.cursors))
	for campaignID := range f.cursors {
		ids = append(ids, campaignID)
	}
	return ids, nil
}

func (f *fakeWebCacheStore) GetCampaignEventCursor(_ context.Context, campaignID string) (webstorage.CampaignEventCursor, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	cursor, ok := f.cursors[campaignID]
	return cursor, ok, nil
}

func (f *fakeWebCacheStore) PutCampaignEventCursor(_ context.Context, cursor webstorage.CampaignEventCursor) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cursors[cursor.CampaignID] = cursor
	return nil
}

func (f *fakeWebCacheStore) MarkCampaignScopeStale(context.Context, string, string, uint64, time.Time) error {
	return nil
}

func (f *fakeWebCampaignClient) ListCampaigns(ctx context.Context, _ *statev1.ListCampaignsRequest, _ ...grpc.CallOption) (*statev1.ListCampaignsResponse, error) {
	f.listCalls++
	md, _ := metadata.FromOutgoingContext(ctx)
	f.listMetadata = md
	if len(f.listMetadataByCall) < f.listCalls {
		f.listMetadataByCall = append(f.listMetadataByCall, md)
	} else {
		f.listMetadataByCall[f.listCalls-1] = md
	}
	if index := f.listCalls - 1; index < len(f.listResponsesByCall) && f.listResponsesByCall[index] != nil {
		return f.listResponsesByCall[index], nil
	}
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

func (f *fakeWebCampaignClient) GetCampaign(ctx context.Context, req *statev1.GetCampaignRequest, _ ...grpc.CallOption) (*statev1.GetCampaignResponse, error) {
	f.getCalls++
	md, _ := metadata.FromOutgoingContext(ctx)
	f.getMetadata = md
	f.getReq = req
	if f == nil {
		return nil, status.Error(codes.Unimplemented, "not implemented")
	}
	if f.getResponse != nil {
		return f.getResponse, nil
	}
	if f.getError != nil {
		return nil, f.getError
	}
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

func (*fakeWebCampaignClient) SetCampaignCover(context.Context, *statev1.SetCampaignCoverRequest, ...grpc.CallOption) (*statev1.SetCampaignCoverResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func TestAppCampaignsPageRedirectsToLoginWhenUnauthenticated(t *testing.T) {
	h := &handler{
		config:       Config{AuthBaseURL: "http://auth.local"},
		sessions:     newSessionStore(),
		pendingFlows: newPendingFlowStore(),
	}
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns", nil)
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
	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/create", strings.NewReader("name=New+Campaign"))
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
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/create", nil)
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
	if introspectCalls != 1 {
		t.Fatalf("introspect calls = %d, want %d", introspectCalls, 1)
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
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/create", nil)
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
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/create", nil)
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
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/create", nil)
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

func TestAppCampaignCreateGetUsesCreateCampaignTitle(t *testing.T) {
	campaignClient := &fakeWebCampaignClient{}
	h := &handler{
		config:         Config{AuthBaseURL: "http://auth.local"},
		sessions:       newSessionStore(),
		pendingFlows:   newPendingFlowStore(),
		campaignClient: campaignClient,
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/create", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignCreate(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Create Campaign</h1>") {
		t.Fatalf("expected create campaign heading in body")
	}
	if !strings.Contains(body, "<title>Create Campaign | "+branding.AppName+"</title>") {
		t.Fatalf("expected create campaign page title suffix")
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
				{
					Id:               "camp-1",
					Name:             "Campaign One",
					CoverAssetId:     "abandoned_castle_courtyard",
					ThemePrompt:      strings.Repeat("x", campaignfeature.CampaignThemePromptLimit+10),
					ParticipantCount: 12,
					CharacterCount:   7,
				},
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
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaigns(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, `class="grid grid-cols-1 md:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5 gap-4"`) {
		t.Fatalf("expected campaigns to render as card grid")
	}
	campaignActionIdx := strings.Index(body, "Start a new Campaign")
	if campaignActionIdx == -1 {
		t.Fatalf("expected create campaign action in response")
	}
	if !strings.Contains(body, "Campaign One") {
		t.Fatalf("expected campaign one in response")
	}
	if !strings.Contains(body, "Campaign Two") {
		t.Fatalf("expected campaign two in response")
	}
	if !strings.Contains(body, "/static/campaign-covers/abandoned_castle_courtyard.png") {
		t.Fatalf("expected campaign cover image URL in response")
	}
	campOneIdx := strings.Index(body, "/app/campaigns/camp-1")
	if campOneIdx == -1 {
		t.Fatalf("expected campaign detail link for camp-1 in response")
	}
	if !strings.Contains(body, `<a href="/app/campaigns/camp-1" class="group block"><img`) {
		t.Fatalf("expected campaign cover image link for camp-1")
	}
	if !strings.Contains(body, `<a href="/app/campaigns/camp-1">Campaign One</a>`) {
		t.Fatalf("expected campaign name link for camp-1")
	}
	expectedTheme := strings.Repeat("x", campaignfeature.CampaignThemePromptLimit) + "..."
	if !strings.Contains(body, `<p class="text-sm opacity-70">`+expectedTheme+`</p>`) {
		t.Fatalf("expected truncated campaign theme in response")
	}
	if !strings.Contains(body, `badge badge-outline">Participants: 12`) {
		t.Fatalf("expected participants badge in response")
	}
	if !strings.Contains(body, `badge badge-outline">Characters: 7`) {
		t.Fatalf("expected characters badge in response")
	}
	if campaignActionIdx > campOneIdx {
		t.Fatalf("expected campaign action to render before campaign list items")
	}
	if !strings.Contains(body, "/app/campaigns/create") {
		t.Fatalf("expected campaign create route in response")
	}
	if strings.Contains(body, "Campaign Name") {
		t.Fatalf("expected list view to not render the create form")
	}
	participantIDs := campaignClient.listMetadata.Get(grpcmeta.ParticipantIDHeader)
	if len(participantIDs) != 0 {
		t.Fatalf("metadata %s = %v, want []", grpcmeta.ParticipantIDHeader, participantIDs)
	}
	userIDs := campaignClient.listMetadata.Get(grpcmeta.UserIDHeader)
	if len(userIDs) != 1 || userIDs[0] != "user-123" {
		t.Fatalf("metadata %s = %v, want [user-123]", grpcmeta.UserIDHeader, userIDs)
	}
}

func TestAppCampaignsPageOrdersCampaignsNewestFirst(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns", nil)
	renderAppCampaignsPage(w, req, []*statev1.Campaign{
		{
			Id:        "camp-old",
			Name:      "Older Campaign",
			CreatedAt: timestamppb.New(time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)),
		},
		{
			Id:        "camp-new",
			Name:      "Newer Campaign",
			CreatedAt: timestamppb.New(time.Date(2025, 2, 3, 0, 0, 0, 0, time.UTC)),
		},
	})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	newerIdx := strings.Index(body, `href="/app/campaigns/camp-new"`)
	olderIdx := strings.Index(body, `href="/app/campaigns/camp-old"`)
	if newerIdx == -1 || olderIdx == -1 {
		t.Fatalf("expected both campaigns in response")
	}
	if newerIdx > olderIdx {
		t.Fatalf("expected newer campaign to render before older campaign")
	}
}

func TestCampaignsListPageRendersCreateButtonInHeadingRow(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns", nil)
	renderAppCampaignsPage(w, req, []*statev1.Campaign{
		{Id: "camp-1", Name: "Campaign One"},
	})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	headingRowIdx := strings.Index(body, `class="mb-5 flex items-center justify-between gap-3"`)
	if headingRowIdx == -1 {
		t.Fatalf("expected chrome heading row with flex alignment, got %q", body)
	}
	buttonIdx := strings.Index(body, `href="/app/campaigns/create"`)
	if buttonIdx == -1 {
		t.Fatalf("expected create campaign button in output")
	}
	gridIdx := strings.Index(body, `class="grid grid-cols-1 md:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5 gap-4"`)
	if gridIdx == -1 {
		t.Fatalf("expected campaigns grid in output")
	}
	if !(headingRowIdx < buttonIdx && buttonIdx < gridIdx) {
		t.Fatalf("expected create campaign button inside heading row before campaigns grid")
	}
}

func TestTruncateCampaignTheme(t *testing.T) {
	longTheme := strings.Repeat("x", campaignfeature.CampaignThemePromptLimit+1)
	if got := campaignfeature.TruncateCampaignTheme(longTheme); got != strings.Repeat("x", campaignfeature.CampaignThemePromptLimit)+"..." {
		t.Fatalf("campaignfeature.TruncateCampaignTheme(%q) = %q, want %q", longTheme, got, strings.Repeat("x", campaignfeature.CampaignThemePromptLimit)+"...")
	}
	if got := campaignfeature.TruncateCampaignTheme("Quiet dawn"); got != "Quiet dawn" {
		t.Fatalf("campaignfeature.TruncateCampaignTheme(%q) = %q, want %q", "Quiet dawn", got, "Quiet dawn")
	}
	if got := campaignfeature.TruncateCampaignTheme("  trimmed  "); got != "trimmed" {
		t.Fatalf("campaignfeature.TruncateCampaignTheme(%q) = %q, want %q", "  trimmed  ", got, "trimmed")
	}
}

func TestCampaignCoverImageURL_DefaultsToFirstPNGAsset(t *testing.T) {
	got := campaignfeature.CampaignCoverImageURL("", "", "", "")
	want := "/static/campaign-covers/abandoned_castle_courtyard.png"
	if got != want {
		t.Fatalf("campaignfeature.CampaignCoverImageURL(\"\") = %q, want %q", got, want)
	}
}

func TestCampaignCoverImageURL_UsesExternalAssetBaseURLWhenConfigured(t *testing.T) {
	got := campaignfeature.CampaignCoverImageURL(
		"https://cdn.example.com/assets",
		"camp-1",
		catalog.CampaignCoverSetV1,
		"abandoned_castle_courtyard",
	)
	want := "https://cdn.example.com/assets/abandoned_castle_courtyard.png"
	if got != want {
		t.Fatalf("campaignfeature.CampaignCoverImageURL(...) = %q, want %q", got, want)
	}
}

func TestCampaignCoverImageURL_UsesCloudinaryBasePathWithoutResize(t *testing.T) {
	got := campaignfeature.CampaignCoverImageURL(
		"https://res.cloudinary.com/fracturing-space/image/upload",
		"camp-1",
		catalog.CampaignCoverSetV1,
		"abandoned_castle_courtyard",
	)
	want := "https://res.cloudinary.com/fracturing-space/image/upload/abandoned_castle_courtyard.png"
	if got != want {
		t.Fatalf("campaignfeature.CampaignCoverImageURL(...) = %q, want %q", got, want)
	}
}

func TestAppCampaignsPageReturnsEmptyListWhenUserScopeHasNoCampaigns(t *testing.T) {
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
		listResponsesByCall: []*statev1.ListCampaignsResponse{
			{Campaigns: nil},
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
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaigns(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if strings.Contains(body, "Campaign One") {
		t.Fatalf("expected no campaign items when user scope is empty")
	}
	if strings.Contains(body, "Campaign Two") {
		t.Fatalf("expected no campaign items when user scope is empty")
	}
	if campaignClient.listCalls != 1 {
		t.Fatalf("list calls = %d, want %d", campaignClient.listCalls, 1)
	}
	participantIDs := campaignClient.listMetadataByCall[0].Get(grpcmeta.ParticipantIDHeader)
	if len(participantIDs) != 0 {
		t.Fatalf("metadata %s = %v, want []", grpcmeta.ParticipantIDHeader, participantIDs)
	}
	userIDs := campaignClient.listMetadataByCall[0].Get(grpcmeta.UserIDHeader)
	if len(userIDs) != 1 || userIDs[0] != "user-123" {
		t.Fatalf("metadata %s = %v, want [user-123]", grpcmeta.UserIDHeader, userIDs)
	}
	if introspectCalls != 1 {
		t.Fatalf("introspect calls = %d, want %d", introspectCalls, 1)
	}
}

func TestAppCampaignsPageRendersEmptyListWhenUserIdentityUnavailable(t *testing.T) {
	h := &handler{
		config:         Config{AuthBaseURL: "http://auth.local"},
		sessions:       newSessionStore(),
		pendingFlows:   newPendingFlowStore(),
		campaignClient: &fakeWebCampaignClient{},
		campaignAccess: nil,
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaigns(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Start a new Campaign") {
		t.Fatalf("expected empty list action in response")
	}
	if strings.Contains(body, "Campaigns unavailable") {
		t.Fatalf("expected campaign list route to avoid service-unavailable when user identity is missing")
	}
	if strings.Contains(body, "Campaign One") || strings.Contains(body, "Campaign Two") {
		t.Fatalf("expected list to be empty when user identity is missing")
	}
}

func TestAppCampaignsPageRendersEmptyListWhenCampaignServiceUnavailable(t *testing.T) {
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

	h := &handler{
		config: Config{
			AuthBaseURL:         authServer.URL,
			OAuthResourceSecret: "secret-1",
			GameAddr:            "127.0.0.1:1",
		},
		sessions:     newSessionStore(),
		pendingFlows: newPendingFlowStore(),
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaigns(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Start a new Campaign") {
		t.Fatalf("expected empty list action in response")
	}
	if strings.Contains(body, "Campaigns unavailable") {
		t.Fatalf("expected campaign list route to avoid service-unavailable when game is unreachable")
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
	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignCreate(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/app/campaigns/camp-777" {
		t.Fatalf("location = %q, want %q", location, "/app/campaigns/camp-777")
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

func TestAppCampaignCreateExpiresUserCampaignListCache(t *testing.T) {
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

	cacheStore := newFakeWebCacheStore()
	cacheKey := "campaign_list:user:" + strings.TrimSpace("user-123")
	cacheStore.entries[cacheKey] = webstorage.CacheEntry{
		CacheKey:     cacheKey,
		Scope:        cacheScopeCampaignSummary,
		UserID:       "user-123",
		PayloadBytes: []byte("cached"),
		ExpiresAt:    time.Now().Add(time.Minute),
	}

	campaignClient := &fakeWebCampaignClient{
		createResp: &statev1.CreateCampaignResponse{
			Campaign: &statev1.Campaign{Id: "camp-777", Name: "New Campaign"},
		},
	}
	h := &handler{
		config: Config{
			AuthBaseURL:         authServer.URL,
			OAuthResourceSecret: "secret-1",
		},
		sessions:       newSessionStore(),
		pendingFlows:   newPendingFlowStore(),
		cacheStore:     cacheStore,
		campaignClient: campaignClient,
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
		},
	}

	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	form := url.Values{
		"name": {"New Campaign"},
	}
	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignCreate(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if _, ok := cacheStore.entries[cacheKey]; ok {
		t.Fatalf("expected campaigns cache entry %q to be removed after campaign creation", cacheKey)
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
	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/create", strings.NewReader("name=   "))
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
	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/create", strings.NewReader(form.Encode()))
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

func TestCampaignDisplayNameCachesValues(t *testing.T) {
	campaignClient := &fakeWebCampaignClient{
		getResponse: &statev1.GetCampaignResponse{
			Campaign: &statev1.Campaign{
				Id:   "camp-123",
				Name: "Campaign One",
			},
		},
	}

	h := &handler{
		campaignNameCache: make(map[string]campaignNameCache),
		campaignClient:    campaignClient,
	}

	name := h.campaignDisplayName(context.Background(), "camp-123")
	if name != "Campaign One" {
		t.Fatalf("name = %q, want %q", name, "Campaign One")
	}
	if campaignClient.getCalls != 1 {
		t.Fatalf("get calls = %d, want %d", campaignClient.getCalls, 1)
	}

	name = h.campaignDisplayName(context.Background(), "camp-123")
	if name != "Campaign One" {
		t.Fatalf("name = %q, want %q", name, "Campaign One")
	}
	if campaignClient.getCalls != 1 {
		t.Fatalf("get calls = %d, want %d", campaignClient.getCalls, 1)
	}
}

func TestAppCampaignsPageCachesUserScopedCampaigns(t *testing.T) {
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

	cacheStore := newFakeWebCacheStore()
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
		cacheStore:     cacheStore,
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))

	req1 := httptest.NewRequest(http.MethodGet, "/app/campaigns", nil)
	req1.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w1 := httptest.NewRecorder()
	h.handleAppCampaigns(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("first status = %d, want %d", w1.Code, http.StatusOK)
	}
	if !strings.Contains(w1.Body.String(), "Campaign One") {
		t.Fatalf("expected campaign name on first render")
	}

	req2 := httptest.NewRequest(http.MethodGet, "/app/campaigns", nil)
	req2.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w2 := httptest.NewRecorder()
	h.handleAppCampaigns(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("second status = %d, want %d", w2.Code, http.StatusOK)
	}
	if !strings.Contains(w2.Body.String(), "Campaign One") {
		t.Fatalf("expected campaign name on second render")
	}
	if campaignClient.listCalls != 1 {
		t.Fatalf("list calls = %d, want %d", campaignClient.listCalls, 1)
	}
	if cacheStore.putCalls == 0 {
		t.Fatalf("expected cache store put calls")
	}
}

func TestCampaignDisplayNameUsesPersistentCache(t *testing.T) {
	cacheStore := newFakeWebCacheStore()
	campaignClient := &fakeWebCampaignClient{
		getResponse: &statev1.GetCampaignResponse{
			Campaign: &statev1.Campaign{
				Id:   "camp-123",
				Name: "Campaign One",
			},
		},
	}

	h := &handler{
		cacheStore:        cacheStore,
		campaignNameCache: make(map[string]campaignNameCache),
		campaignClient:    campaignClient,
	}

	first := h.campaignDisplayName(context.Background(), "camp-123")
	if first != "Campaign One" {
		t.Fatalf("first name = %q, want %q", first, "Campaign One")
	}
	if campaignClient.getCalls != 1 {
		t.Fatalf("first get calls = %d, want %d", campaignClient.getCalls, 1)
	}

	h.campaignNameCache = make(map[string]campaignNameCache)
	campaignClient.getResponse = nil
	campaignClient.getError = status.Error(codes.Internal, "upstream failure")

	second := h.campaignDisplayName(context.Background(), "camp-123")
	if second != "Campaign One" {
		t.Fatalf("second name = %q, want %q", second, "Campaign One")
	}
	if campaignClient.getCalls != 1 {
		t.Fatalf("second get calls = %d, want %d", campaignClient.getCalls, 1)
	}
}
