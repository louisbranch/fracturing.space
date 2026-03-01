package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewGRPCGatewayWithoutClientFailsClosed(t *testing.T) {
	t.Parallel()

	gateway := NewGRPCGateway(nil)
	_, err := gateway.BeginPasskeyLogin(context.Background())
	if err == nil {
		t.Fatalf("expected unavailable error when auth client is missing")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

func TestGRPCGatewayMapsProtoToDomainTypes(t *testing.T) {
	t.Parallel()

	client := &recordingAuthClient{
		createUserResp:        &authv1.CreateUserResponse{User: &authv1.User{Id: "user-1"}},
		beginPasskeyLoginResp: &authv1.BeginPasskeyLoginResponse{SessionId: "login-1", CredentialRequestOptionsJson: []byte(`{"publicKey":{}}`)},
	}
	gateway := newGRPCGateway(client)

	ctx := context.Background()
	userID, err := gateway.CreateUser(ctx, "user@example.com")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if userID != "user-1" {
		t.Fatalf("CreateUser() userID = %q, want %q", userID, "user-1")
	}
	if client.lastCreateUserReq == nil || client.lastCreateUserReq.GetEmail() != "user@example.com" {
		t.Fatalf("CreateUser request email not forwarded")
	}
	if client.lastCreateUserReq.GetLocale() != commonv1.Locale_LOCALE_EN_US {
		t.Fatalf("CreateUser locale = %v, want %v", client.lastCreateUserReq.GetLocale(), commonv1.Locale_LOCALE_EN_US)
	}

	challenge, err := gateway.BeginPasskeyLogin(ctx)
	if err != nil {
		t.Fatalf("BeginPasskeyLogin() error = %v", err)
	}
	if challenge.SessionID != "login-1" {
		t.Fatalf("SessionID = %q, want %q", challenge.SessionID, "login-1")
	}
	if got := string(challenge.PublicKey); got != `{"publicKey":{}}` {
		t.Fatalf("PublicKey = %s, want %s", got, `{"publicKey":{}}`)
	}
}

func TestGRPCGatewayRejectsMissingProtoFields(t *testing.T) {
	t.Parallel()

	t.Run("create user missing id", func(t *testing.T) {
		t.Parallel()
		g := newGRPCGateway(&recordingAuthClient{createUserResp: &authv1.CreateUserResponse{}})
		_, err := g.CreateUser(context.Background(), "user@example.com")
		if err == nil {
			t.Fatalf("expected error when auth create user response has no user id")
		}
	})

	t.Run("begin login missing session", func(t *testing.T) {
		t.Parallel()
		g := newGRPCGateway(&recordingAuthClient{beginPasskeyLoginResp: &authv1.BeginPasskeyLoginResponse{}})
		_, err := g.BeginPasskeyLogin(context.Background())
		if err == nil {
			t.Fatalf("expected error when begin login response has no session id")
		}
	})
}

func TestGRPCGatewayMapsTransportErrors(t *testing.T) {
	t.Parallel()

	g := newGRPCGateway(&recordingAuthClient{createUserErr: status.Error(codes.AlreadyExists, "user exists")})
	_, err := g.CreateUser(context.Background(), "existing@example.com")
	if err == nil {
		t.Fatalf("expected mapped transport error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
	}
}

type recordingAuthClient struct {
	createUserResp *authv1.CreateUserResponse
	createUserErr  error

	beginPasskeyLoginResp *authv1.BeginPasskeyLoginResponse
	beginPasskeyLoginErr  error

	lastCreateUserReq *authv1.CreateUserRequest
}

func (f *recordingAuthClient) CreateUser(_ context.Context, req *authv1.CreateUserRequest, _ ...grpc.CallOption) (*authv1.CreateUserResponse, error) {
	f.lastCreateUserReq = req
	if f.createUserErr != nil {
		return nil, f.createUserErr
	}
	if f.createUserResp != nil {
		return f.createUserResp, nil
	}
	return &authv1.CreateUserResponse{User: &authv1.User{Id: "user-1"}}, nil
}

func (f *recordingAuthClient) BeginPasskeyRegistration(context.Context, *authv1.BeginPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.BeginPasskeyRegistrationResponse, error) {
	return &authv1.BeginPasskeyRegistrationResponse{SessionId: "register-1", CredentialCreationOptionsJson: json.RawMessage(`{"publicKey":{}}`)}, nil
}

func (f *recordingAuthClient) FinishPasskeyRegistration(context.Context, *authv1.FinishPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.FinishPasskeyRegistrationResponse, error) {
	return &authv1.FinishPasskeyRegistrationResponse{User: &authv1.User{Id: "user-1"}}, nil
}

func (f *recordingAuthClient) BeginPasskeyLogin(context.Context, *authv1.BeginPasskeyLoginRequest, ...grpc.CallOption) (*authv1.BeginPasskeyLoginResponse, error) {
	if f.beginPasskeyLoginErr != nil {
		return nil, f.beginPasskeyLoginErr
	}
	if f.beginPasskeyLoginResp != nil {
		return f.beginPasskeyLoginResp, nil
	}
	return &authv1.BeginPasskeyLoginResponse{SessionId: "login-1", CredentialRequestOptionsJson: json.RawMessage(`{"publicKey":{}}`)}, nil
}

func (f *recordingAuthClient) FinishPasskeyLogin(context.Context, *authv1.FinishPasskeyLoginRequest, ...grpc.CallOption) (*authv1.FinishPasskeyLoginResponse, error) {
	return &authv1.FinishPasskeyLoginResponse{User: &authv1.User{Id: "user-1"}}, nil
}

func (f *recordingAuthClient) CreateWebSession(context.Context, *authv1.CreateWebSessionRequest, ...grpc.CallOption) (*authv1.CreateWebSessionResponse, error) {
	return &authv1.CreateWebSessionResponse{Session: &authv1.WebSession{Id: "ws-1", UserId: "user-1"}}, nil
}

func (f *recordingAuthClient) GetWebSession(context.Context, *authv1.GetWebSessionRequest, ...grpc.CallOption) (*authv1.GetWebSessionResponse, error) {
	return &authv1.GetWebSessionResponse{Session: &authv1.WebSession{Id: "ws-1", UserId: "user-1"}}, nil
}

func (f *recordingAuthClient) RevokeWebSession(context.Context, *authv1.RevokeWebSessionRequest, ...grpc.CallOption) (*authv1.RevokeWebSessionResponse, error) {
	return &authv1.RevokeWebSessionResponse{}, nil
}
