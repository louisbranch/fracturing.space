package app

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

func TestServiceHealthBodyIsStable(t *testing.T) {
	t.Parallel()

	svc := NewService(gatewayContractStub{})
	if got := svc.HealthBody(); got != "ok" {
		t.Fatalf("HealthBody() = %q, want %q", got, "ok")
	}
}

func TestPasskeyRegisterFinishValidatesAndReturnsUser(t *testing.T) {
	t.Parallel()

	svc := NewService(gatewayContractStub{})
	if _, err := svc.PasskeyRegisterFinish(context.Background(), "", json.RawMessage(`{"id":"cred-1"}`)); err == nil {
		t.Fatalf("expected validation error for empty session id")
	}
	if _, err := svc.PasskeyRegisterFinish(context.Background(), "session-1", nil); err == nil {
		t.Fatalf("expected validation error for empty credential")
	}

	svc = NewService(gatewayContractStub{finishRegisterUserID: "user-7"})
	finished, err := svc.PasskeyRegisterFinish(context.Background(), "session-1", json.RawMessage(`{"id":"cred-1"}`))
	if err != nil {
		t.Fatalf("PasskeyRegisterFinish() error = %v", err)
	}
	if finished.UserID != "user-7" {
		t.Fatalf("UserID = %q, want %q", finished.UserID, "user-7")
	}
	if finished.SessionID != "" {
		t.Fatalf("SessionID = %q, want empty", finished.SessionID)
	}
}

func TestUnavailableGatewayFailsClosedAcrossOperations(t *testing.T) {
	t.Parallel()

	gw := NewUnavailableGateway()
	if _, err := gw.CreateUser(context.Background(), "user@example.com"); err == nil {
		t.Fatalf("expected CreateUser unavailable error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("CreateUser HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}
	if _, err := gw.BeginPasskeyRegistration(context.Background(), "user-1"); err == nil {
		t.Fatalf("expected BeginPasskeyRegistration unavailable error")
	}
	if _, err := gw.FinishPasskeyRegistration(context.Background(), "session-1", json.RawMessage(`{"id":"cred-1"}`)); err == nil {
		t.Fatalf("expected FinishPasskeyRegistration unavailable error")
	}
	if _, err := gw.FinishPasskeyLogin(context.Background(), "session-1", json.RawMessage(`{"id":"cred-1"}`)); err == nil {
		t.Fatalf("expected FinishPasskeyLogin unavailable error")
	}
	if _, err := gw.CreateWebSession(context.Background(), "user-1"); err == nil {
		t.Fatalf("expected CreateWebSession unavailable error")
	}
	if err := gw.RevokeWebSession(context.Background(), "session-1"); err == nil {
		t.Fatalf("expected RevokeWebSession unavailable error")
	}
}

type gatewayContractStub struct {
	finishRegisterUserID string
	hasValidSession      bool
}

func (s gatewayContractStub) CreateUser(context.Context, string) (string, error) {
	return "user-1", nil
}

func (s gatewayContractStub) BeginPasskeyRegistration(context.Context, string) (PasskeyChallenge, error) {
	return PasskeyChallenge{SessionID: "register-session", PublicKey: json.RawMessage(`{"publicKey":{}}`)}, nil
}

func (s gatewayContractStub) FinishPasskeyRegistration(context.Context, string, json.RawMessage) (string, error) {
	if s.finishRegisterUserID != "" {
		return s.finishRegisterUserID, nil
	}
	return "user-1", nil
}

func (s gatewayContractStub) BeginPasskeyLogin(context.Context) (PasskeyChallenge, error) {
	return PasskeyChallenge{SessionID: "login-session", PublicKey: json.RawMessage(`{"publicKey":{}}`)}, nil
}

func (s gatewayContractStub) FinishPasskeyLogin(context.Context, string, json.RawMessage) (string, error) {
	return "user-1", nil
}

func (s gatewayContractStub) CreateWebSession(context.Context, string) (string, error) {
	return "ws-1", nil
}

func (s gatewayContractStub) HasValidWebSession(context.Context, string) bool {
	return s.hasValidSession
}

func (s gatewayContractStub) RevokeWebSession(context.Context, string) error {
	return nil
}
