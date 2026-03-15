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
	flashnotice "github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestresolver"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/sessioncookie"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRecoveryPageURLAndLoginPageURLPreserveSafeState(t *testing.T) {
	t.Parallel()

	h := newHandlersFromGateway(nil, "", requestmeta.SchemePolicy{})
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

	h := newHandlersFromGateway(nil, "", requestmeta.SchemePolicy{})
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

	h := newHandlersFromGateway(nil, "", requestmeta.SchemePolicy{})
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

	h := newHandlersFromGateway(nil, "", requestmeta.SchemePolicy{})

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

	h := newHandlersFromGateway(nil, "", requestmeta.SchemePolicy{})
	seedReq := httptest.NewRequest(http.MethodGet, routepath.LoginRecoveryCode, nil)
	seedResp := httptest.NewRecorder()
	writeRecoveryRevealState(seedResp, seedReq, requestmeta.SchemePolicy{}, recoveryRevealState{
		Code: "ABCD-EFGH",
		Next: "/invite/inv-1",
		Mode: recoveryRevealModeRecovery,
	})
	var revealCookie *http.Cookie
	for _, cookie := range seedResp.Result().Cookies() {
		if cookie.Name == recoveryRevealCookieName {
			revealCookie = cookie
			break
		}
	}
	if revealCookie == nil {
		t.Fatal("expected reveal cookie")
	}
	form := url.Values{}
	form.Set("acknowledged", "yes")
	req := httptest.NewRequest(http.MethodPost, "http://app.example.test"+routepath.LoginRecoveryCodeAcknowledge, strings.NewReader(form.Encode()))
	req.Host = "app.example.test"
	req.Header.Set("Origin", "http://app.example.test")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(revealCookie)
	rr := httptest.NewRecorder()

	h.handleRecoveryCodeAcknowledge(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != "/invite/inv-1" {
		t.Fatalf("Location = %q, want %q", got, "/invite/inv-1")
	}
}

