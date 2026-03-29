package settings

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	flashnotice "github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSettingsAvailabilityHelpersCoverAllSurfaces(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		availability settingsSurfaceAvailability
		wantAny      bool
		wantPath     string
	}{
		{name: "none", availability: settingsSurfaceAvailability{}, wantAny: false, wantPath: ""},
		{name: "profile", availability: settingsSurfaceAvailability{profile: true}, wantAny: true, wantPath: routepath.AppSettingsProfile},
		{name: "locale", availability: settingsSurfaceAvailability{locale: true}, wantAny: true, wantPath: routepath.AppSettingsLocale},
		{name: "security", availability: settingsSurfaceAvailability{security: true}, wantAny: true, wantPath: routepath.AppSettingsSecurity},
		{name: "ai keys", availability: settingsSurfaceAvailability{aiKeys: true}, wantAny: true, wantPath: routepath.AppSettingsAIKeys},
		{name: "ai agents", availability: settingsSurfaceAvailability{aiAgents: true}, wantAny: true, wantPath: routepath.AppSettingsAIAgents},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.availability.anyAvailable(); got != tc.wantAny {
				t.Fatalf("anyAvailable() = %v, want %v", got, tc.wantAny)
			}
			if got := tc.availability.defaultPath(); got != tc.wantPath {
				t.Fatalf("defaultPath() = %q, want %q", got, tc.wantPath)
			}
		})
	}
}

func TestSettingsComposeWrappersBuildExpectedModule(t *testing.T) {
	t.Parallel()

	check := func(t *testing.T, built module.Module) {
		t.Helper()
		if got := built.ID(); got != "settings" {
			t.Fatalf("ID() = %q, want %q", got, "settings")
		}
		mount, err := built.Mount()
		if err != nil {
			t.Fatalf("Mount() error = %v", err)
		}
		if got := mount.Prefix; got != routepath.SettingsPrefix {
			t.Fatalf("prefix = %q, want %q", got, routepath.SettingsPrefix)
		}
		if !mount.CanonicalRoot {
			t.Fatal("CanonicalRoot = false, want true")
		}
	}

	check(t, Compose(CompositionConfig{
		Base:             settingsTestBase(),
		SocialClient:     &socialClientStub{},
		AccountClient:    &accountClientStub{},
		PasskeyClient:    &passkeyClientStub{},
		CredentialClient: &credentialClientStub{},
		AgentClient:      &agentClientStub{},
	}))
	check(t, ComposeProtected(ProtectedSurfaceOptions{Base: settingsTestBase()}, Dependencies{
		SocialClient:     &socialClientStub{},
		AccountClient:    &accountClientStub{},
		PasskeyClient:    &passkeyClientStub{},
		CredentialClient: &credentialClientStub{},
		AgentClient:      &agentClientStub{},
	}))
}

func TestSettingsBindDependenciesGuardNilInputsAndAssignClients(t *testing.T) {
	t.Parallel()

	BindAuthDependency(nil, new(grpc.ClientConn))
	BindSocialDependency(nil, new(grpc.ClientConn))
	BindAIDependency(nil, new(grpc.ClientConn))

	deps := &Dependencies{}
	BindAuthDependency(deps, nil)
	BindSocialDependency(deps, nil)
	BindAIDependency(deps, nil)
	if deps.AccountClient != nil || deps.PasskeyClient != nil || deps.SocialClient != nil || deps.CredentialClient != nil || deps.AgentClient != nil {
		t.Fatalf("deps unexpectedly populated after nil binds: %+v", deps)
	}

	conn := new(grpc.ClientConn)
	BindAuthDependency(deps, conn)
	BindSocialDependency(deps, conn)
	BindAIDependency(deps, conn)
	if deps.AccountClient == nil || deps.PasskeyClient == nil || deps.SocialClient == nil || deps.CredentialClient == nil || deps.AgentClient == nil {
		t.Fatalf("deps missing bound clients: %+v", deps)
	}
}

