package public

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestModuleIDReturnsPublic(t *testing.T) {
	t.Parallel()

	if got := New().ID(); got != "public" {
		t.Fatalf("ID() = %q, want %q", got, "public")
	}
}

func TestMountServesRootAndLogin(t *testing.T) {
	t.Parallel()

	m := New()
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	for _, path := range []string{routepath.Root, routepath.Login, routepath.Health} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		mount.Handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("path %q status = %d, want %d", path, rr.Code, http.StatusOK)
		}
		if got := rr.Header().Get("Content-Type"); got == "" {
			t.Fatalf("path %q missing content-type", path)
		}
		if path == routepath.Root || path == routepath.Login {
			assertAuthShellMarkers(t, path, rr.Body.String())
		}
	}
}

func TestMountServesHeadForAuthReads(t *testing.T) {
	t.Parallel()

	m := New()
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	for _, path := range []string{routepath.Root, routepath.Login, routepath.AuthLogin, routepath.Health} {
		req := httptest.NewRequest(http.MethodHead, path, nil)
		rr := httptest.NewRecorder()
		mount.Handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK && !(path == routepath.AuthLogin && rr.Code == http.StatusFound) {
			t.Fatalf("path %q status = %d, want %d", path, rr.Code, http.StatusOK)
		}
	}
}

func TestMountAuthLocaleSwitchPersistsCookieAndHighlightsSelection(t *testing.T) {
	t.Parallel()

	m := New()
	mount, _ := m.Mount(module.Dependencies{})
	req := httptest.NewRequest(http.MethodGet, routepath.Login+"?lang=pt-BR", nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	setCookie := rr.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, "fs_lang=pt-BR") {
		t.Fatalf("Set-Cookie = %q, want fs_lang=pt-BR", setCookie)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `data-lang="pt-BR" class="font-bold"`) {
		t.Fatalf("body missing active PT-BR marker: %q", body)
	}
	if !strings.Contains(body, `data-lang="en-US"`) {
		t.Fatalf("body missing EN option: %q", body)
	}
}

func TestMountAuthLocaleUsesLanguageCookie(t *testing.T) {
	t.Parallel()

	m := New()
	mount, _ := m.Mount(module.Dependencies{})
	req := httptest.NewRequest(http.MethodGet, routepath.Login, nil)
	req.AddCookie(&http.Cookie{Name: "fs_lang", Value: "pt-BR"})
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `<html lang="pt-BR"`) {
		t.Fatalf("body missing lang pt-BR: %q", body)
	}
	if !strings.Contains(body, `data-lang="pt-BR" class="font-bold"`) {
		t.Fatalf("body missing active PT-BR marker: %q", body)
	}
}

func TestMountAuthLocaleRendersPortugueseCopy(t *testing.T) {
	t.Parallel()

	m := New()
	mount, _ := m.Mount(module.Dependencies{})

	for _, path := range []string{routepath.Root, routepath.Login} {
		req := httptest.NewRequest(http.MethodGet, path+"?lang=pt-BR", nil)
		rr := httptest.NewRecorder()
		mount.Handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("path %q status = %d, want %d", path, rr.Code, http.StatusOK)
		}
		body := rr.Body.String()
		if !strings.Contains(body, "Motor de código aberto") {
			t.Fatalf("path %q body missing Portuguese tagline: %q", path, body)
		}
		if path == routepath.Root && !strings.Contains(body, ">Entrar<") {
			t.Fatalf("path %q body missing Portuguese sign-in label: %q", path, body)
		}
		if path == routepath.Login && !strings.Contains(body, "Faça login em Fracturing.Space") {
			t.Fatalf("path %q body missing Portuguese login heading: %q", path, body)
		}
	}
}

func TestMountAuthLocaleRendersPortuguesePasskeyScriptStrings(t *testing.T) {
	t.Parallel()

	m := New()
	mount, _ := m.Mount(module.Dependencies{})
	req := httptest.NewRequest(http.MethodGet, routepath.Login+"?lang=pt-BR", nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Email principal é obrigatório.") {
		t.Fatalf("body missing localized email-required JS string: %q", body)
	}
	if !strings.Contains(body, "Falha no login com chave de acesso.") {
		t.Fatalf("body missing localized passkey-failed JS string: %q", body)
	}
}

