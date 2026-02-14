package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/branding"
	"github.com/louisbranch/fracturing.space/internal/services/admin/i18n"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// TestWebPageRendering verifies layout rendering based on HTMX requests.
func TestWebPageRendering(t *testing.T) {
	handler := NewHandler(nil)

	tests := []struct {
		name        string
		path        string
		htmx        bool
		contains    []string
		notContains []string
	}{
		{
			name: "home full page",
			path: "/",
			contains: []string{
				"<!doctype html>",
				branding.AppName,
			},
			notContains: []string{
				"<h2>Campaigns</h2>",
			},
		},
		{
			name: "campaigns full page",
			path: "/campaigns",
			contains: []string{
				"<!doctype html>",
				branding.AppName,
				"<h2>Campaigns</h2>",
			},
		},
		{
			name: "campaigns htmx",
			path: "/campaigns",
			htmx: true,
			contains: []string{
				"<h2>Campaigns</h2>",
			},
			notContains: []string{
				"<!doctype html>",
				branding.AppName,
				"<html",
			},
		},
		{
			name: "campaign detail full page",
			path: "/campaigns/camp-123",
			contains: []string{
				"<!doctype html>",
				branding.AppName,
				"Campaign service unavailable.",
				"<h2>Campaign</h2>",
			},
		},
		{
			name: "campaign detail htmx",
			path: "/campaigns/camp-123",
			htmx: true,
			contains: []string{
				"Campaign service unavailable.",
				"<h2>Campaign</h2>",
			},
			notContains: []string{
				"<!doctype html>",
				branding.AppName,
				"<html",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://example.com"+tc.path, nil)
			if tc.htmx {
				req.Header.Set("HX-Request", "true")
			}
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
			}

			body := recorder.Body.String()
			for _, expected := range tc.contains {
				assertContains(t, body, expected)
			}
			for _, unexpected := range tc.notContains {
				assertNotContains(t, body, unexpected)
			}
		})
	}
}

// TestCampaignSessionsRoute verifies session routes render pages correctly.
func TestCampaignSessionsRoute(t *testing.T) {
	handler := NewHandler(nil)

	t.Run("sessions htmx", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-123/sessions", nil)
		req.Header.Set("HX-Request", "true")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
		}

		body := recorder.Body.String()
		assertContains(t, body, "<h3>Sessions</h3>")
		assertNotContains(t, body, "<!doctype html>")
	})

	t.Run("sessions full page", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-123/sessions", nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
		}

		body := recorder.Body.String()
		assertContains(t, body, "<!doctype html>")
		assertContains(t, body, "<h3>Sessions</h3>")
	})

	t.Run("sessions table htmx", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-123/sessions/table", nil)
		req.Header.Set("HX-Request", "true")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
		}

		body := recorder.Body.String()
		assertContains(t, body, "Session service unavailable.")
	})
}

type testClientProvider struct {
	auth     authv1.AuthServiceClient
	campaign statev1.CampaignServiceClient
	invite   statev1.InviteServiceClient
}

func (p testClientProvider) CampaignClient() statev1.CampaignServiceClient {
	return p.campaign
}

func (p testClientProvider) SessionClient() statev1.SessionServiceClient {
	return nil
}

func (p testClientProvider) CharacterClient() statev1.CharacterServiceClient {
	return nil
}

func (p testClientProvider) ParticipantClient() statev1.ParticipantServiceClient {
	return nil
}

func (p testClientProvider) InviteClient() statev1.InviteServiceClient {
	return p.invite
}

func (p testClientProvider) EventClient() statev1.EventServiceClient {
	return nil
}

func (p testClientProvider) SnapshotClient() statev1.SnapshotServiceClient {
	return nil
}

func (p testClientProvider) StatisticsClient() statev1.StatisticsServiceClient {
	return nil
}

func (p testClientProvider) AuthClient() authv1.AuthServiceClient {
	return p.auth
}

type testAuthClient struct {
	user          *authv1.User
	lastMetadata  metadata.MD
	lastUserIDReq string
}

func (c *testAuthClient) CreateUser(ctx context.Context, in *authv1.CreateUserRequest, opts ...grpc.CallOption) (*authv1.CreateUserResponse, error) {
	return &authv1.CreateUserResponse{User: c.user}, nil
}

func (c *testAuthClient) IssueJoinGrant(ctx context.Context, in *authv1.IssueJoinGrantRequest, opts ...grpc.CallOption) (*authv1.IssueJoinGrantResponse, error) {
	return &authv1.IssueJoinGrantResponse{}, nil
}