func TestSettingsSecurityPasskeyRoutes(t *testing.T) {
	t.Parallel()

	module := newSettingsModuleFromGateways(newPopulatedFakeGateway(), nil, settingsTestBase(), withFlashMeta(requestmeta.SchemePolicy{}))
	mount, err := module.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	t.Run("start requires same origin", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodPost, routepath.AppSettingsSecurityPasskeysStart, nil)
		rr := httptest.NewRecorder()
		mount.Handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
		}
	})

	t.Run("start returns typed json", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodPost, "http://app.example.test"+routepath.AppSettingsSecurityPasskeysStart, nil)
		req.Host = "app.example.test"
		req.Header.Set("Origin", "http://app.example.test")
		rr := httptest.NewRecorder()
		mount.Handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
		}
		var payload struct {
			SessionID string          `json:"session_id"`
			PublicKey json.RawMessage `json:"public_key"`
		}
		if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if payload.SessionID != "passkey-session-1" {
			t.Fatalf("session_id = %q, want %q", payload.SessionID, "passkey-session-1")
		}
		if strings.TrimSpace(string(payload.PublicKey)) != `{"publicKey":{}}` {
			t.Fatalf("public_key = %s", payload.PublicKey)
		}
	})

	t.Run("finish rejects invalid json with localized error", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodPost, "http://app.example.test"+routepath.AppSettingsSecurityPasskeysFinish, strings.NewReader(`{"session_id":`))
		req.Host = "app.example.test"
		req.Header.Set("Origin", "http://app.example.test")
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		mount.Handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
		if got := rr.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
			t.Fatalf("Content-Type = %q, want JSON", got)
		}
	})

	t.Run("finish returns redirect json and flash cookie", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodPost, "http://app.example.test"+routepath.AppSettingsSecurityPasskeysFinish, strings.NewReader(`{"session_id":"passkey-session-1","credential":{"id":"cred-1"}}`))
		req.Host = "app.example.test"
		req.Header.Set("Origin", "http://app.example.test")
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		mount.Handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
		}
		var payload map[string]string
		if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if payload["redirect_url"] != routepath.AppSettingsSecurity {
			t.Fatalf("redirect_url = %q, want %q", payload["redirect_url"], routepath.AppSettingsSecurity)
		}
		if !responseHasCookieName(rr, flashnotice.CookieName) {
			t.Fatalf("response missing %q cookie", flashnotice.CookieName)
		}
	})
}

func TestHandleAIAgentDeleteRedirectsAndWritesFlash(t *testing.T) {
	t.Parallel()

	gateway := newPopulatedFakeGateway()
	module := newSettingsModuleFromGateways(gateway, nil, settingsTestBase())
	mount, err := module.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, routepath.AppSettingsAIAgentDelete("agent-1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppSettingsAIAgents {
		t.Fatalf("Location = %q, want %q", got, routepath.AppSettingsAIAgents)
	}
	if gateway.lastDeletedAgentID != "agent-1" {
		t.Fatalf("deleted agent id = %q, want %q", gateway.lastDeletedAgentID, "agent-1")
	}
	if !responseHasCookieName(rr, flashnotice.CookieName) {
		t.Fatalf("response missing %q cookie", flashnotice.CookieName)
	}
}

func TestHandleAIAgentDeleteConflictRedirectsWithErrorFlash(t *testing.T) {
	t.Parallel()

	gateway := newPopulatedFakeGateway()
	gateway.deleteAIAgentErr = apperrors.EK(apperrors.KindConflict, "web.settings.ai_agents.error_active", "agent is active")
	module := newSettingsModuleFromGateways(gateway, nil, settingsTestBase())
	mount, err := module.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, routepath.AppSettingsAIAgentDelete("agent-1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppSettingsAIAgents {
		t.Fatalf("Location = %q, want %q", got, routepath.AppSettingsAIAgents)
	}
	if !responseHasCookieName(rr, flashnotice.CookieName) {
		t.Fatalf("response missing %q cookie", flashnotice.CookieName)
	}
}

