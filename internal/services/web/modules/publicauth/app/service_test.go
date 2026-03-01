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
	err := svc.RevokeWebSession(context.Background(), "session-1")
	if err == nil {
		t.Fatalf("expected revoke failure")
	}
}

type authGatewayStub struct {
	createUserErr               error
	beginPasskeyRegistrationErr error
	finishPasskeyLoginResult    string
	createWebSessionResult      string
	createWebSessionErr         error
	revokeWebSessionErr         error
	revokeCalled                bool
}

func (f *authGatewayStub) CreateUser(context.Context, string) (string, error) {
	if f.createUserErr != nil {
		return "", f.createUserErr
	}
	return "user-1", nil
}

func (f *authGatewayStub) BeginPasskeyRegistration(context.Context, string) (PasskeyChallenge, error) {
	if f.beginPasskeyRegistrationErr != nil {
		return PasskeyChallenge{}, f.beginPasskeyRegistrationErr
	}
	return PasskeyChallenge{SessionID: "register-session", PublicKey: json.RawMessage(`{"publicKey":{}}`)}, nil
}

func (f *authGatewayStub) FinishPasskeyRegistration(context.Context, string, json.RawMessage) (string, error) {
	return "user-1", nil
}

func (f *authGatewayStub) BeginPasskeyLogin(context.Context) (PasskeyChallenge, error) {
	return PasskeyChallenge{SessionID: "login-session", PublicKey: json.RawMessage(`{"publicKey":{}}`)}, nil
}

func (f *authGatewayStub) FinishPasskeyLogin(context.Context, string, json.RawMessage) (string, error) {
	if f.finishPasskeyLoginResult != "" {
		return f.finishPasskeyLoginResult, nil
	}
	return "user-1", nil
}

func (f *authGatewayStub) CreateWebSession(context.Context, string) (string, error) {
	if f.createWebSessionErr != nil {
		return "", f.createWebSessionErr
	}
	if f.createWebSessionResult != "" {
		return f.createWebSessionResult, nil
	}
	return "ws-1", nil
}

func (f *authGatewayStub) HasValidWebSession(context.Context, string) bool {
	return false
}

func (f *authGatewayStub) RevokeWebSession(context.Context, string) error {
	f.revokeCalled = true
	return f.revokeWebSessionErr
}
