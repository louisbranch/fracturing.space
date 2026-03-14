package settings

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	settingsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/app"
	settingsgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/gateway"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	flashnotice "github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMountServesSettingsProfileGet(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: newPopulatedFakeGateway(), Base: settingsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppSettingsProfile, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("content-type = %q, want %q", got, "text/html; charset=utf-8")
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`id="settings-profile"`,
		`<form method="post" action="/app/settings/profile"`,
		`value="rhea"`,
		`name="name"`,
		`value="Rhea Vale"`,
		`name="bio"`,
		`<option value="she/her"></option>`,
		`<option value="he/him"></option>`,
		`<option value="they/them"></option>`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing marker %q: %q", marker, body)
		}
	}
	if strings.Contains(body, `<option value="it/its"></option>`) {
		t.Fatalf("body should hide it/its from pronoun selection: %q", body)
	}
	// Invariant: username is auth-owned and display-only in settings.
	if strings.Contains(body, `name="username"`) {
		t.Fatalf("profile settings body unexpectedly exposes editable username input: %q", body)
	}
	// Invariant: avatar catalog ids are not user-editable until catalog access is available.
	if strings.Contains(body, `name="avatar_set_id"`) || strings.Contains(body, `name="avatar_asset_id"`) {
		t.Fatalf("profile settings body unexpectedly exposes avatar catalog id inputs: %q", body)
	}
}

func TestMountServesSettingsProfileHead(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: newPopulatedFakeGateway(), Base: settingsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodHead, routepath.AppSettingsProfile, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestMountReturnsServiceUnavailableWhenGatewayNotConfigured(t *testing.T) {
	t.Parallel()

	m := New(Config{Base: settingsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppSettingsProfile, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error state marker: %q", body)
	}
	// Invariant: default module wiring must fail closed when settings backend is absent.
	if strings.Contains(body, `id="settings-profile"`) {
		t.Fatalf("body unexpectedly rendered settings profile without backend: %q", body)
	}
}

func TestMountSettingsProfileGetRendersPortugueseCopyWhenLanguageResolved(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: newPopulatedFakeGateway(), Base: modulehandler.NewBase(
		func(*http.Request) string { return "user-1" },
		func(*http.Request) string { return "pt-BR" },
		nil,
	)})

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppSettingsProfile, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`<h1 class="mb-0">Configurações</h1>`,
		`<h2 class="card-title">Perfil</h2>`,
		`<span class="label-text">Nome de usuário</span>`,
		`<button class="btn btn-primary" type="submit">Salvar perfil</button>`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing pt-BR marker %q: %q", marker, body)
		}
	}
}

func TestMountSettingsProfileMenuUsesPublicProfileLabelInEnglish(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: newPopulatedFakeGateway(), Base: settingsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppSettingsProfile, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `>Profile`) {
		t.Fatalf("body missing English profile menu label: %q", body)
	}
}

func TestMountRedirectsSettingsRootToProfile(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: newPopulatedFakeGateway(), Base: settingsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	for _, path := range []string{routepath.AppSettings, routepath.SettingsPrefix} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		mount.Handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusFound {
			t.Fatalf("path %q status = %d, want %d", path, rr.Code, http.StatusFound)
		}
		if got := rr.Header().Get("Location"); got != routepath.AppSettingsProfile {
			t.Fatalf("path %q Location = %q, want %q", path, got, routepath.AppSettingsProfile)
		}
	}
}

func TestMountSettingsRootHTMXUsesHXRedirect(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: newPopulatedFakeGateway(), Base: settingsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.SettingsPrefix, nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("HX-Redirect"); got != routepath.AppSettingsProfile {
		t.Fatalf("HX-Redirect = %q, want %q", got, routepath.AppSettingsProfile)
	}
}

