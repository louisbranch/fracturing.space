package settings

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	settingsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestRegisterRoutesHandlesNilMux(t *testing.T) {
	t.Parallel()

	registerRoutes(nil, newHandlers(settingsapp.NewService(staticGateway{}), settingsTestBase(), requestmeta.SchemePolicy{}, nil))
}

func TestRegisterRoutesSettingsPathAndMethodContracts(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	registerRoutes(mux, newHandlers(settingsapp.NewService(staticGateway{}), settingsTestBase(), requestmeta.SchemePolicy{}, nil))

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantAllow  string
	}{
		{name: "settings root", method: http.MethodGet, path: routepath.AppSettings, wantStatus: http.StatusFound},
		{name: "settings slash root", method: http.MethodGet, path: routepath.SettingsPrefix, wantStatus: http.StatusFound},
		{name: "profile required redirect", method: http.MethodGet, path: routepath.AppSettingsProfileRequired, wantStatus: http.StatusFound},
		{name: "profile get", method: http.MethodGet, path: routepath.AppSettingsProfile, wantStatus: http.StatusOK},
		{name: "profile head", method: http.MethodHead, path: routepath.AppSettingsProfile, wantStatus: http.StatusOK},
		{name: "ai agents get", method: http.MethodGet, path: routepath.AppSettingsAIAgents, wantStatus: http.StatusOK},
		{name: "profile delete rejected", method: http.MethodDelete, path: routepath.AppSettingsProfile, wantStatus: http.StatusMethodNotAllowed},
		{name: "ai key revoke get rejected", method: http.MethodGet, path: routepath.AppSettingsAIKeyRevoke("cred-1"), wantStatus: http.StatusMethodNotAllowed, wantAllow: http.MethodPost},
		{name: "unknown subpath", method: http.MethodGet, path: routepath.SettingsPrefix + "unknown", wantStatus: http.StatusNotFound},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
			if tc.wantAllow != "" {
				if got := rr.Header().Get("Allow"); got != tc.wantAllow {
					t.Fatalf("Allow = %q, want %q", got, tc.wantAllow)
				}
			}
		})
	}
}

func TestWithCredentialIDReturnsNotFoundForMissingPathValue(t *testing.T) {
	t.Parallel()

	h := newHandlers(settingsapp.NewService(staticGateway{}), settingsTestBase(), requestmeta.SchemePolicy{}, nil)
	called := false
	handler := h.withCredentialID(func(http.ResponseWriter, *http.Request, string) {
		called = true
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if called {
		t.Fatalf("expected delegate not to be called")
	}
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestWithCredentialIDDelegatesResolvedID(t *testing.T) {
	t.Parallel()

	h := newHandlers(settingsapp.NewService(staticGateway{}), settingsTestBase(), requestmeta.SchemePolicy{}, nil)
	called := false
	var gotID string
	handler := h.withCredentialID(func(_ http.ResponseWriter, _ *http.Request, credentialID string) {
		called = true
		gotID = credentialID
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.SetPathValue("credentialID", " cred-1 ")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Fatalf("expected delegate to be called")
	}
	if gotID != "cred-1" {
		t.Fatalf("credentialID = %q, want %q", gotID, "cred-1")
	}
}

// staticGateway returns canned settings data for route-level tests.
type staticGateway struct{}

func (staticGateway) LoadProfile(context.Context, string) (SettingsProfile, error) {
	return SettingsProfile{Username: "adventurer", Name: "Adventurer"}, nil
}

func (staticGateway) SaveProfile(context.Context, string, SettingsProfile) error {
	return nil
}

func (staticGateway) LoadLocale(context.Context, string) (string, error) {
	return "en-US", nil
}

func (staticGateway) SaveLocale(context.Context, string, string) error {
	return nil
}

func (staticGateway) ListAIKeys(context.Context, string) ([]SettingsAIKey, error) {
	return []SettingsAIKey{}, nil
}

func (staticGateway) CreateAIKey(context.Context, string, string, string) error {
	return nil
}

func (staticGateway) ListAIAgentCredentials(context.Context, string) ([]SettingsAICredentialOption, error) {
	return []SettingsAICredentialOption{{ID: "cred-1", Label: "Primary", Provider: "OpenAI"}}, nil
}

func (staticGateway) ListAIAgents(context.Context, string) ([]SettingsAIAgent, error) {
	return []SettingsAIAgent{{ID: "agent-1", Name: "Narrator", Provider: "OpenAI", Model: "gpt-4o-mini", Status: "Active", CreatedAt: "2026-01-01 00:00 UTC"}}, nil
}

func (staticGateway) ListAIProviderModels(context.Context, string, string) ([]SettingsAIModelOption, error) {
	return []SettingsAIModelOption{{ID: "gpt-4o-mini", OwnedBy: "openai"}}, nil
}

func (staticGateway) CreateAIAgent(context.Context, string, CreateAIAgentInput) error {
	return nil
}

func (staticGateway) RevokeAIKey(context.Context, string, string) error {
	return nil
}
