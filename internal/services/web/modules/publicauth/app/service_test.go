package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

func TestNewServiceFailsClosedWhenGatewayMissing(t *testing.T) {
	t.Parallel()

	svc := NewService(nil)
	_, err := svc.PasskeyLoginStart(context.Background())
	if err == nil {
		t.Fatalf("expected unavailable error when auth gateway is missing")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}
	if svc.HasValidWebSession(context.Background(), "ws-local") {
		t.Fatalf("expected missing gateway to reject web sessions")
	}
}

func TestPasskeyLoginFinishValidatesInput(t *testing.T) {
	t.Parallel()

	svc := NewService(&authGatewayStub{})
	if _, err := svc.PasskeyLoginFinish(context.Background(), "", json.RawMessage(`{"id":"cred-1"}`)); err == nil {
		t.Fatalf("expected session id validation error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
	}

	if _, err := svc.PasskeyLoginFinish(context.Background(), "session-1", nil); err == nil {
		t.Fatalf("expected credential validation error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
	}
}

func TestPasskeyLoginFinishCreatesWebSession(t *testing.T) {
	t.Parallel()

	svc := NewService(&authGatewayStub{finishPasskeyLoginResult: "user-7", createWebSessionResult: "ws-7"})
	finished, err := svc.PasskeyLoginFinish(context.Background(), "session-1", json.RawMessage(`{"id":"cred-1"}`))
	if err != nil {
		t.Fatalf("PasskeyLoginFinish() error = %v", err)
	}
	if finished.UserID != "user-7" {
		t.Fatalf("UserID = %q, want %q", finished.UserID, "user-7")
	}
	if finished.SessionID != "ws-7" {
		t.Fatalf("SessionID = %q, want %q", finished.SessionID, "ws-7")
	}
}

func TestPasskeyLoginFinishNormalizesDelegatedValues(t *testing.T) {
	t.Parallel()

	stub := &authGatewayStub{finishPasskeyLoginResult: " user-9 ", createWebSessionResult: " ws-9 "}
	svc := NewService(stub)
	finished, err := svc.PasskeyLoginFinish(context.Background(), " session-9 ", json.RawMessage(`{"id":"cred-9"}`))
	if err != nil {
		t.Fatalf("PasskeyLoginFinish() error = %v", err)
	}
	if stub.lastFinishPasskeyLoginSessionID != "session-9" {
		t.Fatalf("finish login session id = %q, want %q", stub.lastFinishPasskeyLoginSessionID, "session-9")
	}
	if stub.lastCreateWebSessionUserID != "user-9" {
		t.Fatalf("create session user id = %q, want %q", stub.lastCreateWebSessionUserID, "user-9")
	}
	if finished.UserID != "user-9" {
		t.Fatalf("UserID = %q, want %q", finished.UserID, "user-9")
	}
	if finished.SessionID != "ws-9" {
		t.Fatalf("SessionID = %q, want %q", finished.SessionID, "ws-9")
	}
}

func TestPasskeyLoginFinishRejectsBlankGatewayValues(t *testing.T) {
	t.Parallel()

	svcBlankUser := NewService(&authGatewayStub{finishPasskeyLoginResult: "   ", createWebSessionResult: "ws-1"})
	if _, err := svcBlankUser.PasskeyLoginFinish(context.Background(), "session-1", json.RawMessage(`{"id":"cred-1"}`)); err == nil {
		t.Fatalf("expected blank gateway user id error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}

	svcBlankSession := NewService(&authGatewayStub{finishPasskeyLoginResult: "user-1", createWebSessionResult: "   "})
	if _, err := svcBlankSession.PasskeyLoginFinish(context.Background(), "session-1", json.RawMessage(`{"id":"cred-1"}`)); err == nil {
		t.Fatalf("expected blank gateway session id error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

func TestPasskeyRegisterStartValidatesEmailAndGatewayErrors(t *testing.T) {
	t.Parallel()

	svc := NewService(&authGatewayStub{})
	if _, err := svc.PasskeyRegisterStart(context.Background(), "   "); err == nil {
		t.Fatalf("expected email validation error")
	}

	svcCreateUserFail := NewService(&authGatewayStub{createUserErr: errors.New("boom")})
	if _, err := svcCreateUserFail.PasskeyRegisterStart(context.Background(), "user@example.com"); err == nil {
		t.Fatalf("expected create user error")
	}

	svcRegisterFail := NewService(&authGatewayStub{beginPasskeyRegistrationErr: errors.New("boom")})
	if _, err := svcRegisterFail.PasskeyRegisterStart(context.Background(), "user@example.com"); err == nil {
		t.Fatalf("expected begin registration error")
	}
}

func TestPasskeyRegisterStartNormalizesDelegatedValues(t *testing.T) {
	t.Parallel()

	stub := &authGatewayStub{createUserResult: " user-5 "}
	svc := NewService(stub)
	result, err := svc.PasskeyRegisterStart(context.Background(), " user@example.com ")
	if err != nil {
		t.Fatalf("PasskeyRegisterStart() error = %v", err)
	}
	if stub.lastCreateUserEmail != "user@example.com" {
		t.Fatalf("create user email = %q, want %q", stub.lastCreateUserEmail, "user@example.com")
	}
	if stub.lastBeginPasskeyRegistrationUserID != "user-5" {
		t.Fatalf("begin registration user id = %q, want %q", stub.lastBeginPasskeyRegistrationUserID, "user-5")
	}
	if result.UserID != "user-5" {
		t.Fatalf("UserID = %q, want %q", result.UserID, "user-5")
	}
}

func TestPasskeyRegisterStartRejectsBlankGatewayValues(t *testing.T) {
	t.Parallel()

	svcBlankUser := NewService(&authGatewayStub{createUserResult: "   "})
	if _, err := svcBlankUser.PasskeyRegisterStart(context.Background(), "user@example.com"); err == nil {
		t.Fatalf("expected blank gateway user id error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}

	svcBlankSession := NewService(&authGatewayStub{createUserResult: "user-1", beginPasskeyRegistrationSessionID: "   "})
	if _, err := svcBlankSession.PasskeyRegisterStart(context.Background(), "user@example.com"); err == nil {
		t.Fatalf("expected blank gateway challenge session id error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

func TestPasskeyRegisterFinishNormalizesDelegatedValues(t *testing.T) {
	t.Parallel()

	stub := &authGatewayStub{finishPasskeyRegistrationResult: " user-8 "}
	svc := NewService(stub)
	result, err := svc.PasskeyRegisterFinish(context.Background(), " session-8 ", json.RawMessage(`{"id":"cred-8"}`))
	if err != nil {
		t.Fatalf("PasskeyRegisterFinish() error = %v", err)
	}
	if stub.lastFinishPasskeyRegistrationSessionID != "session-8" {
		t.Fatalf("finish registration session id = %q, want %q", stub.lastFinishPasskeyRegistrationSessionID, "session-8")
	}
	if result.UserID != "user-8" {
		t.Fatalf("UserID = %q, want %q", result.UserID, "user-8")
	}
}

func TestPasskeyRegisterFinishRejectsBlankGatewayUserID(t *testing.T) {
	t.Parallel()

	svc := NewService(&authGatewayStub{finishPasskeyRegistrationResult: "   "})
	if _, err := svc.PasskeyRegisterFinish(context.Background(), "session-8", json.RawMessage(`{"id":"cred-8"}`)); err == nil {
		t.Fatalf("expected blank gateway user id error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

func TestRevokeWebSessionHandlesEmptyAndGatewayError(t *testing.T) {
	t.Parallel()

	stub := &authGatewayStub{}
	svc := NewService(stub)
	if err := svc.RevokeWebSession(context.Background(), ""); err != nil {
		t.Fatalf("RevokeWebSession(empty) error = %v", err)
	}
	if stub.revokeCalled {
		t.Fatalf("expected revoke not called for empty session id")
	}

	svc = NewService(&authGatewayStub{revokeWebSessionErr: errors.New("boom")})
	err := svc.RevokeWebSession(context.Background(), " session-1 ")
	if err == nil {
		t.Fatalf("expected revoke failure")
	}
}

func TestHasValidWebSessionNormalizesSessionID(t *testing.T) {
	t.Parallel()

	stub := &authGatewayStub{hasValidSessionResult: true}
	svc := NewService(stub)
	if !svc.HasValidWebSession(context.Background(), " ws-1 ") {
		t.Fatalf("HasValidWebSession() = false, want true")
	}
	if stub.lastHasValidSessionID != "ws-1" {
		t.Fatalf("has valid session id = %q, want %q", stub.lastHasValidSessionID, "ws-1")
	}
}

type authGatewayStub struct {
	createUserErr                          error
	beginPasskeyRegistrationErr            error
	beginPasskeyRegistrationSessionID      string
	finishPasskeyRegistrationResult        string
	finishPasskeyLoginResult               string
	createUserResult                       string
	createWebSessionResult                 string
	createWebSessionErr                    error
	hasValidSessionResult                  bool
	revokeWebSessionErr                    error
	revokeCalled                           bool
	lastCreateUserEmail                    string
	lastBeginPasskeyRegistrationUserID     string
	lastFinishPasskeyLoginSessionID        string
	lastFinishPasskeyRegistrationSessionID string
	lastCreateWebSessionUserID             string
	lastHasValidSessionID                  string
	lastRevokeSessionID                    string
}

func (f *authGatewayStub) CreateUser(_ context.Context, email string) (string, error) {
	f.lastCreateUserEmail = email
	if f.createUserErr != nil {
		return "", f.createUserErr
	}
	if f.createUserResult != "" {
		return f.createUserResult, nil
	}
	return "user-1", nil
}

func (f *authGatewayStub) BeginPasskeyRegistration(_ context.Context, userID string) (PasskeyChallenge, error) {
	f.lastBeginPasskeyRegistrationUserID = userID
	if f.beginPasskeyRegistrationErr != nil {
		return PasskeyChallenge{}, f.beginPasskeyRegistrationErr
	}
	sessionID := f.beginPasskeyRegistrationSessionID
	if sessionID == "" {
		sessionID = "register-session"
	}
	return PasskeyChallenge{SessionID: sessionID, PublicKey: json.RawMessage(`{"publicKey":{}}`)}, nil
}

func (f *authGatewayStub) FinishPasskeyRegistration(_ context.Context, sessionID string, _ json.RawMessage) (string, error) {
	f.lastFinishPasskeyRegistrationSessionID = sessionID
	if f.finishPasskeyRegistrationResult != "" {
		return f.finishPasskeyRegistrationResult, nil
	}
	return "user-1", nil
}

func (f *authGatewayStub) BeginPasskeyLogin(context.Context) (PasskeyChallenge, error) {
	return PasskeyChallenge{SessionID: "login-session", PublicKey: json.RawMessage(`{"publicKey":{}}`)}, nil
}

func (f *authGatewayStub) FinishPasskeyLogin(_ context.Context, sessionID string, _ json.RawMessage) (string, error) {
	f.lastFinishPasskeyLoginSessionID = sessionID
	if f.finishPasskeyLoginResult != "" {
		return f.finishPasskeyLoginResult, nil
	}
	return "user-1", nil
}

func (f *authGatewayStub) CreateWebSession(_ context.Context, userID string) (string, error) {
	f.lastCreateWebSessionUserID = userID
	if f.createWebSessionErr != nil {
		return "", f.createWebSessionErr
	}
	if f.createWebSessionResult != "" {
		return f.createWebSessionResult, nil
	}
	return "ws-1", nil
}

func (f *authGatewayStub) HasValidWebSession(_ context.Context, sessionID string) bool {
	f.lastHasValidSessionID = sessionID
	return f.hasValidSessionResult
}

func (f *authGatewayStub) RevokeWebSession(_ context.Context, sessionID string) error {
	f.revokeCalled = true
	f.lastRevokeSessionID = sessionID
	return f.revokeWebSessionErr
}
