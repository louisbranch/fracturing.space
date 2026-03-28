package app

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestPageAndSessionServicesResolvePostAuthRedirect(t *testing.T) {
	t.Parallel()

	page := NewPageService(" https://auth.example.test/base/ ")
	session := NewSessionService(&publicauthSessionGatewayStub{}, " https://auth.example.test/base/ ")

	if got := page.ResolvePostAuthRedirect("", "/invite/inv-1?from=mail"); got != "/invite/inv-1?from=mail" {
		t.Fatalf("page ResolvePostAuthRedirect() = %q, want invite path", got)
	}
	if got := session.ResolvePostAuthRedirect("pending-1", "/app/dashboard"); got != "https://auth.example.test/base/authorize/consent?pending_id=pending-1" {
		t.Fatalf("session ResolvePostAuthRedirect() = %q", got)
	}

	fallbackPage := NewPageService("://bad auth base")
	if got := fallbackPage.ResolvePostAuthRedirect("pending-1", "/invite/inv-1"); got != routepath.AppDashboard {
		t.Fatalf("fallback ResolvePostAuthRedirect() = %q, want %q", got, routepath.AppDashboard)
	}
}

func TestSessionServiceRevokeWebSessionBranches(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	gateway := &publicauthSessionGatewayStub{}
	svc := NewSessionService(gateway, "")

	if err := svc.RevokeWebSession(ctx, "   "); err != nil {
		t.Fatalf("RevokeWebSession(blank) error = %v", err)
	}
	if gateway.lastSessionID != "" {
		t.Fatalf("lastSessionID = %q, want empty", gateway.lastSessionID)
	}

	if err := svc.RevokeWebSession(ctx, "  web-1  "); err != nil {
		t.Fatalf("RevokeWebSession(trimmed) error = %v", err)
	}
	if gateway.lastSessionID != "web-1" {
		t.Fatalf("lastSessionID = %q, want %q", gateway.lastSessionID, "web-1")
	}

	unavailable := NewSessionService(nil, "")
	err := unavailable.RevokeWebSession(ctx, "web-1")
	if err == nil {
		t.Fatal("RevokeWebSession(nil gateway) error = nil, want unavailable")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

func TestRecoveryServiceStartAndFinishBranches(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	gateway := &publicauthRecoveryGatewayStub{
		recoverySessionID: " recovery-1 ",
		beginChallenge:    PasskeyChallenge{SessionID: " passkey-1 ", PublicKey: json.RawMessage(`{"publicKey":{}}`)},
		finishResult:      PasskeyFinish{SessionID: " web-1 ", UserID: " user-1 ", RecoveryCode: " CODE-1 "},
	}
	svc := NewRecoveryService(gateway)

	start, err := svc.RecoveryStart(ctx, "  louis  ", "  CODE-1  ")
	if err != nil {
		t.Fatalf("RecoveryStart() error = %v", err)
	}
	if gateway.lastUsername != "louis" || gateway.lastRecoveryCode != "CODE-1" {
		t.Fatalf("gateway start args = (%q, %q), want trimmed values", gateway.lastUsername, gateway.lastRecoveryCode)
	}
	if start.RecoverySessionID != "recovery-1" || start.SessionID != "passkey-1" {
		t.Fatalf("start = %+v", start)
	}

	finish, err := svc.RecoveryFinish(ctx, " rec-1 ", " sess-1 ", json.RawMessage(`{}`), " pending-1 ")
	if err != nil {
		t.Fatalf("RecoveryFinish() error = %v", err)
	}
	if gateway.lastFinishPendingID != "pending-1" {
		t.Fatalf("lastFinishPendingID = %q, want %q", gateway.lastFinishPendingID, "pending-1")
	}
	if finish.UserID != "user-1" || finish.SessionID != "web-1" || finish.RecoveryCode != "CODE-1" {
		t.Fatalf("finish = %+v", finish)
	}

	missingRecoverySession := NewRecoveryService(&publicauthRecoveryGatewayStub{
		beginChallenge: PasskeyChallenge{SessionID: "passkey-1"},
	})
	err = nil
	_, err = missingRecoverySession.RecoveryStart(ctx, "louis", "code")
	if err == nil || apperrors.HTTPStatus(err) != http.StatusServiceUnavailable {
		t.Fatalf("RecoveryStart(missing recovery session) err = %v", err)
	}

	missingPasskeySession := NewRecoveryService(&publicauthRecoveryGatewayStub{
		recoverySessionID: "recovery-1",
		beginChallenge:    PasskeyChallenge{},
	})
	_, err = missingPasskeySession.RecoveryStart(ctx, "louis", "code")
	if err == nil || apperrors.HTTPStatus(err) != http.StatusServiceUnavailable {
		t.Fatalf("RecoveryStart(missing passkey session) err = %v", err)
	}

	invalidFinish := NewRecoveryService(&publicauthRecoveryGatewayStub{})
	_, err = invalidFinish.RecoveryFinish(ctx, "", "sess-1", json.RawMessage(`{}`), "")
	if err == nil || apperrors.HTTPStatus(err) != http.StatusBadRequest {
		t.Fatalf("RecoveryFinish(missing recovery session) err = %v", err)
	}

	missingRecoveryCode := NewRecoveryService(&publicauthRecoveryGatewayStub{
		finishResult: PasskeyFinish{SessionID: "web-1", UserID: "user-1"},
	})
	_, err = missingRecoveryCode.RecoveryFinish(ctx, "rec-1", "sess-1", json.RawMessage(`{}`), "")
	if err == nil || apperrors.HTTPStatus(err) != http.StatusBadRequest {
		t.Fatalf("RecoveryFinish(missing recovery code) err = %v", err)
	}
}

func TestUnavailableGatewayReturnsUnavailableAcrossSurface(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	gateway := NewUnavailableGateway()
	checkUnavailable := func(t *testing.T, err error) {
		t.Helper()
		if err == nil {
			t.Fatal("expected unavailable error")
		}
		if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
			t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
		}
	}

	t.Run("passkey and availability", func(t *testing.T) {
		_, err := gateway.BeginAccountRegistration(ctx, "louis")
		checkUnavailable(t, err)
		_, err = gateway.CheckUsernameAvailability(ctx, "louis")
		checkUnavailable(t, err)
		_, err = gateway.FinishAccountRegistration(ctx, "sess-1", json.RawMessage(`{}`))
		checkUnavailable(t, err)
		_, err = gateway.AcknowledgeAccountRegistration(ctx, "sess-1", "pending-1")
		checkUnavailable(t, err)
		_, err = gateway.BeginPasskeyLogin(ctx, "louis")
		checkUnavailable(t, err)
		_, err = gateway.FinishPasskeyLogin(ctx, "sess-1", json.RawMessage(`{}`), "pending-1")
		checkUnavailable(t, err)
		_, err = gateway.CreateWebSession(ctx, "user-1")
		checkUnavailable(t, err)
	})

	t.Run("recovery and session", func(t *testing.T) {
		_, err := gateway.BeginAccountRecovery(ctx, "louis", "code")
		checkUnavailable(t, err)
		_, err = gateway.BeginRecoveryPasskeyRegistration(ctx, "rec-1")
		checkUnavailable(t, err)
		_, err = gateway.FinishRecoveryPasskeyRegistration(ctx, "rec-1", "sess-1", json.RawMessage(`{}`), "pending-1")
		checkUnavailable(t, err)
		checkUnavailable(t, gateway.RevokeWebSession(ctx, "web-1"))
	})
}

type publicauthSessionGatewayStub struct {
	lastSessionID string
	revokeErr     error
}

func (s *publicauthSessionGatewayStub) RevokeWebSession(_ context.Context, sessionID string) error {
	s.lastSessionID = sessionID
	return s.revokeErr
}

type publicauthRecoveryGatewayStub struct {
	recoverySessionID   string
	beginChallenge      PasskeyChallenge
	finishResult        PasskeyFinish
	beginRecoveryErr    error
	beginChallengeErr   error
	finishRecoveryErr   error
	lastUsername        string
	lastRecoveryCode    string
	lastFinishPendingID string
}

func (s *publicauthRecoveryGatewayStub) BeginAccountRecovery(_ context.Context, username string, recoveryCode string) (string, error) {
	s.lastUsername = username
	s.lastRecoveryCode = recoveryCode
	return s.recoverySessionID, s.beginRecoveryErr
}

func (s *publicauthRecoveryGatewayStub) BeginRecoveryPasskeyRegistration(_ context.Context, recoverySessionID string) (PasskeyChallenge, error) {
	return s.beginChallenge, s.beginChallengeErr
}

func (s *publicauthRecoveryGatewayStub) FinishRecoveryPasskeyRegistration(_ context.Context, recoverySessionID string, sessionID string, credential json.RawMessage, pendingID string) (PasskeyFinish, error) {
	s.lastFinishPendingID = pendingID
	return s.finishResult, s.finishRecoveryErr
}