func TestMountRedirectsSettingsRootToFirstAvailableSurface(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		gateway  settingsapp.Gateway
		expected string
	}{
		{
			name: "locale when profile unavailable",
			gateway: settingsgateway.NewGRPCGateway(
				nil,
				&accountClientStub{getResp: &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Locale: commonv1.Locale_LOCALE_EN_US}}},
				nil,
				nil,
				nil,
			),
			expected: routepath.AppSettingsLocale,
		},
		{
			name:     "ai keys when account unavailable",
			gateway:  settingsgateway.NewGRPCGateway(nil, nil, nil, &credentialClientStub{}, nil),
			expected: routepath.AppSettingsAIKeys,
		},
		{
			name:     "ai agents when only agents surface available",
			gateway:  settingsgateway.NewGRPCGateway(nil, nil, nil, &credentialClientStub{}, &agentClientStub{}),
			expected: routepath.AppSettingsAIKeys,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := New(Config{Gateway: tc.gateway, Base: settingsTestBase()})
			mount, err := m.Mount()
			if err != nil {
				t.Fatalf("Mount() error = %v", err)
			}
			req := httptest.NewRequest(http.MethodGet, routepath.AppSettings, nil)
			rr := httptest.NewRecorder()
			mount.Handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusFound {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
			}
			if got := rr.Header().Get("Location"); got != tc.expected {
				t.Fatalf("Location = %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestModuleIDReturnsSettings(t *testing.T) {
	t.Parallel()

	if got := New(Config{}).ID(); got != "settings" {
		t.Fatalf("ID() = %q, want %q", got, "settings")
	}
}

func TestModuleHealthyAndSchemePolicyOptions(t *testing.T) {
	t.Parallel()

	if New(Config{}).Healthy() {
		t.Fatalf("New().Healthy() = true, want false for degraded module")
	}

	scheme := requestmeta.SchemePolicy{TrustForwardedProto: true}
	module := New(Config{Gateway: newPopulatedFakeGateway(), Base: settingsTestBase(), FlashMeta: scheme})

	if !module.Healthy() {
		t.Fatalf("module.Healthy() = false, want true")
	}
	if module.flashMeta != scheme {
		t.Fatalf("module.flashMeta = %+v, want %+v", module.flashMeta, scheme)
	}
}

func TestModuleHealthyWhenAnySettingsSurfaceIsAvailable(t *testing.T) {
	t.Parallel()

	if !New(Config{
		Gateway: settingsgateway.NewGRPCGateway(
			nil,
			&accountClientStub{getResp: &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Locale: commonv1.Locale_LOCALE_EN_US}}},
			nil,
			nil,
			nil,
		),
		Base: settingsTestBase(),
	}).Healthy() {
		t.Fatalf("locale-only settings module = false, want true")
	}

	if !New(Config{
		Gateway: settingsgateway.NewGRPCGateway(nil, nil, nil, &credentialClientStub{}, nil),
		Base:    settingsTestBase(),
	}).Healthy() {
		t.Fatalf("ai-only settings module = false, want true")
	}
}

func TestMountUsesDependenciesSocialClientWhenGatewayNotProvided(t *testing.T) {
	t.Parallel()

	social := &socialClientStub{getResp: &socialv1.GetUserProfileResponse{UserProfile: &socialv1.UserProfile{
		UserId: "user-1",
		Name:   "Remote Name",
		Bio:    "From dependencies",
	}}}
	account := &accountClientStub{getResp: &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Username: "remote-user", Locale: commonv1.Locale_LOCALE_EN_US}}}
	m := New(Config{Gateway: settingsgateway.NewGRPCGateway(social, account, &passkeyClientStub{}, &credentialClientStub{}, &agentClientStub{}), Base: settingsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppSettingsProfile, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{`value="remote-user"`, `value="Remote Name"`, `From dependencies`} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing dependencies marker %q: %q", marker, body)
		}
	}
}