func TestMountLoginPublishesPasskeyEndpointDataAttributes(t *testing.T) {
	t.Parallel()

	m := New()
	mount, _ := m.Mount(module.Dependencies{})
	req := httptest.NewRequest(http.MethodGet, routepath.Login, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-login-start-path="` + routepath.PasskeyLoginStart + `"`,
		`data-login-finish-path="` + routepath.PasskeyLoginFinish + `"`,
		`data-register-start-path="` + routepath.PasskeyRegisterStart + `"`,
		`data-register-finish-path="` + routepath.PasskeyRegisterFinish + `"`,
		`src="/static/passkey-auth.js"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing passkey endpoint marker %q: %q", marker, body)
		}
	}
	if strings.Contains(body, "async function performPasskeyLogin") {
		t.Fatalf("body unexpectedly includes inline passkey auth script")
	}
}

func assertAuthShellMarkers(t *testing.T, path, body string) {
	t.Helper()
	for _, marker := range []string{"id=\"auth-shell\"", "id=\"auth-language-menu\""} {
		if !strings.Contains(body, marker) {
			t.Fatalf("path %q body missing marker %q", path, marker)
		}
	}
	if path == routepath.Root {
		for _, marker := range []string{"class=\"landing-hero\"", "Sign in", "Docs", "GitHub"} {
			if !strings.Contains(body, marker) {
				t.Fatalf("path %q body missing marker %q", path, marker)
			}
		}
	}
}

func TestMountAuthLoginRedirectsToLogin(t *testing.T) {
	t.Parallel()

	m := New()
	mount, _ := m.Mount(module.Dependencies{})
	req := httptest.NewRequest(http.MethodGet, routepath.AuthLogin, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.Login {
		t.Fatalf("Location = %q, want %q", got, routepath.Login)
	}
}

func TestMountAuthPagesRedirectAuthenticatedUsersToCampaigns(t *testing.T) {
	t.Parallel()

	m := New()
	mount, _ := m.Mount(module.Dependencies{AuthClient: validatingAuthClient{validSessionID: "ws-1"}})

	for _, path := range []string{routepath.Root, routepath.Login, routepath.AuthLogin} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.AddCookie(&http.Cookie{Name: "web_session", Value: "ws-1"})
		rr := httptest.NewRecorder()
		mount.Handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusFound {
			t.Fatalf("path %q status = %d, want %d", path, rr.Code, http.StatusFound)
		}
		if got := rr.Header().Get("Location"); got != routepath.AppCampaigns {
			t.Fatalf("path %q Location = %q, want %q", path, got, routepath.AppCampaigns)
		}
	}
}

func TestMountAuthPagesDoNotRedirectUnknownSessionCookie(t *testing.T) {
	t.Parallel()

	m := New()
	mount, _ := m.Mount(module.Dependencies{AuthClient: validatingAuthClient{validSessionID: "ws-1"}})

	for _, path := range []string{routepath.Root, routepath.Login} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.AddCookie(&http.Cookie{Name: "web_session", Value: "missing-session"})
		rr := httptest.NewRecorder()
		mount.Handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("path %q status = %d, want %d", path, rr.Code, http.StatusOK)
		}
		assertAuthShellMarkers(t, path, rr.Body.String())
	}
}

func TestMountLoginIgnoresUntrustedUserHeader(t *testing.T) {
	t.Parallel()

	m := New()
	mount, _ := m.Mount(module.Dependencies{AuthClient: validatingAuthClient{validSessionID: "ws-1"}})
	req := httptest.NewRequest(http.MethodGet, routepath.Login, nil)
	req.Header.Set("X-Web-User", "user-1")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	assertAuthShellMarkers(t, routepath.Login, rr.Body.String())
}

