package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAppSettingsPageRendersPrimaryNavigation(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	h, err := NewHandler(defaultProtectedConfig(auth))
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
	assertPrimaryNavLinks(t, rr.Body.String())
}

func TestAppSettingsRootRedirectsToProfile(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	h, err := NewHandler(defaultProtectedConfig(auth))
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/settings/", nil)
	attachSessionCookie(t, req, auth, "user-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != "/app/settings/profile" {
		t.Fatalf("Location = %q, want %q", got, "/app/settings/profile")
	}
}

func TestAppSettingsProfileRendersSettingsMenuAndContent(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	h, err := NewHandler(defaultProtectedConfig(auth))
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	attachSessionCookie(t, req, auth, "user-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`<h1 class="mb-0">Settings</h1>`,
		`id="settings-profile"`,
		`href="/app/settings/profile"`,
		`href="/app/settings/locale"`,
		`href="/app/settings/ai-keys"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing settings marker %q: %q", marker, body)
		}
	}
}