func TestMountSettingsProfileFailsClosedWhenSocialClientMissing(t *testing.T) {
	t.Parallel()

	account := &accountClientStub{getResp: &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Locale: commonv1.Locale_LOCALE_EN_US}}}
	m := New(Config{Gateway: settingsgateway.NewGRPCGateway(nil, account, nil, nil, nil), Base: settingsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppSettingsProfile, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
	if body := rr.Body.String(); !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error marker: %q", body)
	}
}

func TestMountSettingsLocaleFailsClosedWhenAccountClientMissing(t *testing.T) {
	t.Parallel()

	social := &socialClientStub{getResp: &socialv1.GetUserProfileResponse{UserProfile: &socialv1.UserProfile{UserId: "user-1", Name: "Remote Name"}}}
	m := New(Config{Gateway: settingsgateway.NewGRPCGateway(social, nil, nil, nil, nil), Base: settingsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppSettingsLocale, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
	if body := rr.Body.String(); !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error marker: %q", body)
	}
}

func TestMountSettingsAIKeysFailsClosedWhenCredentialClientMissing(t *testing.T) {
	t.Parallel()

	social := &socialClientStub{getResp: &socialv1.GetUserProfileResponse{UserProfile: &socialv1.UserProfile{UserId: "user-1", Name: "Remote Name"}}}
	m := New(Config{Gateway: settingsgateway.NewGRPCGateway(social, nil, nil, nil, nil), Base: settingsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppSettingsAIKeys, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
	if body := rr.Body.String(); !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error marker: %q", body)
	}
}

func TestMountRejectsSettingsUnsupportedMethod(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: newPopulatedFakeGateway(), Base: settingsTestBase()})
	mount, _ := m.Mount()
	req := httptest.NewRequest(http.MethodDelete, routepath.AppSettingsProfile, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestMountMapsSettingsGatewayErrorToHTTPStatus(t *testing.T) {
	t.Parallel()

	gateway := newPopulatedFakeGateway()
	gateway.loadProfileErr = apperrors.E(apperrors.KindUnauthorized, "missing session")
	m := New(Config{Gateway: gateway, Base: settingsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppSettingsProfile, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestMountSettingsGRPCNotFoundRendersAppErrorPage(t *testing.T) {
	t.Parallel()

	gateway := newPopulatedFakeGateway()
	gateway.loadProfileErr = status.Error(codes.NotFound, "user profile not found")
	m := New(Config{Gateway: gateway, Base: settingsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppSettingsProfile, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error state marker: %q", body)
	}
	// Invariant: backend transport errors must never leak raw gRPC strings to user-facing pages.
	if strings.Contains(body, "rpc error:") {
		t.Fatalf("body leaked raw grpc error: %q", body)
	}
}

func TestMountSettingsInternalErrorRendersServerErrorPage(t *testing.T) {
	t.Parallel()

	gateway := newPopulatedFakeGateway()
	gateway.loadProfileErr = errors.New("boom")
	m := New(Config{Gateway: gateway, Base: settingsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppSettingsProfile, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
	if body := rr.Body.String(); !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error state marker: %q", body)
	}
}

func TestMountSettingsGRPCNotFoundHTMXRendersErrorFragment(t *testing.T) {
	t.Parallel()

	gateway := newPopulatedFakeGateway()
	gateway.loadProfileErr = status.Error(codes.NotFound, "user profile not found")
	m := New(Config{Gateway: gateway, Base: settingsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppSettingsProfile, nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error state marker: %q", body)
	}
	// Invariant: HTMX failures must swap a fragment and not a full document.
	if strings.Contains(strings.ToLower(body), "<!doctype html") || strings.Contains(strings.ToLower(body), "<html") {
		t.Fatalf("expected htmx error fragment without document wrapper")
	}
}

func TestMountServesSettingsSubpaths(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: newPopulatedFakeGateway(), Base: settingsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	paths := map[string]string{
		routepath.AppSettingsProfile:  `action="/app/settings/profile"`,
		routepath.AppSettingsLocale:   `action="/app/settings/locale"`,
		routepath.AppSettingsAIKeys:   `action="/app/settings/ai-keys"`,
		routepath.AppSettingsAIAgents: `action="/app/settings/ai-agents"`,
	}
	for path, marker := range paths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		mount.Handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("path %q status = %d, want %d", path, rr.Code, http.StatusOK)
		}
		body := rr.Body.String()
		if !strings.Contains(body, marker) {
			t.Fatalf("path %q body missing marker %q: %q", path, marker, body)
		}
		if !strings.Contains(body, `<h1 class="mb-0">Settings</h1>`) {
			t.Fatalf("path %q body missing settings heading", path)
		}
		for _, href := range []string{routepath.AppSettingsProfile, routepath.AppSettingsLocale, routepath.AppSettingsAIKeys, routepath.AppSettingsAIAgents} {
			if !strings.Contains(body, `href="`+href+`"`) {
				t.Fatalf("path %q body missing menu href %q", path, href)
			}
		}
		if !strings.Contains(body, `href="#lucide-languages"`) {
			t.Fatalf("path %q body missing locale menu icon", path)
		}
	}
}

func TestMountHidesUnavailableSettingsLinksFromMenu(t *testing.T) {
	t.Parallel()

	m := New(Config{
		Gateway: settingsgateway.NewGRPCGateway(
			nil,
			&accountClientStub{getResp: &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Locale: commonv1.Locale_LOCALE_EN_US}}},
			nil,
			nil,
			nil,
		),
		Base: settingsTestBase(),
	})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppSettingsLocale, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `href="`+routepath.AppSettingsLocale+`"`) {
		t.Fatalf("body missing locale menu href: %q", body)
	}
	for _, href := range []string{routepath.AppSettingsProfile, routepath.AppSettingsAIKeys, routepath.AppSettingsAIAgents} {
		if strings.Contains(body, `href="`+href+`"`) {
			t.Fatalf("body unexpectedly exposes unavailable menu href %q: %q", href, body)
		}
	}
}

func TestMountSettingsHTMXReturnsFragmentWithoutDocumentWrapper(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: newPopulatedFakeGateway(), Base: settingsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppSettingsAIKeys, nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="settings-ai-keys"`) {
		t.Fatalf("body = %q, want settings ai keys marker", body)
	}
	// Invariant: HTMX requests must receive partial content, never a full document envelope.
	if strings.Contains(strings.ToLower(body), "<!doctype html") || strings.Contains(strings.ToLower(body), "<html") {
		t.Fatalf("expected htmx fragment without document wrapper")
	}
}

func TestMountAIAgentsGetLoadsModelsAndListsAgents(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: newPopulatedFakeGateway(), Base: settingsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppSettingsAIAgents+"?credential_id=cred-1", nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`id="settings-ai-agents"`,
		`action="/app/settings/ai-agents"`,
		`name="label"`,
		`name="credential_id"`,
		`name="model"`,
		`name="instructions"`,
		`value="gpt-4o-mini"`,
		`narrator`,
		`Keep the session moving.`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing marker %q: %q", marker, body)
		}
	}
}

func TestMountProfilePostSavesAndRedirects(t *testing.T) {
	t.Parallel()

	gateway := newPopulatedFakeGateway()
	m := New(Config{Gateway: gateway, Base: settingsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	form := url.Values{
		"username":        {"updated-user"},
		"name":            {"Rhea Vale"},
		"avatar_set_id":   {"catalog-hack-set"},
		"avatar_asset_id": {"catalog-hack-asset"},
		"bio":             {"Traveler"},
	}
	req := httptest.NewRequest(http.MethodPost, routepath.AppSettingsProfile, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppSettingsProfile {
		t.Fatalf("Location = %q, want %q", got, routepath.AppSettingsProfile)
	}
	if !responseHasCookieName(rr, flashnotice.CookieName) {
		t.Fatalf("response missing %q cookie", flashnotice.CookieName)
	}
	if gateway.lastSavedProfile.Username != "rhea" {
		t.Fatalf("saved username = %q, want existing auth username %q", gateway.lastSavedProfile.Username, "rhea")
	}
	if gateway.lastSavedProfile.AvatarSetID != "set-a" {
		t.Fatalf("saved avatar set id = %q, want existing value %q", gateway.lastSavedProfile.AvatarSetID, "set-a")
	}
	if gateway.lastSavedProfile.AvatarAssetID != "asset-1" {
		t.Fatalf("saved avatar asset id = %q, want existing value %q", gateway.lastSavedProfile.AvatarAssetID, "asset-1")
	}
}

func TestMountProfilePostUsesDependenciesSocialClientWhenGatewayNotProvided(t *testing.T) {
	t.Parallel()

	social := &socialClientStub{getResp: &socialv1.GetUserProfileResponse{UserProfile: &socialv1.UserProfile{
		UserId:        "user-1",
		Name:          "Remote Name",
		AvatarSetId:   "set-a",
		AvatarAssetId: "asset-1",
		Bio:           "Before",
	}}}
	account := &accountClientStub{getResp: &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Username: "remote-user", Locale: commonv1.Locale_LOCALE_EN_US}}}
	m := New(Config{Gateway: settingsgateway.NewGRPCGateway(social, account, &passkeyClientStub{}, &credentialClientStub{}, &agentClientStub{}), Base: settingsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	form := url.Values{
		"name":            {"Updated Name"},
		"avatar_set_id":   {"catalog-hack-set"},
		"avatar_asset_id": {"catalog-hack-asset"},
		"bio":             {"After"},
	}
	req := httptest.NewRequest(http.MethodPost, routepath.AppSettingsProfile, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppSettingsProfile {
		t.Fatalf("Location = %q, want %q", got, routepath.AppSettingsProfile)
	}
	if !responseHasCookieName(rr, flashnotice.CookieName) {
		t.Fatalf("response missing %q cookie", flashnotice.CookieName)
	}
	if social.lastSetReq == nil {
		t.Fatalf("expected SetUserProfile to be called")
	}
	if social.lastSetReq.GetName() != "Updated Name" {
		t.Fatalf("name = %q, want %q", social.lastSetReq.GetName(), "Updated Name")
	}
	if social.lastSetReq.GetBio() != "After" {
		t.Fatalf("bio = %q, want %q", social.lastSetReq.GetBio(), "After")
	}
	if social.lastSetReq.GetAvatarSetId() != "set-a" {
		t.Fatalf("avatar set id = %q, want %q", social.lastSetReq.GetAvatarSetId(), "set-a")
	}
	if social.lastSetReq.GetAvatarAssetId() != "asset-1" {
		t.Fatalf("avatar asset id = %q, want %q", social.lastSetReq.GetAvatarAssetId(), "asset-1")
	}
}

func TestMountProfilePostBlankPronounsSavesUnspecifiedPronouns(t *testing.T) {
	t.Parallel()

	social := &socialClientStub{getResp: &socialv1.GetUserProfileResponse{UserProfile: &socialv1.UserProfile{
		UserId:        "user-1",
		Name:          "Remote Name",
		AvatarSetId:   "set-a",
		AvatarAssetId: "asset-1",
		Bio:           "Before",
	}}}
	account := &accountClientStub{getResp: &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Username: "remote-user", Locale: commonv1.Locale_LOCALE_EN_US}}}
	m := New(Config{Gateway: settingsgateway.NewGRPCGateway(social, account, &passkeyClientStub{}, &credentialClientStub{}, &agentClientStub{}), Base: settingsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	form := url.Values{
		"name":            {"Updated Name"},
		"pronouns":        {"   "},
		"avatar_set_id":   {"catalog-hack-set"},
		"avatar_asset_id": {"catalog-hack-asset"},
		"bio":             {"After"},
	}
	req := httptest.NewRequest(http.MethodPost, routepath.AppSettingsProfile, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if social.lastSetReq == nil {
		t.Fatalf("expected SetUserProfile to be called")
	}
	if social.lastSetReq.GetPronouns() != nil {
		t.Fatalf("expected pronouns to be unspecified, got %v", social.lastSetReq.GetPronouns())
	}
}

func TestMountProfilePostValidationErrorRendersBadRequest(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: newPopulatedFakeGateway(), Base: settingsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	form := url.Values{"name": {strings.Repeat("x", settingsapp.UserProfileNameMaxLength+1)}}
	req := httptest.NewRequest(http.MethodPost, routepath.AppSettingsProfile, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rr.Body.String(), "Name must be at most 64 characters.") {
		t.Fatalf("body missing validation error: %q", rr.Body.String())
	}
}

func TestMountProfilePostValidationErrorRendersLocalizedBadRequest(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: newPopulatedFakeGateway(), Base: modulehandler.NewBase(
		func(*http.Request) string { return "user-1" },
		func(*http.Request) string { return "pt-BR" },
		nil,
	)})

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	form := url.Values{"name": {strings.Repeat("x", settingsapp.UserProfileNameMaxLength+1)}}
	req := httptest.NewRequest(http.MethodPost, routepath.AppSettingsProfile, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rr.Body.String(), "Nome deve ter no máximo 64 caracteres") {
		t.Fatalf("body missing localized validation error: %q", rr.Body.String())
	}
}

func TestMountLocalePostSavesAndRedirects(t *testing.T) {
	t.Parallel()

	gateway := newPopulatedFakeGateway()
	m := New(Config{Gateway: gateway, Base: settingsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	form := url.Values{"locale": {"pt-BR"}}
	req := httptest.NewRequest(http.MethodPost, routepath.AppSettingsLocale, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppSettingsLocale {
		t.Fatalf("Location = %q, want %q", got, routepath.AppSettingsLocale)
	}
	if !responseHasCookieName(rr, flashnotice.CookieName) {
		t.Fatalf("response missing %q cookie", flashnotice.CookieName)
	}
	if gateway.lastSavedLocale != "pt-BR" {
		t.Fatalf("saved locale = %v, want %v", gateway.lastSavedLocale, "pt-BR")
	}
}

func TestMountLocalePostValidationErrorRendersBadRequest(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: newPopulatedFakeGateway(), Base: settingsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	form := url.Values{"locale": {"es-ES"}}
	req := httptest.NewRequest(http.MethodPost, routepath.AppSettingsLocale, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rr.Body.String(), "invalid locale") {
		t.Fatalf("body missing validation error: %q", rr.Body.String())
	}
}

func TestMountAIKeysCreatePostSavesAndRedirects(t *testing.T) {
	t.Parallel()

	gateway := newPopulatedFakeGateway()
	m := New(Config{Gateway: gateway, Base: settingsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	form := url.Values{"label": {"Primary"}, "secret": {"sk-test"}}
	req := httptest.NewRequest(http.MethodPost, routepath.AppSettingsAIKeys, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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
	if gateway.lastCreatedLabel != "Primary" {
		t.Fatalf("created label = %q, want %q", gateway.lastCreatedLabel, "Primary")
	}
}

func TestMountAIKeysCreatePostValidationErrorRendersBadRequest(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: newPopulatedFakeGateway(), Base: settingsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	form := url.Values{"label": {"Primary"}}
	req := httptest.NewRequest(http.MethodPost, routepath.AppSettingsAIKeys, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rr.Body.String(), "Label and API key secret are required.") {
		t.Fatalf("body missing validation error: %q", rr.Body.String())
	}
}

func TestMountAIAgentsCreatePostSavesAndRedirects(t *testing.T) {
	t.Parallel()

	gateway := newPopulatedFakeGateway()
	m := New(Config{Gateway: gateway, Base: settingsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	form := url.Values{
		"label":         {"narrator"},
		"credential_id": {"cred-1"},
		"model":         {"gpt-4o-mini"},
		"instructions":  {"Keep the session moving."},
	}
	req := httptest.NewRequest(http.MethodPost, routepath.AppSettingsAIAgents, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppSettingsAIAgents {
		t.Fatalf("Location = %q, want %q", got, routepath.AppSettingsAIAgents)
	}
	if gateway.lastCreatedAgent.Label != "narrator" || gateway.lastCreatedAgent.CredentialID != "cred-1" {
		t.Fatalf("created agent = %+v", gateway.lastCreatedAgent)
	}
}

func TestMountAIAgentsCreatePostValidationErrorRendersBadRequest(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: newPopulatedFakeGateway(), Base: settingsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	form := url.Values{
		"label":         {"narrator"},
		"credential_id": {"cred-1"},
	}
	req := httptest.NewRequest(http.MethodPost, routepath.AppSettingsAIAgents, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rr.Body.String(), "Label, credential, and model are required.") {
		t.Fatalf("body missing validation error: %q", rr.Body.String())
	}
}

func TestMountAIKeyRevokeUsesHTTPRedirect(t *testing.T) {
	t.Parallel()

	gateway := newPopulatedFakeGateway()
	m := New(Config{Gateway: gateway, Base: settingsTestBase()})
	mount, err := m.Mount()
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
	if gateway.lastRevokedCredentialID != "cred-1" {
		t.Fatalf("revoked id = %q, want %q", gateway.lastRevokedCredentialID, "cred-1")
	}
}

func TestMountAIKeyRevokeHTMXUsesHXRedirect(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: newPopulatedFakeGateway(), Base: settingsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, routepath.AppSettingsAIKeyRevoke("cred-1"), nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("HX-Redirect"); got != routepath.AppSettingsAIKeys {
		t.Fatalf("HX-Redirect = %q, want %q", got, routepath.AppSettingsAIKeys)
	}
	if !responseHasCookieName(rr, flashnotice.CookieName) {
		t.Fatalf("response missing %q cookie", flashnotice.CookieName)
	}
}

func TestMountSettingsUnknownSubpathReturnsNotFound(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: newPopulatedFakeGateway(), Base: settingsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.SettingsPrefix+"unknown", nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	if got := rr.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("content-type = %q, want text/html", got)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error state marker: %q", body)
	}
	// Invariant: unknown app routes should use the shared not-found page, not net/http plain text.
	if strings.Contains(body, "404 page not found") {
		t.Fatalf("body unexpectedly rendered plain 404 text: %q", body)
	}
}

func TestMountAIKeyRevokeRejectsNonPost(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: newPopulatedFakeGateway(), Base: settingsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppSettingsAIKeyRevoke("cred-1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
	if got := rr.Header().Get("Allow"); got != http.MethodPost {
		t.Fatalf("Allow = %q, want %q", got, http.MethodPost)
	}
}

func TestMountAIKeyRevokeMapsGatewayErrorStatus(t *testing.T) {
	t.Parallel()

	gateway := newPopulatedFakeGateway()
	gateway.revokeAIKeyErr = apperrors.E(apperrors.KindForbidden, "forbidden")
	m := New(Config{Gateway: gateway, Base: settingsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, routepath.AppSettingsAIKeyRevoke("cred-1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func settingsTestBase() modulehandler.Base {
	return modulehandler.NewBase(func(*http.Request) string { return "user-1" }, nil, nil)
}

func responseHasCookieName(rr *httptest.ResponseRecorder, name string) bool {
	return responseCookieByName(rr, name) != nil
}

func responseCookieByName(rr *httptest.ResponseRecorder, name string) *http.Cookie {
	if rr == nil {
		return nil
	}
	for _, cookie := range rr.Result().Cookies() {
		if cookie != nil && cookie.Name == name {
			return cookie
		}
	}
	return nil
}