func TestMountPasskeyLoginStartReturnsJSONChallenge(t *testing.T) {
	t.Parallel()

	m := New()
	mount, _ := m.Mount(module.Dependencies{AuthClient: fakeAuthClient{}})
	req := httptest.NewRequest(http.MethodPost, routepath.PasskeyLoginStart, strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("content-type = %q, want application/json", got)
	}
	payload := decodeJSONBody(t, rr.Body.Bytes())
	if strings.TrimSpace(asString(payload["session_id"])) == "" {
		t.Fatalf("expected session_id in response payload")
	}
	if payload["public_key"] == nil {
		t.Fatalf("expected public_key in response payload")
	}
}

func TestMountPasskeyLoginFinishSetsCookieAndRedirect(t *testing.T) {
	t.Parallel()

	m := New()
	mount, _ := m.Mount(module.Dependencies{AuthClient: fakeAuthClient{}})
	payload := map[string]any{"session_id": "session-1", "credential": map[string]any{"id": "cred-1"}}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, routepath.PasskeyLoginFinish, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	setCookie := rr.Header().Get("Set-Cookie")
	cookie, err := http.ParseSetCookie(setCookie)
	if err != nil {
		t.Fatalf("ParseSetCookie() error = %v", err)
	}
	if cookie.Name != "web_session" {
		t.Fatalf("cookie name = %q, want %q", cookie.Name, "web_session")
	}
	if cookie.Value == "" || cookie.Value == "user-1" {
		t.Fatalf("cookie value must be opaque session id, got %q", cookie.Value)
	}
	if cookie.Path != "/" {
		t.Fatalf("cookie path = %q, want %q", cookie.Path, "/")
	}
	if !cookie.HttpOnly {
		t.Fatalf("expected HttpOnly cookie")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie SameSite = %v, want %v", cookie.SameSite, http.SameSiteLaxMode)
	}
	payloadBody := decodeJSONBody(t, rr.Body.Bytes())
	if got := asString(payloadBody["redirect_url"]); got != routepath.AppCampaigns {
		t.Fatalf("redirect_url = %q, want %q", got, routepath.AppCampaigns)
	}
}

func TestMountPasskeyLoginFinishSetsSecureCookieForHTTPS(t *testing.T) {
	t.Parallel()

	m := New()
	mount, _ := m.Mount(module.Dependencies{AuthClient: fakeAuthClient{}})
	payload := map[string]any{"session_id": "session-1", "credential": map[string]any{"id": "cred-1"}}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "https://app.example.test"+routepath.PasskeyLoginFinish, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	cookie, err := http.ParseSetCookie(rr.Header().Get("Set-Cookie"))
	if err != nil {
		t.Fatalf("ParseSetCookie() error = %v", err)
	}
	if !cookie.Secure {
		t.Fatalf("expected secure session cookie for https request")
	}
}

func TestMountLogoutClearsSessionCookieAndRedirectsToLogin(t *testing.T) {
	t.Parallel()

	m := New()
	mount, _ := m.Mount(module.Dependencies{})
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: "web_session", Value: "user-1"})
	req.Header.Set("Origin", "http://example.com")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.Login {
		t.Fatalf("Location = %q, want %q", got, routepath.Login)
	}
	setCookie := rr.Header().Get("Set-Cookie")
	cookie, err := http.ParseSetCookie(setCookie)
	if err != nil {
		t.Fatalf("ParseSetCookie() error = %v", err)
	}
	if cookie.Name != "web_session" {
		t.Fatalf("cookie name = %q, want %q", cookie.Name, "web_session")
	}
	if cookie.MaxAge > 0 {
		t.Fatalf("cookie max-age = %d, want immediate expiration", cookie.MaxAge)
	}
}

func TestMountLogoutSetsSecureClearingCookieForHTTPS(t *testing.T) {
	t.Parallel()

	m := New()
	mount, _ := m.Mount(module.Dependencies{})
	req := httptest.NewRequest(http.MethodPost, "https://app.example.test"+routepath.Logout, nil)
	req.AddCookie(&http.Cookie{Name: "web_session", Value: "user-1"})
	req.Header.Set("Origin", "https://app.example.test")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	cookie, err := http.ParseSetCookie(rr.Header().Get("Set-Cookie"))
	if err != nil {
		t.Fatalf("ParseSetCookie() error = %v", err)
	}
	if !cookie.Secure {
		t.Fatalf("expected secure clearing cookie for https request")
	}
}

