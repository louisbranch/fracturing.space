package publicauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	publicauthapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/sessioncookie"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestRecoveryPageURLAndLoginPageURLPreserveSafeState(t *testing.T) {
	t.Parallel()

	h := newHandlers(publicauthapp.NewService(nil, ""), requestmeta.SchemePolicy{})
	req := httptest.NewRequest(http.MethodGet, routepath.Login+"?pending_id=pending-1&next=%2Finvite%2Finv-1", nil)

	if got := h.recoveryPageURL(req); got != routepath.LoginRecovery+"?next=%2Finvite%2Finv-1&pending_id=pending-1" {
		t.Fatalf("recoveryPageURL() = %q", got)
	}
	if got := h.loginPageURL(req); got != routepath.Login+"?next=%2Finvite%2Finv-1&pending_id=pending-1" {
		t.Fatalf("loginPageURL() = %q", got)
	}
}

func TestRecoveryPageURLRejectsUnsafeNext(t *testing.T) {
	t.Parallel()

	h := newHandlers(publicauthapp.NewService(nil, ""), requestmeta.SchemePolicy{})
	req := httptest.NewRequest(http.MethodGet, routepath.Login+"?pending_id=pending-1&next=https://evil.example/app", nil)

	if got := h.nextPath(req); got != "" {
		t.Fatalf("nextPath() = %q, want empty", got)
	}
	if got := h.recoveryPageURL(req); got != routepath.LoginRecovery+"?pending_id=pending-1" {
		t.Fatalf("recoveryPageURL() = %q", got)
	}
}

func TestHandleAuthLoginRedirectsToLoginWithSafeNext(t *testing.T) {
	t.Parallel()

	h := newHandlers(publicauthapp.NewService(nil, ""), requestmeta.SchemePolicy{})
	req := httptest.NewRequest(http.MethodGet, routepath.AuthLogin+"?pending_id=pending-1&next=%2Finvite%2Finv-1", nil)
	rr := httptest.NewRecorder()

	h.handleAuthLogin(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.Login+"?next=%2Finvite%2Finv-1&pending_id=pending-1" {
		t.Fatalf("Location = %q", got)
	}
}

func TestHandleRecoveryCodeAcknowledgeRequiresSameOriginAndAcknowledgement(t *testing.T) {
	t.Parallel()

	h := newHandlers(publicauthapp.NewService(nil, ""), requestmeta.SchemePolicy{})

	forbiddenReq := httptest.NewRequest(http.MethodPost, routepath.LoginRecoveryCodeAcknowledge, strings.NewReader("acknowledged=yes"))
	forbiddenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	forbiddenRR := httptest.NewRecorder()
	h.handleRecoveryCodeAcknowledge(forbiddenRR, forbiddenReq)
	if forbiddenRR.Code != http.StatusForbidden {
		t.Fatalf("forbidden status = %d, want %d", forbiddenRR.Code, http.StatusForbidden)
	}

	badReq := httptest.NewRequest(http.MethodPost, "http://app.example.test"+routepath.LoginRecoveryCodeAcknowledge, strings.NewReader(""))
	badReq.Host = "app.example.test"
	badReq.Header.Set("Origin", "http://app.example.test")
	badReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	badRR := httptest.NewRecorder()
	h.handleRecoveryCodeAcknowledge(badRR, badReq)
	if badRR.Code != http.StatusBadRequest {
		t.Fatalf("bad request status = %d, want %d", badRR.Code, http.StatusBadRequest)
	}
}

func TestHandleRecoveryCodeAcknowledgeRedirectsToSafeNext(t *testing.T) {
	t.Parallel()

	h := newHandlers(publicauthapp.NewService(nil, ""), requestmeta.SchemePolicy{})
	form := url.Values{}
	form.Set("acknowledged", "yes")
	form.Set("next", "/invite/inv-1")
	req := httptest.NewRequest(http.MethodPost, "http://app.example.test"+routepath.LoginRecoveryCodeAcknowledge, strings.NewReader(form.Encode()))
	req.Host = "app.example.test"
	req.Header.Set("Origin", "http://app.example.test")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	h.handleRecoveryCodeAcknowledge(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != "/invite/inv-1" {
		t.Fatalf("Location = %q, want %q", got, "/invite/inv-1")
	}
}

func TestRedirectAuthenticatedToAppUsesValidatedNextPath(t *testing.T) {
	t.Parallel()

	h := newHandlers(publicauthapp.NewService(publicauthGatewayStub{validSession: true}, ""), requestmeta.SchemePolicy{})
	req := httptest.NewRequest(http.MethodGet, routepath.Login+"?next=%2Finvite%2Finv-1", nil)
	req.AddCookie(&http.Cookie{Name: sessioncookie.Name, Value: "sess-1"})
	rr := httptest.NewRecorder()

	if ok := h.redirectAuthenticatedToApp(rr, req); !ok {
		t.Fatal("redirectAuthenticatedToApp() = false, want true")
	}
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != "/invite/inv-1" {
		t.Fatalf("Location = %q, want %q", got, "/invite/inv-1")
	}
}

type publicauthGatewayStub struct {
	validSession bool
}

func (publicauthGatewayStub) BeginAccountRegistration(context.Context, string) (publicauthapp.PasskeyChallenge, error) {
	return publicauthapp.PasskeyChallenge{}, nil
}

func (publicauthGatewayStub) CheckUsernameAvailability(context.Context, string) (publicauthapp.UsernameAvailability, error) {
	return publicauthapp.UsernameAvailability{}, nil
}

func (publicauthGatewayStub) FinishAccountRegistration(context.Context, string, json.RawMessage) (publicauthapp.PasskeyFinish, error) {
	return publicauthapp.PasskeyFinish{}, nil
}

func (publicauthGatewayStub) BeginPasskeyLogin(context.Context, string) (publicauthapp.PasskeyChallenge, error) {
	return publicauthapp.PasskeyChallenge{}, nil
}

func (publicauthGatewayStub) FinishPasskeyLogin(context.Context, string, json.RawMessage, string) (string, error) {
	return "", nil
}

func (publicauthGatewayStub) BeginAccountRecovery(context.Context, string, string) (string, error) {
	return "", nil
}

func (publicauthGatewayStub) BeginRecoveryPasskeyRegistration(context.Context, string) (publicauthapp.PasskeyChallenge, error) {
	return publicauthapp.PasskeyChallenge{}, nil
}

func (publicauthGatewayStub) FinishRecoveryPasskeyRegistration(context.Context, string, string, json.RawMessage, string) (publicauthapp.PasskeyFinish, error) {
	return publicauthapp.PasskeyFinish{}, nil
}

func (publicauthGatewayStub) CreateWebSession(context.Context, string) (string, error) {
	return "", nil
}

func (s publicauthGatewayStub) HasValidWebSession(context.Context, string) bool {
	return s.validSession
}

func (publicauthGatewayStub) RevokeWebSession(context.Context, string) error {
	return nil
}
