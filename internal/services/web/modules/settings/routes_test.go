package settings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	settingsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestRegisterRoutesHandlesNilMux(t *testing.T) {
	t.Parallel()

	svc := settingsapp.NewService(staticGateway{})
	registerRoutes(nil, newHandlers(svc, svc, svc, svc, svc, settingsSurfaceAvailability{
		profile:  true,
		locale:   true,
		security: true,
		aiKeys:   true,
		aiAgents: true,
	}, settingsTestBase(), requestmeta.SchemePolicy{}, nil))
}

func TestRegisterRoutesSettingsPathAndMethodContracts(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	svc := settingsapp.NewService(staticGateway{})
	registerRoutes(mux, newHandlers(svc, svc, svc, svc, svc, settingsSurfaceAvailability{
		profile:  true,
		locale:   true,
		security: true,
		aiKeys:   true,
		aiAgents: true,
	}, settingsTestBase(), requestmeta.SchemePolicy{}, nil))

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantAllow  string
	}{
		{name: "settings root", method: http.MethodGet, path: routepath.AppSettings, wantStatus: http.StatusFound},
		{name: "settings slash root", method: http.MethodGet, path: routepath.SettingsPrefix, wantStatus: http.StatusFound},
		{name: "profile get", method: http.MethodGet, path: routepath.AppSettingsProfile, wantStatus: http.StatusOK},
		{name: "profile head", method: http.MethodHead, path: routepath.AppSettingsProfile, wantStatus: http.StatusOK},
		{name: "security get", method: http.MethodGet, path: routepath.AppSettingsSecurity, wantStatus: http.StatusOK},
		{name: "ai agents get", method: http.MethodGet, path: routepath.AppSettingsAIAgents, wantStatus: http.StatusOK},
		{name: "profile delete rejected", method: http.MethodDelete, path: routepath.AppSettingsProfile, wantStatus: http.StatusMethodNotAllowed},
		{name: "passkey start get rejected", method: http.MethodGet, path: routepath.AppSettingsSecurityPasskeysStart, wantStatus: http.StatusMethodNotAllowed, wantAllow: http.MethodPost},
		{name: "passkey finish get rejected", method: http.MethodGet, path: routepath.AppSettingsSecurityPasskeysFinish, wantStatus: http.StatusMethodNotAllowed, wantAllow: http.MethodPost},
		{name: "ai key revoke get rejected", method: http.MethodGet, path: routepath.AppSettingsAIKeyRevoke("cred-1"), wantStatus: http.StatusMethodNotAllowed, wantAllow: http.MethodPost},
		{name: "ai agent delete get rejected", method: http.MethodGet, path: routepath.AppSettingsAIAgentDelete("agent-1"), wantStatus: http.StatusMethodNotAllowed, wantAllow: http.MethodPost},
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

	svc := settingsapp.NewService(staticGateway{})
	h := newHandlers(svc, svc, svc, svc, svc, settingsSurfaceAvailability{
		profile:  true,
		locale:   true,
		security: true,
		aiKeys:   true,
		aiAgents: true,
	}, settingsTestBase(), requestmeta.SchemePolicy{}, nil)
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

	svc := settingsapp.NewService(staticGateway{})
	h := newHandlers(svc, svc, svc, svc, svc, settingsSurfaceAvailability{
		profile:  true,
		locale:   true,
		security: true,
		aiKeys:   true,
		aiAgents: true,
	}, settingsTestBase(), requestmeta.SchemePolicy{}, nil)
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

func TestWithAgentIDDelegatesResolvedID(t *testing.T) {
	t.Parallel()

	svc := settingsapp.NewService(staticGateway{})
	h := newHandlers(svc, svc, svc, svc, svc, settingsSurfaceAvailability{
		profile:  true,
		locale:   true,
		security: true,
		aiKeys:   true,
		aiAgents: true,
	}, settingsTestBase(), requestmeta.SchemePolicy{}, nil)
	called := false
	var gotID string
	handler := h.withAgentID(func(_ http.ResponseWriter, _ *http.Request, agentID string) {
		called = true
		gotID = agentID
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.SetPathValue("agentID", " agent-1 ")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Fatalf("expected delegate to be called")
	}
	if gotID != "agent-1" {
		t.Fatalf("agentID = %q, want %q", gotID, "agent-1")
	}
}

// staticGateway returns canned settings data for route-level tests.
type staticGateway struct{}

func (staticGateway) LoadProfile(context.Context, string) (settingsapp.SettingsProfile, error) {
	return settingsapp.SettingsProfile{Username: "adventurer", Name: "Adventurer"}, nil
}

func (staticGateway) SaveProfile(context.Context, string, settingsapp.SettingsProfile) error {
	return nil
}

func (staticGateway) LoadLocale(context.Context, string) (string, error) {
	return "en-US", nil
}

func (staticGateway) SaveLocale(context.Context, string, string) error {
	return nil
}

func (staticGateway) ListPasskeys(context.Context, string) ([]settingsapp.SettingsPasskey, error) {
	return []settingsapp.SettingsPasskey{}, nil
}

func (staticGateway) BeginPasskeyRegistration(context.Context, string) (settingsapp.PasskeyChallenge, error) {
	return settingsapp.PasskeyChallenge{SessionID: "passkey-session-1", PublicKey: json.RawMessage(`{"publicKey":{}}`)}, nil
}

func (staticGateway) FinishPasskeyRegistration(context.Context, string, json.RawMessage) error {
	return nil
}

func (staticGateway) ListAIKeys(context.Context, string) ([]settingsapp.SettingsAIKey, error) {
	return []settingsapp.SettingsAIKey{}, nil
}

func (staticGateway) CreateAIKey(context.Context, string, string, string) error {
	return nil
}

func (staticGateway) ListAIAgentCredentials(context.Context, string) ([]settingsapp.SettingsAICredentialOption, error) {
	return []settingsapp.SettingsAICredentialOption{{ID: "cred-1", Label: "Primary", Provider: "OpenAI"}}, nil
}

func (staticGateway) ListAIAgents(context.Context, string) ([]settingsapp.SettingsAIAgent, error) {
	return []settingsapp.SettingsAIAgent{{ID: "agent-1", Label: "narrator", Provider: "OpenAI", Model: "gpt-4o-mini", AuthState: "Ready", CanDelete: true, CreatedAt: "2026-01-01 00:00 UTC"}}, nil
}

func (staticGateway) ListAIProviderModels(context.Context, string, string) ([]settingsapp.SettingsAIModelOption, error) {
	return []settingsapp.SettingsAIModelOption{{ID: "gpt-4o-mini", OwnedBy: "openai"}}, nil
}

func (staticGateway) CreateAIAgent(context.Context, string, settingsapp.CreateAIAgentInput) error {
	return nil
}

func (staticGateway) DeleteAIAgent(context.Context, string, string) error {
	return nil
}

func (staticGateway) RevokeAIKey(context.Context, string, string) error {
	return nil
}