func TestHandleAIAgentsGetCredentialModelErrorRendersBadRequest(t *testing.T) {
	t.Parallel()

	gateway := newPopulatedFakeGateway()
	gateway.listModelsErr = status.Error(codes.InvalidArgument, "bad credential")
	module := newSettingsModuleFromGateways(gateway, nil, settingsTestBase())
	mount, err := module.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppSettingsAIAgents+"?credential_id=cred-1", nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	if gateway.lastSelectedCredentialID != "cred-1" {
		t.Fatalf("credential id = %q, want %q", gateway.lastSelectedCredentialID, "cred-1")
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="settings-ai-agents"`) {
		t.Fatalf("body missing ai agents marker: %q", body)
	}
}

func TestRedirectSettingsRootFallsClosedWhenNoSurfaceAvailable(t *testing.T) {
	t.Parallel()

	module := New(Config{Base: settingsTestBase()})
	mount, err := module.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppSettings, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestMountAIKeysCreateConflictRendersConflictAndPreservesForm(t *testing.T) {
	t.Parallel()

	gateway := newPopulatedFakeGateway()
	gateway.createAIKeyErr = apperrors.EK(apperrors.KindConflict, "web.settings.ai_keys.error_duplicate", "duplicate key")
	module := newSettingsModuleFromGateways(gateway, nil, settingsTestBase())
	mount, err := module.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, routepath.AppSettingsAIKeys, strings.NewReader("label=Primary&provider=anthropic&secret=sk-test"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusConflict)
	}
	body := rr.Body.String()
	for _, want := range []string{`id="settings-ai-keys"`, `value="Primary"`, `option value="anthropic" selected`} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q: %q", want, body)
		}
	}
}

func TestMountAIKeyRevokeConflictRedirectsWithErrorFlash(t *testing.T) {
	t.Parallel()

	gateway := newPopulatedFakeGateway()
	gateway.revokeAIKeyErr = apperrors.EK(apperrors.KindConflict, "web.settings.ai_keys.error_in_use", "in use")
	module := newSettingsModuleFromGateways(gateway, nil, settingsTestBase())
	mount, err := module.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, routepath.AppSettingsAIKeyRevoke("cred-1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppSettingsAIKeys {
		t.Fatalf("Location = %q, want %q", got, routepath.AppSettingsAIKeys)
	}
	if !responseHasCookieName(rr, flashnotice.CookieName) {
		t.Fatalf("response missing %q cookie", flashnotice.CookieName)
	}
}

func TestMountSecurityGetBackendFailureRendersAppErrorState(t *testing.T) {
	t.Parallel()

	module := newSettingsModuleFromGateways(
		testSettingsGateway(nil, nil, &passkeyClientStub{listErr: status.Error(codes.Unavailable, "offline")}, nil, nil),
		nil,
		settingsTestBase(),
	)
	mount, err := module.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppSettingsSecurity, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(rr.Body.String(), `id="app-error-state"`) {
		t.Fatalf("body missing app error state: %q", rr.Body.String())
	}
}

func TestMountAIAgentsGetDependencyFailuresRenderAppErrorState(t *testing.T) {
	t.Parallel()

	t.Run("credential options", func(t *testing.T) {
		t.Parallel()

		gateway := newPopulatedFakeGateway()
		gateway.listAIKeysErr = status.Error(codes.Unavailable, "offline")
		module := newSettingsModuleFromGateways(gateway, nil, settingsTestBase())
		mount, err := module.Mount()
		if err != nil {
			t.Fatalf("Mount() error = %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, routepath.AppSettingsAIAgents, nil)
		rr := httptest.NewRecorder()
		mount.Handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
		}
		if !strings.Contains(rr.Body.String(), `id="app-error-state"`) {
			t.Fatalf("body missing app error state: %q", rr.Body.String())
		}
	})

	t.Run("agent rows", func(t *testing.T) {
		t.Parallel()

		gateway := newPopulatedFakeGateway()
		gateway.listAgentsErr = status.Error(codes.Unavailable, "offline")
		module := newSettingsModuleFromGateways(gateway, nil, settingsTestBase())
		mount, err := module.Mount()
		if err != nil {
			t.Fatalf("Mount() error = %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, routepath.AppSettingsAIAgents, nil)
		rr := httptest.NewRecorder()
		mount.Handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
		}
		if !strings.Contains(rr.Body.String(), `id="app-error-state"`) {
			t.Fatalf("body missing app error state: %q", rr.Body.String())
		}
	})
}