func (c *testAuthClient) GetUser(ctx context.Context, in *authv1.GetUserRequest, opts ...grpc.CallOption) (*authv1.GetUserResponse, error) {
	c.lastUserIDReq = in.GetUserId()
	md, _ := metadata.FromOutgoingContext(ctx)
	c.lastMetadata = md
	return &authv1.GetUserResponse{User: c.user}, nil
}

func (c *testAuthClient) ListUsers(ctx context.Context, in *authv1.ListUsersRequest, opts ...grpc.CallOption) (*authv1.ListUsersResponse, error) {
	return &authv1.ListUsersResponse{}, nil
}

type testCampaignClient struct {
	lastMetadata metadata.MD
	lastRequest  *statev1.CreateCampaignRequest
	response     *statev1.CreateCampaignResponse
}

func (c *testCampaignClient) CreateCampaign(ctx context.Context, in *statev1.CreateCampaignRequest, opts ...grpc.CallOption) (*statev1.CreateCampaignResponse, error) {
	c.lastRequest = in
	md, _ := metadata.FromOutgoingContext(ctx)
	c.lastMetadata = md
	if c.response != nil {
		return c.response, nil
	}
	return &statev1.CreateCampaignResponse{Campaign: &statev1.Campaign{Id: "camp-123"}}, nil
}

func (c *testCampaignClient) ListCampaigns(ctx context.Context, in *statev1.ListCampaignsRequest, opts ...grpc.CallOption) (*statev1.ListCampaignsResponse, error) {
	return &statev1.ListCampaignsResponse{}, nil
}

func (c *testCampaignClient) GetCampaign(ctx context.Context, in *statev1.GetCampaignRequest, opts ...grpc.CallOption) (*statev1.GetCampaignResponse, error) {
	return &statev1.GetCampaignResponse{}, nil
}

func (c *testCampaignClient) EndCampaign(ctx context.Context, in *statev1.EndCampaignRequest, opts ...grpc.CallOption) (*statev1.EndCampaignResponse, error) {
	return &statev1.EndCampaignResponse{}, nil
}

func (c *testCampaignClient) ArchiveCampaign(ctx context.Context, in *statev1.ArchiveCampaignRequest, opts ...grpc.CallOption) (*statev1.ArchiveCampaignResponse, error) {
	return &statev1.ArchiveCampaignResponse{}, nil
}

func (c *testCampaignClient) RestoreCampaign(ctx context.Context, in *statev1.RestoreCampaignRequest, opts ...grpc.CallOption) (*statev1.RestoreCampaignResponse, error) {
	return &statev1.RestoreCampaignResponse{}, nil
}

func TestCampaignCreateFlow(t *testing.T) {
	campaignClient := &testCampaignClient{}
	provider := testClientProvider{campaign: campaignClient}
	webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
	handler := webHandler.routes()

	form := url.Values{}
	form.Set("user_id", "user-123")
	form.Set("name", "New Campaign")
	form.Set("system", "daggerheart")
	form.Set("gm_mode", "human")
	form.Set("theme_prompt", "Misty marshes")
	form.Set("creator_display_name", "Owner")

	req := httptest.NewRequest(http.MethodPost, "http://example.com/campaigns/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://example.com")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, recorder.Code)
	}
	location := recorder.Header().Get("Location")
	if location != "/campaigns/camp-123" {
		t.Fatalf("expected redirect to /campaigns/camp-123, got %q", location)
	}
	values := campaignClient.lastMetadata.Get(grpcmeta.UserIDHeader)
	if len(values) != 1 || values[0] != "user-123" {
		t.Fatalf("expected metadata %s to be set, got %v", grpcmeta.UserIDHeader, values)
	}
	if campaignClient.lastRequest == nil {
		t.Fatalf("expected CreateCampaign request to be captured")
	}
	if campaignClient.lastRequest.GetName() != "New Campaign" {
		t.Fatalf("expected campaign name to be set")
	}
	if campaignClient.lastRequest.GetSystem() != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		t.Fatalf("expected system to be daggerheart")
	}
	if campaignClient.lastRequest.GetGmMode() != statev1.GmMode_HUMAN {
		t.Fatalf("expected gm mode to be human")
	}
}

