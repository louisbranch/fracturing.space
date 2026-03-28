package publicauth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	publicauthgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/sessioncookie"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	"google.golang.org/grpc"
)

func TestComposeWrappersBuildExpectedSurfaceModules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		build      func(CompositionConfig) module.Module
		wantID     string
		wantPrefix string
	}{
		{name: "shell", build: func(config CompositionConfig) module.Module { return ComposeShell(config) }, wantID: "public", wantPrefix: routepath.Root},
		{name: "passkeys", build: func(config CompositionConfig) module.Module { return ComposePasskeys(config) }, wantID: "public-passkeys", wantPrefix: routepath.PasskeysPrefix},
		{name: "auth redirect", build: func(config CompositionConfig) module.Module { return ComposeAuthRedirect(config) }, wantID: "public-auth-redirect", wantPrefix: routepath.AuthPrefix},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			module := tc.build(CompositionConfig{
				AuthClient:  fakeAuthClient{},
				AuthBaseURL: "https://auth.example.test",
			})
			if got := module.ID(); got != tc.wantID {
				t.Fatalf("ID() = %q, want %q", got, tc.wantID)
			}
			mount, err := module.Mount()
			if err != nil {
				t.Fatalf("Mount() error = %v", err)
			}
			if got := mount.Prefix; got != tc.wantPrefix {
				t.Fatalf("prefix = %q, want %q", got, tc.wantPrefix)
			}
		})
	}
}

func TestBindAuthDependencyGuardsNilInputsAndAssignsClient(t *testing.T) {
	t.Parallel()

	BindAuthDependency(nil, new(grpc.ClientConn))

	deps := &Dependencies{}
	BindAuthDependency(deps, nil)
	if deps.AuthClient != nil {
		t.Fatalf("AuthClient = %#v, want nil after nil conn", deps.AuthClient)
	}

	BindAuthDependency(deps, new(grpc.ClientConn))
	if deps.AuthClient == nil {
		t.Fatal("AuthClient = nil, want client")
	}
}

func TestHandleRecoveryGetRendersRecoveryPage(t *testing.T) {
	t.Parallel()

	module := newModuleFromGatewayWithFactory(nil, "", NewShell)
	mount, err := module.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.LoginRecovery+"?pending_id=pending-1&next=%2Finvite%2Finv-1", nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, want := range []string{
		routepath.PasskeyRecoveryStart,
		routepath.PasskeyRecoveryFinish,
		"pending-1",
		"/invite/inv-1",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q", want)
		}
	}
}

func TestRecoveryStartReturnsTypedJSONContract(t *testing.T) {
	t.Parallel()

	module := newModuleFromGatewayWithFactory(publicauthgateway.NewGRPCGateway(fakeAuthClient{}), "", NewPasskeys)
	mount, err := module.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, routepath.PasskeyRecoveryStart, strings.NewReader(`{"username":"louis","recovery_code":"ABCD-EFGH"}`))
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want JSON", got)
	}

	var payload recoveryStartResponse
	if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.RecoverySessionID != "recover-1" {
		t.Fatalf("recovery_session_id = %q, want %q", payload.RecoverySessionID, "recover-1")
	}
	if payload.SessionID != "recovery-passkey-1" {
		t.Fatalf("session_id = %q, want %q", payload.SessionID, "recovery-passkey-1")
	}
	if strings.TrimSpace(string(payload.PublicKey)) != `{"publicKey":{}}` {
		t.Fatalf("public_key = %s", payload.PublicKey)
	}
}

func TestRecoveryFinishReturnsTypedJSONContractAndCookies(t *testing.T) {
	t.Parallel()

	module := newModuleFromGatewayWithFactory(publicauthgateway.NewGRPCGateway(fakeAuthClient{}), "", NewPasskeys)
	mount, err := module.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, routepath.PasskeyRecoveryFinish, strings.NewReader(`{"recovery_session_id":"recover-1","session_id":"recovery-passkey-1","pending_id":"pending-1","next":" /invite/inv-1 ","credential":{"id":"cred-1"}}`))
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var payload passkeyLoginFinishResponse
	if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.RedirectURL != routepath.LoginRecoveryCode {
		t.Fatalf("redirect_url = %q, want %q", payload.RedirectURL, routepath.LoginRecoveryCode)
	}
	if cookie := responseCookieByName(rr, sessioncookie.Name); cookie == nil || cookie.Value == "" {
		t.Fatal("expected web session cookie")
	}
	if cookie := responseCookieByName(rr, recoveryRevealCookieName); cookie == nil || cookie.Value == "" {
		t.Fatalf("expected %q cookie", recoveryRevealCookieName)
	}
}
