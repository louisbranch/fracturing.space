package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules"
)

func TestNewServerRequiresHTTPAddr(t *testing.T) {
	t.Parallel()

	_, err := NewServer(context.Background(), Config{})
	if err == nil {
		t.Fatalf("expected error for empty HTTPAddr")
	}
}

func TestNewHandlerMountsOnlyStableModulesByDefault(t *testing.T) {
	t.Parallel()

	h, err := NewHandler(Config{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	publicReq := httptest.NewRequest(http.MethodGet, "/discover/campaigns", nil)
	publicRR := httptest.NewRecorder()
	h.ServeHTTP(publicRR, publicReq)
	if publicRR.Code != http.StatusOK {
		t.Fatalf("public status = %d, want %d", publicRR.Code, http.StatusOK)
	}

	publicProfileReq := httptest.NewRequest(http.MethodGet, "/u/alice", nil)
	publicProfileRR := httptest.NewRecorder()
	h.ServeHTTP(publicProfileRR, publicProfileReq)
	if publicProfileRR.Code != http.StatusServiceUnavailable {
		t.Fatalf("public profile status = %d, want %d", publicProfileRR.Code, http.StatusServiceUnavailable)
	}

	protectedReq := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	protectedRR := httptest.NewRecorder()
	h.ServeHTTP(protectedRR, protectedReq)
	if protectedRR.Code != http.StatusFound {
		t.Fatalf("protected status = %d, want %d", protectedRR.Code, http.StatusFound)
	}

	dashboardReq := httptest.NewRequest(http.MethodGet, "/app/dashboard/", nil)
	dashboardRR := httptest.NewRecorder()
	h.ServeHTTP(dashboardRR, dashboardReq)
	if dashboardRR.Code != http.StatusFound {
		t.Fatalf("dashboard status = %d, want %d", dashboardRR.Code, http.StatusFound)
	}

	dashboardNoSlashReq := httptest.NewRequest(http.MethodGet, "/app/dashboard", nil)
	dashboardNoSlashRR := httptest.NewRecorder()
	h.ServeHTTP(dashboardNoSlashRR, dashboardNoSlashReq)
	if dashboardNoSlashRR.Code != http.StatusTemporaryRedirect {
		t.Fatalf("dashboard (no slash) status = %d, want %d", dashboardNoSlashRR.Code, http.StatusTemporaryRedirect)
	}

	campaignsReq := httptest.NewRequest(http.MethodGet, "/app/campaigns/123", nil)
	campaignsRR := httptest.NewRecorder()
	h.ServeHTTP(campaignsRR, campaignsReq)
	if campaignsRR.Code != http.StatusFound {
		t.Fatalf("campaigns status = %d, want %d", campaignsRR.Code, http.StatusFound)
	}
	if got := campaignsRR.Header().Get("Location"); got != "/login" {
		t.Fatalf("campaigns redirect = %q, want %q", got, "/login")
	}
	if got := dashboardRR.Header().Get("Location"); got != "/login" {
		t.Fatalf("dashboard redirect = %q, want %q", got, "/login")
	}
	if got := dashboardNoSlashRR.Header().Get("Location"); got != "/app/dashboard/" {
		t.Fatalf("dashboard (no slash) redirect = %q, want %q", got, "/app/dashboard/")
	}

	notificationsReq := httptest.NewRequest(http.MethodGet, "/app/notifications/", nil)
	notificationsRR := httptest.NewRecorder()
	h.ServeHTTP(notificationsRR, notificationsReq)
	if notificationsRR.Code != http.StatusFound {
		t.Fatalf("notifications status = %d, want %d", notificationsRR.Code, http.StatusFound)
	}
	if got := notificationsRR.Header().Get("Location"); got != "/login" {
		t.Fatalf("notifications redirect = %q, want %q", got, "/login")
	}
}

func TestDefaultCampaignSurfaceExposesDetailAndMutationRoutes(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	h, err := NewHandler(defaultProtectedConfig(auth))
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	for _, path := range []string{
		"/app/campaigns/c1/sessions",
		"/app/campaigns/c1/sessions/sess-1",
		"/app/campaigns/c1/invites",
		"/app/campaigns/c1/game",
		"/app/campaigns/c1/characters/char-1",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		attachSessionCookie(t, req, auth, "user-1")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("path %q status = %d, want %d", path, rr.Code, http.StatusOK)
		}
	}

	for _, tc := range []struct {
		path     string
		body     string
		wantPath string
	}{
		{path: "/app/campaigns/c1/sessions/start", body: "name=Session+One", wantPath: "/app/campaigns/c1/sessions"},
		{path: "/app/campaigns/c1/sessions/end", body: "session_id=sess-1", wantPath: "/app/campaigns/c1/sessions"},
		{path: "/app/campaigns/c1/invites/create", body: "participant_id=p1&username=alice", wantPath: "/app/campaigns/c1/invites"},
		{path: "/app/campaigns/c1/invites/revoke", body: "invite_id=inv-1", wantPath: "/app/campaigns/c1/invites"},
	} {
		req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", "http://example.com")
		attachSessionCookie(t, req, auth, "user-1")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusFound {
			t.Fatalf("path %q status = %d, want %d", tc.path, rr.Code, http.StatusFound)
		}
		if got := rr.Header().Get("Location"); got != tc.wantPath {
			t.Fatalf("path %q Location = %q, want %q", tc.path, got, tc.wantPath)
		}
	}

	createReq := httptest.NewRequest(http.MethodPost, "/app/campaigns/c1/characters/create", strings.NewReader("name=Hero&kind=pc"))
	createReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	createReq.Header.Set("Origin", "http://example.com")
	attachSessionCookie(t, createReq, auth, "user-1")
	createRR := httptest.NewRecorder()
	h.ServeHTTP(createRR, createReq)
	if createRR.Code == http.StatusNotFound {
		t.Fatalf("stable character create route unexpectedly hidden")
	}
}

func TestProtectedRouteDoesNotTrustUserHeader(t *testing.T) {
	t.Parallel()

	h, err := NewHandler(Config{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	req.Header.Set("X-Web-User", "user-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != "/login" {
		t.Fatalf("Location = %q, want %q", got, "/login")
	}
}

func TestNewHandlerAddsRequestIDHeader(t *testing.T) {
	t.Parallel()

	h, err := NewHandler(Config{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/discover/campaigns", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if got := rr.Header().Get("X-Request-ID"); got == "" {
		t.Fatalf("expected response request id header")
	}
}

func TestNewHandlerUsesConfiguredCampaignClient(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	h, err := NewHandler(Config{
		Dependencies: newDependencyBundle(
			PrincipalDependencies{SessionClient: auth},
			modules.Dependencies{
				PublicAuth: modules.PublicAuthDependencies{AuthClient: auth},
				Campaigns: modules.CampaignDependencies{
					CampaignClient:           fakeCampaignClient{response: &statev1.ListCampaignsResponse{Campaigns: []*statev1.Campaign{{Id: "c1", Name: "Remote"}}}},
					CommunicationClient:      defaultCommunicationClient(),
					ParticipantClient:        defaultParticipantClient(),
					CharacterClient:          defaultCharacterClient(),
					DaggerheartContentClient: defaultDaggerheartContentClient(),
					DaggerheartAssetClient:   defaultDaggerheartAssetClient(),
					SessionClient:            defaultSessionClient(),
					InviteClient:             defaultInviteClient(),
					AuthClient:               auth,
					AuthorizationClient:      defaultAuthorizationClient(),
				},
			},
		),
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/", nil)
	attachSessionCookie(t, req, auth, "user-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Remote") {
		t.Fatalf("body = %q, want configured campaign response", body)
	}
}

func TestNewServerBuildsHTTPServer(t *testing.T) {
	t.Parallel()

	srv, err := NewServer(context.Background(), Config{
		HTTPAddr: "127.0.0.1:0",
		Dependencies: newDependencyBundle(
			PrincipalDependencies{},
			modules.Dependencies{PublicAuth: modules.PublicAuthDependencies{AuthClient: newFakeWebAuthClient()}},
		),
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	if srv.httpAddr != "127.0.0.1:0" {
		t.Fatalf("httpAddr = %q, want %q", srv.httpAddr, "127.0.0.1:0")
	}
	if srv.httpServer == nil {
		t.Fatalf("expected http server")
	}
	srv.Close()
}

func TestListenAndServeRejectsNilServer(t *testing.T) {
	t.Parallel()

	var srv *Server
	err := srv.ListenAndServe(context.Background())
	if err == nil {
		t.Fatalf("expected nil server error")
	}
	if !strings.Contains(err.Error(), "web server is nil") {
		t.Fatalf("error = %q, want nil server message", err.Error())
	}
}

func TestListenAndServeRequiresContext(t *testing.T) {
	t.Parallel()

	srv := &Server{httpServer: &http.Server{Addr: "127.0.0.1:0", Handler: http.NotFoundHandler()}}
	err := srv.ListenAndServe(nil)
	if err == nil {
		t.Fatalf("expected context-required error")
	}
	if !strings.Contains(err.Error(), "context is required") {
		t.Fatalf("error = %q, want context-required message", err.Error())
	}
}

func TestListenAndServeReturnsServeError(t *testing.T) {
	t.Parallel()

	srv := &Server{httpServer: &http.Server{Addr: "bad address", Handler: http.NotFoundHandler()}}
	err := srv.ListenAndServe(context.Background())
	if err == nil {
		t.Fatalf("expected serve error")
	}
	if !strings.Contains(err.Error(), "serve web http") {
		t.Fatalf("error = %q, want wrapped serve message", err.Error())
	}
}

func TestListenAndServeShutsDownOnContextCancel(t *testing.T) {
	t.Parallel()

	srv := &Server{httpServer: &http.Server{Addr: "127.0.0.1:0", Handler: http.NotFoundHandler()}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("ListenAndServe() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for server shutdown")
	}
}

func TestCloseHandlesNilServerAndNilHTTPServer(t *testing.T) {
	t.Parallel()

	var nilServer *Server
	nilServer.Close()

	(&Server{}).Close()
}