func TestCampaignCreateHTMXRedirect(t *testing.T) {
	campaignClient := &testCampaignClient{}
	provider := testClientProvider{campaign: campaignClient}
	webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
	handler := webHandler.routes()

	form := url.Values{}
	form.Set("user_id", "user-htmx")
	form.Set("name", "HTMX Campaign")
	form.Set("system", "daggerheart")
	form.Set("gm_mode", "human")

	req := httptest.NewRequest(http.MethodPost, "http://example.com/campaigns/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("HX-Request", "true")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, recorder.Code)
	}
	location := recorder.Header().Get("Location")
	if location != "/campaigns/camp-123" {
		t.Fatalf("expected Location to /campaigns/camp-123, got %q", location)
	}
	redirect := recorder.Header().Get("HX-Redirect")
	if redirect != "/campaigns/camp-123" {
		t.Fatalf("expected HX-Redirect to /campaigns/camp-123, got %q", redirect)
	}
}

func TestCampaignCreateValidationErrors(t *testing.T) {
	campaignClient := &testCampaignClient{}
	provider := testClientProvider{campaign: campaignClient}
	webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
	handler := webHandler.routes()

	loc := i18n.Printer(i18n.Default())

	tests := []struct {
		name     string
		form     url.Values
		expected string
	}{
		{
			name: "empty system",
			form: url.Values{
				"user_id": []string{"user-123"},
				"name":    []string{"New Campaign"},
				"gm_mode": []string{"human"},
			},
			expected: loc.Sprintf("error.campaign_system_required"),
		},
		{
			name: "invalid system",
			form: url.Values{
				"user_id": []string{"user-123"},
				"name":    []string{"New Campaign"},
				"system":  []string{"bad"},
				"gm_mode": []string{"human"},
			},
			expected: loc.Sprintf("error.campaign_system_invalid"),
		},
		{
			name: "empty gm mode",
			form: url.Values{
				"user_id": []string{"user-123"},
				"name":    []string{"New Campaign"},
				"system":  []string{"daggerheart"},
			},
			expected: loc.Sprintf("error.campaign_gm_mode_required"),
		},
		{
			name: "invalid gm mode",
			form: url.Values{
				"user_id": []string{"user-123"},
				"name":    []string{"New Campaign"},
				"system":  []string{"daggerheart"},
				"gm_mode": []string{"robot"},
			},
			expected: loc.Sprintf("error.campaign_gm_mode_invalid"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "http://example.com/campaigns/create", strings.NewReader(tc.form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Origin", "http://example.com")
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
			}
			assertContains(t, recorder.Body.String(), tc.expected)
		})
	}
}

func TestCampaignCreateImpersonationOverridesUserID(t *testing.T) {
	campaignClient := &testCampaignClient{}
	provider := testClientProvider{campaign: campaignClient}
	webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
	handler := webHandler.routes()

	sessionID := "session-impersonate"
	webHandler.impersonation.Set(sessionID, impersonationSession{userID: "user-imp", displayName: "Impersonated"})

	form := url.Values{}
	form.Set("user_id", "user-other")
	form.Set("name", "Impersonated Campaign")
	form.Set("system", "daggerheart")
	form.Set("gm_mode", "human")
	form.Set("creator_display_name", "Other")

	req := httptest.NewRequest(http.MethodPost, "http://example.com/campaigns/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://example.com")
	req.AddCookie(&http.Cookie{Name: impersonationCookieName, Value: sessionID})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	values := campaignClient.lastMetadata.Get(grpcmeta.UserIDHeader)
	if len(values) != 1 || values[0] != "user-imp" {
		t.Fatalf("expected metadata %s to be impersonation user, got %v", grpcmeta.UserIDHeader, values)
	}
	if campaignClient.lastRequest == nil || campaignClient.lastRequest.GetCreatorDisplayName() != "Impersonated" {
		t.Fatalf("expected creator display name to be impersonation display name")
	}
}

