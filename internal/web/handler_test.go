package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/campaign/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/web/i18n"
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
				"Fracturing.Space",
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
				"Fracturing.Space",
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
				"Fracturing.Space",
				"<html",
			},
		},
		{
			name: "campaign detail full page",
			path: "/campaigns/camp-123",
			contains: []string{
				"<!doctype html>",
				"Fracturing.Space",
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
				"Fracturing.Space",
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
	auth authv1.AuthServiceClient
}

func (p testClientProvider) CampaignClient() statev1.CampaignServiceClient {
	return nil
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

func (p testClientProvider) EventClient() statev1.EventServiceClient {
	return nil
}

func (p testClientProvider) SnapshotClient() statev1.SnapshotServiceClient {
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

func (c *testAuthClient) GetUser(ctx context.Context, in *authv1.GetUserRequest, opts ...grpc.CallOption) (*authv1.GetUserResponse, error) {
	c.lastUserIDReq = in.GetUserId()
	md, _ := metadata.FromOutgoingContext(ctx)
	c.lastMetadata = md
	return &authv1.GetUserResponse{User: c.user}, nil
}

func (c *testAuthClient) ListUsers(ctx context.Context, in *authv1.ListUsersRequest, opts ...grpc.CallOption) (*authv1.ListUsersResponse, error) {
	return &authv1.ListUsersResponse{}, nil
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
