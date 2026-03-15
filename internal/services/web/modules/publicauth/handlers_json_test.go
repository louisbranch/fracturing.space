package publicauth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	publicauthgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestPasskeyRegisterStartReturnsTypedJSONContract(t *testing.T) {
	t.Parallel()

	m := newModuleFromGatewayWithFactory(publicauthgateway.NewGRPCGateway(fakeAuthClient{}), "", NewPasskeys)
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, routepath.PasskeyRegisterStart, strings.NewReader(`{"username":"louis"}`))
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want JSON", got)
	}

	var payload passkeyChallengeResponse
	if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.SessionID != "reg-1" {
		t.Fatalf("session_id = %q, want %q", payload.SessionID, "reg-1")
	}
	if strings.TrimSpace(string(payload.PublicKey)) != `{"publicKey":{}}` {
		t.Fatalf("public_key = %s", payload.PublicKey)
	}
}

func TestPasskeyRegisterFinishReturnsTypedJSONContract(t *testing.T) {
	t.Parallel()

	m := newModuleFromGatewayWithFactory(publicauthgateway.NewGRPCGateway(fakeAuthClient{}), "", NewPasskeys)
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, routepath.PasskeyRegisterFinish, strings.NewReader(`{"session_id":"reg-1","credential":{"id":"cred-1"}}`))
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var payload passkeyRegisterFinishResponse
	if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.RedirectURL != routepath.LoginRecoveryCode {
		t.Fatalf("redirect_url = %q, want %q", payload.RedirectURL, routepath.LoginRecoveryCode)
	}
}

func TestPasskeyLoginFinishReturnsTypedJSONContract(t *testing.T) {
	t.Parallel()

	m := newModuleFromGatewayWithFactory(publicauthgateway.NewGRPCGateway(fakeAuthClient{}), "", NewPasskeys)
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, routepath.PasskeyLoginFinish, strings.NewReader(`{"session_id":"login-1","credential":{"id":"cred-1"}}`))
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var payload passkeyLoginFinishResponse
	if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.RedirectURL != routepath.AppDashboard {
		t.Fatalf("redirect_url = %q, want %q", payload.RedirectURL, routepath.AppDashboard)
	}
}

func TestUsernameCheckReturnsTypedJSONContract(t *testing.T) {
	t.Parallel()

	m := newModuleFromGatewayWithFactory(publicauthgateway.NewGRPCGateway(fakeAuthClient{}), "", NewPasskeys)
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, routepath.PasskeyRegisterCheck, strings.NewReader(`{"username":"louis"}`))
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var payload usernameAvailabilityResponse
	if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.CanonicalUsername != "louis" {
		t.Fatalf("canonical_username = %q, want %q", payload.CanonicalUsername, "louis")
	}
	if payload.State != "available" {
		t.Fatalf("state = %q, want %q", payload.State, "available")
	}
}