func TestHandleRecoveryCodeAcknowledgeSignupActivatesSession(t *testing.T) {
	t.Parallel()

	gateway := &signupAcknowledgeGatewayStub{
		ackResult: publicauthapp.PasskeyFinish{SessionID: "sess-1", UserID: "user-1"},
	}
	h := newHandlersFromGateway(gateway, "", requestmeta.SchemePolicy{})
	revealCookie := mustSeedRecoveryRevealCookie(t, recoveryRevealState{
		Code:      "ABCD-EFGH",
		SessionID: "reg-1",
		Next:      "/invite/inv-1",
		Mode:      recoveryRevealModeSignup,
	})

	form := url.Values{}
	form.Set("acknowledged", "yes")
	req := httptest.NewRequest(http.MethodPost, "http://app.example.test"+routepath.LoginRecoveryCodeAcknowledge, strings.NewReader(form.Encode()))
	req.Host = "app.example.test"
	req.Header.Set("Origin", "http://app.example.test")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(revealCookie)
	rr := httptest.NewRecorder()

	h.handleRecoveryCodeAcknowledge(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if gateway.ackSessionID != "reg-1" {
		t.Fatalf("ack session id = %q, want %q", gateway.ackSessionID, "reg-1")
	}
	if got := rr.Header().Get("Location"); got != "/invite/inv-1" {
		t.Fatalf("Location = %q, want %q", got, "/invite/inv-1")
	}
	if cookie := responseCookieByName(rr, sessioncookie.Name); cookie == nil || cookie.Value == "" {
		t.Fatalf("expected %q session cookie", sessioncookie.Name)
	}
	if cookie := responseCookieByName(rr, recoveryRevealCookieName); cookie == nil || cookie.MaxAge != -1 {
		t.Fatalf("expected cleared %q cookie", recoveryRevealCookieName)
	}
}

func TestHandleRecoveryCodeAcknowledgeSignupExpiredRedirectsToLogin(t *testing.T) {
	t.Parallel()

	gateway := &signupAcknowledgeGatewayStub{
		ackErr: status.Error(codes.FailedPrecondition, "expired"),
	}
	h := newHandlersFromGateway(gateway, "", requestmeta.SchemePolicy{})
	revealCookie := mustSeedRecoveryRevealCookie(t, recoveryRevealState{
		Code:      "ABCD-EFGH",
		SessionID: "reg-1",
		PendingID: "pending-1",
		Next:      "/invite/inv-1",
		Mode:      recoveryRevealModeSignup,
	})

	form := url.Values{}
	form.Set("acknowledged", "yes")
	req := httptest.NewRequest(http.MethodPost, "http://app.example.test"+routepath.LoginRecoveryCodeAcknowledge, strings.NewReader(form.Encode()))
	req.Host = "app.example.test"
	req.Header.Set("Origin", "http://app.example.test")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(revealCookie)
	rr := httptest.NewRecorder()

	h.handleRecoveryCodeAcknowledge(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.Login+"?next=%2Finvite%2Finv-1&pending_id=pending-1" {
		t.Fatalf("Location = %q", got)
	}
	if cookie := responseCookieByName(rr, recoveryRevealCookieName); cookie == nil || cookie.MaxAge != -1 {
		t.Fatalf("expected cleared %q cookie", recoveryRevealCookieName)
	}
	if cookie := responseCookieByName(rr, flashnotice.CookieName); cookie == nil || cookie.Value == "" {
		t.Fatalf("expected %q flash cookie", flashnotice.CookieName)
	}
}

func TestRedirectAuthenticatedToAppUsesValidatedNextPath(t *testing.T) {
	t.Parallel()

	h := newHandlersFromGateway(publicauthGatewayStub{}, "", requestmeta.SchemePolicy{}, requestresolver.NewPrincipal(
		nil,
		func(*http.Request) bool { return true },
		nil,
		nil,
		nil,
	))
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

type publicauthGatewayStub struct{}

type signupAcknowledgeGatewayStub struct {
	publicauthGatewayStub
	ackResult    publicauthapp.PasskeyFinish
	ackErr       error
	ackSessionID string
	ackPendingID string
}

func (publicauthGatewayStub) BeginAccountRegistration(context.Context, string) (publicauthapp.PasskeyChallenge, error) {
	return publicauthapp.PasskeyChallenge{}, nil
}

func (publicauthGatewayStub) CheckUsernameAvailability(context.Context, string) (publicauthapp.UsernameAvailability, error) {
	return publicauthapp.UsernameAvailability{}, nil
}

func (publicauthGatewayStub) FinishAccountRegistration(context.Context, string, json.RawMessage) (publicauthapp.PasskeyRegistrationReveal, error) {
	return publicauthapp.PasskeyRegistrationReveal{}, nil
}

func (publicauthGatewayStub) AcknowledgeAccountRegistration(context.Context, string, string) (publicauthapp.PasskeyFinish, error) {
	return publicauthapp.PasskeyFinish{}, nil
}

func (s *signupAcknowledgeGatewayStub) AcknowledgeAccountRegistration(_ context.Context, sessionID string, pendingID string) (publicauthapp.PasskeyFinish, error) {
	s.ackSessionID = sessionID
	s.ackPendingID = pendingID
	return s.ackResult, s.ackErr
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

func (publicauthGatewayStub) RevokeWebSession(context.Context, string) error {
	return nil
}

func mustSeedRecoveryRevealCookie(t *testing.T, state recoveryRevealState) *http.Cookie {
	t.Helper()

	seedReq := httptest.NewRequest(http.MethodGet, routepath.LoginRecoveryCode, nil)
	seedResp := httptest.NewRecorder()
	writeRecoveryRevealState(seedResp, seedReq, requestmeta.SchemePolicy{}, state)
	cookie := responseCookieByName(seedResp, recoveryRevealCookieName)
	if cookie == nil {
		t.Fatal("expected reveal cookie")
	}
	return cookie
}

func responseCookieByName(rr *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, cookie := range rr.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}