func TestMountLogoutRejectsGetMethod(t *testing.T) {
	t.Parallel()

	m := New()
	mount, _ := m.Mount(module.Dependencies{})
	req := httptest.NewRequest(http.MethodGet, "/logout", nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
	if got := rr.Header().Get("Allow"); got != http.MethodPost {
		t.Fatalf("Allow = %q, want %q", got, http.MethodPost)
	}
}

func TestMountLogoutRejectsCookieMutationWithoutSameOriginProof(t *testing.T) {
	t.Parallel()

	m := New()
	mount, _ := m.Mount(module.Dependencies{})
	req := httptest.NewRequest(http.MethodPost, routepath.Logout, nil)
	req.AddCookie(&http.Cookie{Name: "web_session", Value: "user-1"})
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestMountAuthPagesRedirectAuthenticatedUsersToValidatedNextPath(t *testing.T) {
	t.Parallel()

	m := New()
	mount, _ := m.Mount(module.Dependencies{AuthClient: validatingAuthClient{validSessionID: "ws-1"}})
	req := httptest.NewRequest(http.MethodGet, routepath.Login+"?next=/app/settings", nil)
	req.AddCookie(&http.Cookie{Name: "web_session", Value: "ws-1"})
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppSettings {
		t.Fatalf("Location = %q, want %q", got, routepath.AppSettings)
	}
}

func TestResolveAppRedirectPathValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "empty falls back", raw: "", want: routepath.AppCampaigns},
		{name: "whitespace falls back", raw: "   ", want: routepath.AppCampaigns},
		{name: "non app path falls back", raw: "/login", want: routepath.AppCampaigns},
		{name: "absolute url falls back", raw: "https://example.com/app/settings", want: routepath.AppCampaigns},
		{name: "invalid url falls back", raw: "http://[::1", want: routepath.AppCampaigns},
		{name: "keeps app route", raw: "/app/settings", want: "/app/settings"},
		{name: "keeps app route query", raw: "/app/settings?tab=profile", want: "/app/settings?tab=profile"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := resolveAppRedirectPath(tc.raw); got != tc.want {
				t.Fatalf("resolveAppRedirectPath(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

func TestHandlersPasskeyEndpointsReturnJSONErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		handler handlers
		path    string
		body    string
		invoke  func(handlers, http.ResponseWriter, *http.Request)
	}{
		{
			name:    "login start service failure",
			handler: newHandlers(service{auth: &authGatewayStub{beginPasskeyLoginErr: errors.New("boom")}}),
			path:    routepath.PasskeyLoginStart,
			body:    `{}`,
			invoke:  func(h handlers, w http.ResponseWriter, r *http.Request) { h.handlePasskeyLoginStart(w, r) },
		},
		{
			name:    "login finish invalid json",
			handler: newHandlers(service{auth: &authGatewayStub{}}),
			path:    routepath.PasskeyLoginFinish,
			body:    `{`,
			invoke:  func(h handlers, w http.ResponseWriter, r *http.Request) { h.handlePasskeyLoginFinish(w, r) },
		},
		{
			name:    "register start invalid json",
			handler: newHandlers(service{auth: &authGatewayStub{}}),
			path:    routepath.PasskeyRegisterStart,
			body:    `{`,
			invoke:  func(h handlers, w http.ResponseWriter, r *http.Request) { h.handlePasskeyRegisterStart(w, r) },
		},
		{
			name:    "register finish service failure",
			handler: newHandlers(service{auth: &authGatewayStub{finishPasskeyRegistrationErr: errors.New("boom")}}),
			path:    routepath.PasskeyRegisterFinish,
			body:    `{"session_id":"session-1","credential":{"id":"cred-1"}}`,
			invoke:  func(h handlers, w http.ResponseWriter, r *http.Request) { h.handlePasskeyRegisterFinish(w, r) },
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			tc.invoke(tc.handler, rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
			}
			if got := rr.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
				t.Fatalf("content-type = %q, want application/json", got)
			}
			payload := decodeJSONBody(t, rr.Body.Bytes())
			if strings.TrimSpace(asString(payload["error"])) == "" {
				t.Fatalf("expected non-empty json error payload")
			}
		})
	}
}

