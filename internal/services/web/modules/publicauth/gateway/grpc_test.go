package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc"
)

func TestNewGRPCGatewayFailsClosedWhenClientMissing(t *testing.T) {
	gateway := NewGRPCGateway(nil)
	_, err := gateway.BeginPasskeyLogin(context.Background(), "louis")
	if err == nil {
		t.Fatal("expected unavailable error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

func TestBeginAccountRegistrationForwardsUsername(t *testing.T) {
	client := &recordingAuthClient{
		beginRegistrationResp: &authv1.BeginAccountRegistrationResponse{SessionId: "reg-1", CredentialCreationOptionsJson: []byte(`{"publicKey":{}}`)},
	}
	gateway := newGRPCGateway(client)

	challenge, err := gateway.BeginAccountRegistration(context.Background(), "louis")
	if err != nil {
		t.Fatalf("BeginAccountRegistration() error = %v", err)
	}
	if client.lastBeginRegistration.GetUsername() != "louis" {
		t.Fatalf("username = %q, want %q", client.lastBeginRegistration.GetUsername(), "louis")
	}
	if challenge.SessionID != "reg-1" {
		t.Fatalf("SessionID = %q, want %q", challenge.SessionID, "reg-1")
	}
}

func TestFinishAccountRegistrationMapsSessionAndRecoveryCode(t *testing.T) {
	client := &recordingAuthClient{
		finishRegistrationResp: &authv1.FinishAccountRegistrationResponse{
			User:         &authv1.User{Id: "user-1"},
			Session:      &authv1.WebSession{Id: "web-1"},
			RecoveryCode: "ABCD-EFGH",
		},
	}
	gateway := newGRPCGateway(client)

	finish, err := gateway.FinishAccountRegistration(context.Background(), "reg-1", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("FinishAccountRegistration() error = %v", err)
	}
	if finish.UserID != "user-1" || finish.SessionID != "web-1" || finish.RecoveryCode != "ABCD-EFGH" {
		t.Fatalf("finish = %+v", finish)
	}
}

type recordingAuthClient struct {
	beginRegistrationResp  *authv1.BeginAccountRegistrationResponse
	finishRegistrationResp *authv1.FinishAccountRegistrationResponse
	lastBeginRegistration  *authv1.BeginAccountRegistrationRequest
}

func (f *recordingAuthClient) BeginAccountRegistration(_ context.Context, req *authv1.BeginAccountRegistrationRequest, _ ...grpc.CallOption) (*authv1.BeginAccountRegistrationResponse, error) {
	f.lastBeginRegistration = req
	if f.beginRegistrationResp != nil {
		return f.beginRegistrationResp, nil
	}
	return &authv1.BeginAccountRegistrationResponse{}, nil
}

func (f *recordingAuthClient) FinishAccountRegistration(context.Context, *authv1.FinishAccountRegistrationRequest, ...grpc.CallOption) (*authv1.FinishAccountRegistrationResponse, error) {
	if f.finishRegistrationResp != nil {
		return f.finishRegistrationResp, nil
	}
	return &authv1.FinishAccountRegistrationResponse{}, nil
}

func (*recordingAuthClient) BeginPasskeyLogin(context.Context, *authv1.BeginPasskeyLoginRequest, ...grpc.CallOption) (*authv1.BeginPasskeyLoginResponse, error) {
	return &authv1.BeginPasskeyLoginResponse{}, nil
}

func (*recordingAuthClient) FinishPasskeyLogin(context.Context, *authv1.FinishPasskeyLoginRequest, ...grpc.CallOption) (*authv1.FinishPasskeyLoginResponse, error) {
	return &authv1.FinishPasskeyLoginResponse{}, nil
}

func (*recordingAuthClient) CreateWebSession(context.Context, *authv1.CreateWebSessionRequest, ...grpc.CallOption) (*authv1.CreateWebSessionResponse, error) {
	return &authv1.CreateWebSessionResponse{}, nil
}

func (*recordingAuthClient) GetWebSession(context.Context, *authv1.GetWebSessionRequest, ...grpc.CallOption) (*authv1.GetWebSessionResponse, error) {
	return &authv1.GetWebSessionResponse{}, nil
}

func (*recordingAuthClient) RevokeWebSession(context.Context, *authv1.RevokeWebSessionRequest, ...grpc.CallOption) (*authv1.RevokeWebSessionResponse, error) {
	return &authv1.RevokeWebSessionResponse{}, nil
}