func TestImpersonationFlow(t *testing.T) {
	user := &authv1.User{Id: "user-1", DisplayName: "Test User"}
	authClient := &testAuthClient{user: user}
	provider := testClientProvider{auth: authClient}
	webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
	handler := webHandler.routes()

	t.Run("csrf required", func(t *testing.T) {
		body := strings.NewReader("user_id=user-1")
		req := httptest.NewRequest(http.MethodPost, "http://example.com/users/impersonate", body)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("expected status %d, got %d", http.StatusForbidden, recorder.Code)
		}
	})

	t.Run("impersonate sets cookie and indicator", func(t *testing.T) {
		body := strings.NewReader("user_id=user-1")
		req := httptest.NewRequest(http.MethodPost, "http://example.com/users/impersonate", body)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", "http://example.com")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
		}
		cookies := recorder.Result().Cookies()
		var sessionCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == impersonationCookieName {
				sessionCookie = cookie
				break
			}
		}
		if sessionCookie == nil || sessionCookie.Value == "" {
			t.Fatalf("expected impersonation cookie to be set")
		}
		bodyText := recorder.Body.String()
		assertContains(t, bodyText, "Impersonating")
		assertContains(t, bodyText, "Test User")
	})

	t.Run("impersonate clears previous session", func(t *testing.T) {
		previousSessionID := "session-old"
		webHandler.impersonation.Set(previousSessionID, impersonationSession{userID: "user-old"})

		form := url.Values{}
		form.Set("user_id", "user-1")
		req := httptest.NewRequest(http.MethodPost, "http://example.com/users/impersonate", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", "http://example.com")
		req.AddCookie(&http.Cookie{Name: impersonationCookieName, Value: previousSessionID})
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		if _, ok := webHandler.impersonation.Get(previousSessionID); ok {
			t.Fatalf("expected previous session to be cleared")
		}
	})

	t.Run("logout clears cookie", func(t *testing.T) {
		sessionID := "session-logout"
		webHandler.impersonation.Set(sessionID, impersonationSession{userID: "user-logout"})

		req := httptest.NewRequest(http.MethodPost, "http://example.com/users/logout", nil)
		req.Header.Set("Origin", "http://example.com")
		req.AddCookie(&http.Cookie{Name: impersonationCookieName, Value: sessionID})
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
		}
		cookies := recorder.Result().Cookies()
		var clearedCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == impersonationCookieName {
				clearedCookie = cookie
				break
			}
		}
		if clearedCookie == nil || clearedCookie.MaxAge != -1 {
			t.Fatalf("expected impersonation cookie to be cleared")
		}
	})
}

func TestImpersonationMetadataInjection(t *testing.T) {
	user := &authv1.User{Id: "user-lookup", DisplayName: "Lookup"}
	authClient := &testAuthClient{user: user}
	provider := testClientProvider{auth: authClient}
	webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
	handler := webHandler.routes()

	sessionID := "session-meta"
	webHandler.impersonation.Set(sessionID, impersonationSession{userID: "user-impersonated"})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/users?user_id=user-lookup", nil)
	req.AddCookie(&http.Cookie{Name: impersonationCookieName, Value: sessionID})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	values := authClient.lastMetadata.Get(grpcmeta.UserIDHeader)
	if len(values) != 1 || values[0] != "user-impersonated" {
		t.Fatalf("expected metadata %s to be set, got %v", grpcmeta.UserIDHeader, values)
	}
}

// assertContains fails the test when the body lacks the expected fragment.
func assertContains(t *testing.T, body string, expected string) {
	t.Helper()
	if !strings.Contains(body, expected) {
		t.Fatalf("expected response to contain %q", expected)
	}
}

// assertNotContains fails the test when the body includes an unexpected fragment.
func assertNotContains(t *testing.T, body string, unexpected string) {
	t.Helper()
	if strings.Contains(body, unexpected) {
		t.Fatalf("expected response to not contain %q", unexpected)
	}
}

// TestEscapeAIP160StringLiteral verifies special character escaping.
func TestEscapeAIP160StringLiteral(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{`with"quote`, `with\"quote`},
		{`with\backslash`, `with\\backslash`},
		{`both\"chars`, `both\\\"chars`},
		{`a"b\c"d`, `a\"b\\c\"d`},
		{"", ""},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := escapeAIP160StringLiteral(tc.input)
			if result != tc.expected {
				t.Errorf("escapeAIP160StringLiteral(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestFormatActorType(t *testing.T) {
	loc := i18n.Printer(i18n.Default())

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty",
			input:    "",
			expected: "",
		},
		{
			name:     "system",
			input:    "system",
			expected: loc.Sprintf("filter.actor.system"),
		},
		{
			name:     "participant",
			input:    "participant",
			expected: loc.Sprintf("filter.actor.participant"),
		},
		{
			name:     "gm",
			input:    "gm",
			expected: loc.Sprintf("filter.actor.gm"),
		},
		{
			name:     "fallback",
			input:    "custom",
			expected: "custom",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := formatActorType(tc.input, loc)
			if result != tc.expected {
				t.Errorf("formatActorType(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}