func TestHandlersWriteAuthPageRenderFailureWritesErrorBody(t *testing.T) {
	t.Parallel()

	h := newHandlers(service{auth: &authGatewayStub{}})
	req := httptest.NewRequest(http.MethodGet, routepath.Login, nil)
	rr := httptest.NewRecorder()
	h.writeAuthPage(rr, req, "Login", "desc", "en-US", failingTemplComponent{err: errors.New("render failed")})

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
	if got := rr.Header().Get("Content-Type"); !strings.Contains(got, "text/plain") {
		t.Fatalf("content-type = %q, want text/plain error response", got)
	}
	body := rr.Body.String()
	if !strings.Contains(body, http.StatusText(http.StatusInternalServerError)) {
		t.Fatalf("body = %q, want generic internal-server-error body", body)
	}
	// Invariant: template/render failures must not leak internal error details to users.
	if strings.Contains(body, "render failed") {
		t.Fatalf("body leaked internal render error: %q", body)
	}
}

func TestWriteJSONErrorDoesNotLeakInternalErrorStrings(t *testing.T) {
	t.Parallel()

	h := newHandlers(service{auth: &authGatewayStub{}})
	req := httptest.NewRequest(http.MethodPost, routepath.PasskeyLoginStart, nil)
	rr := httptest.NewRecorder()
	h.writeJSONError(rr, req, errors.New("backend exploded"))

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
	payload := decodeJSONBody(t, rr.Body.Bytes())
	errMsg := asString(payload["error"])
	if errMsg != http.StatusText(http.StatusInternalServerError) {
		t.Fatalf("error = %q, want %q", errMsg, http.StatusText(http.StatusInternalServerError))
	}
}

func TestMountPasskeyRegisterStartAndFinish(t *testing.T) {
	t.Parallel()

	m := New()
	mount, _ := m.Mount(module.Dependencies{AuthClient: fakeAuthClient{}})
	startReq := httptest.NewRequest(http.MethodPost, routepath.PasskeyRegisterStart, strings.NewReader(`{"email":"new@example.com"}`))
	startReq.Header.Set("Content-Type", "application/json")
	startRR := httptest.NewRecorder()
	mount.Handler.ServeHTTP(startRR, startReq)
	if startRR.Code != http.StatusOK {
		t.Fatalf("register start status = %d, want %d", startRR.Code, http.StatusOK)
	}
	startPayload := decodeJSONBody(t, startRR.Body.Bytes())
	if strings.TrimSpace(asString(startPayload["user_id"])) == "" {
		t.Fatalf("expected user_id in register start payload")
	}

	finishReq := httptest.NewRequest(http.MethodPost, routepath.PasskeyRegisterFinish, strings.NewReader(`{"session_id":"register-session","credential":{"id":"cred-1"}}`))
	finishReq.Header.Set("Content-Type", "application/json")
	finishRR := httptest.NewRecorder()
	mount.Handler.ServeHTTP(finishRR, finishReq)
	if finishRR.Code != http.StatusOK {
		t.Fatalf("register finish status = %d, want %d", finishRR.Code, http.StatusOK)
	}
	finishPayload := decodeJSONBody(t, finishRR.Body.Bytes())
	if strings.TrimSpace(asString(finishPayload["user_id"])) == "" {
		t.Fatalf("expected user_id in register finish payload")
	}
}

