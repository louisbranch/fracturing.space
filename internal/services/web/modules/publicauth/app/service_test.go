package app

import (
	"context"
	"encoding/json"
	"testing"
)

func TestPasskeyLoginStartRequiresUsername(t *testing.T) {
	svc := NewService(&authGatewayStub{})
	_, err := svc.PasskeyLoginStart(context.Background(), "  ")
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestPasskeyRegisterStartAndFinish(t *testing.T) {
	stub := &authGatewayStub{
		beginRegistrationResp: PasskeyChallenge{SessionID: "reg-1", PublicKey: json.RawMessage(`{"publicKey":{}}`)},
		finishRegistrationResp: PasskeyFinish{
			SessionID:    "web-1",
			UserID:       "user-1",
			RecoveryCode: "ABCD-EFGH",
		},
	}
	svc := NewService(stub)

	start, err := svc.PasskeyRegisterStart(context.Background(), "louis")
	if err != nil {
		t.Fatalf("PasskeyRegisterStart() error = %v", err)
	}
	if start.SessionID != "reg-1" {
		t.Fatalf("SessionID = %q, want %q", start.SessionID, "reg-1")
	}

	finish, err := svc.PasskeyRegisterFinish(context.Background(), "reg-1", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("PasskeyRegisterFinish() error = %v", err)
	}
	if finish.SessionID != "web-1" || finish.UserID != "user-1" || finish.RecoveryCode != "ABCD-EFGH" {
		t.Fatalf("finish = %+v", finish)
	}
}

func TestPasskeyLoginFinishCreatesWebSession(t *testing.T) {
	stub := &authGatewayStub{
		finishLoginUserID: "user-1",
		webSessionID:      "web-1",
	}
	svc := NewService(stub)

	finish, err := svc.PasskeyLoginFinish(context.Background(), "login-1", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("PasskeyLoginFinish() error = %v", err)
	}
	if finish.SessionID != "web-1" || finish.UserID != "user-1" {
		t.Fatalf("finish = %+v", finish)
	}
}

type authGatewayStub struct {
	beginRegistrationResp  PasskeyChallenge
	finishRegistrationResp PasskeyFinish
	beginLoginResp         PasskeyChallenge
	finishLoginUserID      string
	webSessionID           string
}

func (f *authGatewayStub) BeginAccountRegistration(context.Context, string) (PasskeyChallenge, error) {
	return f.beginRegistrationResp, nil
}

func (f *authGatewayStub) FinishAccountRegistration(context.Context, string, json.RawMessage) (PasskeyFinish, error) {
	return f.finishRegistrationResp, nil
}

func (f *authGatewayStub) BeginPasskeyLogin(context.Context, string) (PasskeyChallenge, error) {
	return f.beginLoginResp, nil
}

func (f *authGatewayStub) FinishPasskeyLogin(context.Context, string, json.RawMessage) (string, error) {
	return f.finishLoginUserID, nil
}

func (f *authGatewayStub) CreateWebSession(context.Context, string) (string, error) {
	return f.webSessionID, nil
}

func (*authGatewayStub) HasValidWebSession(context.Context, string) bool {
	return false
}

func (*authGatewayStub) RevokeWebSession(context.Context, string) error {
	return nil
}