func TestMountPasskeyRegisterStartCreateUserFailureReturnsLegacyErrorMessage(t *testing.T) {
	t.Parallel()

	m := New()
	mount, _ := m.Mount(module.Dependencies{AuthClient: failingCreateUserAuthClient{}})
	req := httptest.NewRequest(http.MethodPost, routepath.PasskeyRegisterStart, strings.NewReader(`{"email":"existing@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	payload := decodeJSONBody(t, rr.Body.Bytes())
	if got := asString(payload["error"]); got != "failed to create user" {
		t.Fatalf("error = %q, want %q", got, "failed to create user")
	}
}

func decodeJSONBody(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v, body=%q", err, string(body))
	}
	return payload
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

type failingTemplComponent struct {
	err error
}

func (c failingTemplComponent) Render(context.Context, io.Writer) error {
	if c.err != nil {
		return c.err
	}
	return errors.New("render failed")
}

type fakeAuthClient struct{}

type failingCreateUserAuthClient struct {
	fakeAuthClient
}

type validatingAuthClient struct {
	fakeAuthClient
	validSessionID string
}

func (f validatingAuthClient) GetWebSession(_ context.Context, req *authv1.GetWebSessionRequest, _ ...grpc.CallOption) (*authv1.GetWebSessionResponse, error) {
	if req.GetSessionId() != f.validSessionID {
		return nil, errors.New("unknown session")
	}
	return &authv1.GetWebSessionResponse{Session: &authv1.WebSession{Id: f.validSessionID, UserId: "user-1"}, User: &authv1.User{Id: "user-1"}}, nil
}

func (fakeAuthClient) CreateUser(context.Context, *authv1.CreateUserRequest, ...grpc.CallOption) (*authv1.CreateUserResponse, error) {
	return &authv1.CreateUserResponse{User: &authv1.User{Id: "user-1"}}, nil
}

func (failingCreateUserAuthClient) CreateUser(context.Context, *authv1.CreateUserRequest, ...grpc.CallOption) (*authv1.CreateUserResponse, error) {
	return nil, status.Error(codes.AlreadyExists, "email already in use")
}

func (fakeAuthClient) BeginPasskeyRegistration(context.Context, *authv1.BeginPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.BeginPasskeyRegistrationResponse, error) {
	return &authv1.BeginPasskeyRegistrationResponse{SessionId: "register-session", CredentialCreationOptionsJson: []byte(`{"publicKey":{"challenge":"ZmFrZQ","rp":{"name":"web"},"user":{"id":"dXNlcg","name":"new@example.com","displayName":"new@example.com"},"pubKeyCredParams":[{"type":"public-key","alg":-7}]}}`)}, nil
}

func (fakeAuthClient) FinishPasskeyRegistration(context.Context, *authv1.FinishPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.FinishPasskeyRegistrationResponse, error) {
	return &authv1.FinishPasskeyRegistrationResponse{User: &authv1.User{Id: "user-1"}}, nil
}

func (fakeAuthClient) BeginPasskeyLogin(context.Context, *authv1.BeginPasskeyLoginRequest, ...grpc.CallOption) (*authv1.BeginPasskeyLoginResponse, error) {
	return &authv1.BeginPasskeyLoginResponse{SessionId: "login-session", CredentialRequestOptionsJson: []byte(`{"publicKey":{"challenge":"ZmFrZQ","timeout":60000,"userVerification":"preferred"}}`)}, nil
}

func (fakeAuthClient) FinishPasskeyLogin(context.Context, *authv1.FinishPasskeyLoginRequest, ...grpc.CallOption) (*authv1.FinishPasskeyLoginResponse, error) {
	return &authv1.FinishPasskeyLoginResponse{User: &authv1.User{Id: "user-1"}}, nil
}

func (fakeAuthClient) CreateWebSession(context.Context, *authv1.CreateWebSessionRequest, ...grpc.CallOption) (*authv1.CreateWebSessionResponse, error) {
	return &authv1.CreateWebSessionResponse{Session: &authv1.WebSession{Id: "ws-1", UserId: "user-1"}}, nil
}

func (fakeAuthClient) GetWebSession(context.Context, *authv1.GetWebSessionRequest, ...grpc.CallOption) (*authv1.GetWebSessionResponse, error) {
	return &authv1.GetWebSessionResponse{Session: &authv1.WebSession{Id: "ws-1", UserId: "user-1"}, User: &authv1.User{Id: "user-1"}}, nil
}

func (fakeAuthClient) RevokeWebSession(context.Context, *authv1.RevokeWebSessionRequest, ...grpc.CallOption) (*authv1.RevokeWebSessionResponse, error) {
	return &authv1.RevokeWebSessionResponse{}, nil
}
