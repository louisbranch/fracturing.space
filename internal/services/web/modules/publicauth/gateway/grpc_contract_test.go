package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGRPCGatewayRegistrationAndSessionMethods(t *testing.T) {
	t.Parallel()

	client := &contractAuthClient{
		beginPasskeyRegistrationResp:  &authv1.BeginPasskeyRegistrationResponse{SessionId: "register-1", CredentialCreationOptionsJson: json.RawMessage(`{"publicKey":{"rpId":"example.com"}}`)},
		finishPasskeyRegistrationResp: &authv1.FinishPasskeyRegistrationResponse{User: &authv1.User{Id: "user-1"}},
		finishPasskeyLoginResp:        &authv1.FinishPasskeyLoginResponse{User: &authv1.User{Id: "user-2"}},
		createWebSessionResp:          &authv1.CreateWebSessionResponse{Session: &authv1.WebSession{Id: "ws-7"}},
		getWebSessionResp:             &authv1.GetWebSessionResponse{Session: &authv1.WebSession{Id: "ws-7"}},
	}
	gateway := newGRPCGateway(client)

	challenge, err := gateway.BeginPasskeyRegistration(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("BeginPasskeyRegistration() error = %v", err)
	}
	if challenge.SessionID != "register-1" {
		t.Fatalf("SessionID = %q, want %q", challenge.SessionID, "register-1")
	}
	if client.lastBeginPasskeyRegistrationReq.GetUserId() != "user-1" {
		t.Fatalf("UserId = %q, want %q", client.lastBeginPasskeyRegistrationReq.GetUserId(), "user-1")
	}

	registeredUserID, err := gateway.FinishPasskeyRegistration(context.Background(), "register-1", json.RawMessage(`{"id":"cred-r"}`))
	if err != nil {
		t.Fatalf("FinishPasskeyRegistration() error = %v", err)
	}
	if registeredUserID != "user-1" {
		t.Fatalf("user id = %q, want %q", registeredUserID, "user-1")
	}

	loggedInUserID, err := gateway.FinishPasskeyLogin(context.Background(), "login-1", json.RawMessage(`{"id":"cred-l"}`))
	if err != nil {
		t.Fatalf("FinishPasskeyLogin() error = %v", err)
	}
	if loggedInUserID != "user-2" {
		t.Fatalf("user id = %q, want %q", loggedInUserID, "user-2")
	}

	webSessionID, err := gateway.CreateWebSession(context.Background(), "user-2")
	if err != nil {
		t.Fatalf("CreateWebSession() error = %v", err)
	}
	if webSessionID != "ws-7" {
		t.Fatalf("web session id = %q, want %q", webSessionID, "ws-7")
	}
	if !gateway.HasValidWebSession(context.Background(), "ws-7") {
		t.Fatalf("expected HasValidWebSession() true")
	}
}

func TestGRPCGatewaySessionLifecycleErrorMapping(t *testing.T) {
	t.Parallel()

	t.Run("missing proto fields are rejected", func(t *testing.T) {
		t.Parallel()

		gateway := newGRPCGateway(&contractAuthClient{
			finishPasskeyRegistrationResp: &authv1.FinishPasskeyRegistrationResponse{},
			finishPasskeyLoginResp:        &authv1.FinishPasskeyLoginResponse{},
			createWebSessionResp:          &authv1.CreateWebSessionResponse{},
		})

		if _, err := gateway.FinishPasskeyRegistration(context.Background(), "session-1", json.RawMessage(`{"id":"cred"}`)); err == nil {
			t.Fatalf("expected missing user id error")
		}
		if _, err := gateway.FinishPasskeyLogin(context.Background(), "session-1", json.RawMessage(`{"id":"cred"}`)); err == nil {
			t.Fatalf("expected missing user id error")
		}
		if _, err := gateway.CreateWebSession(context.Background(), "user-1"); err == nil {
			t.Fatalf("expected missing web session id error")
		}
	})

	t.Run("transport errors map to typed errors", func(t *testing.T) {
		t.Parallel()

		gateway := newGRPCGateway(&contractAuthClient{
			finishPasskeyRegistrationErr: status.Error(codes.InvalidArgument, "bad credential"),
			finishPasskeyLoginErr:        status.Error(codes.InvalidArgument, "bad credential"),
			createWebSessionErr:          errors.New("backend boom"),
			revokeWebSessionErr:          status.Error(codes.Unavailable, "down"),
		})

		if _, err := gateway.FinishPasskeyRegistration(context.Background(), "session-1", json.RawMessage(`{"id":"cred"}`)); err == nil {
			t.Fatalf("expected registration mapping error")
		} else if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
			t.Fatalf("FinishPasskeyRegistration HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
		}
		if _, err := gateway.FinishPasskeyLogin(context.Background(), "session-1", json.RawMessage(`{"id":"cred"}`)); err == nil {
			t.Fatalf("expected login mapping error")
		} else if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
			t.Fatalf("FinishPasskeyLogin HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
		}
		if _, err := gateway.CreateWebSession(context.Background(), "user-1"); err == nil {
			t.Fatalf("expected create session mapping error")
		} else if got := apperrors.HTTPStatus(err); got != http.StatusInternalServerError {
			t.Fatalf("CreateWebSession HTTPStatus(err) = %d, want %d", got, http.StatusInternalServerError)
		}
		if err := gateway.RevokeWebSession(context.Background(), "ws-1"); err == nil {
			t.Fatalf("expected revoke mapping error")
		} else if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
			t.Fatalf("RevokeWebSession HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
		}
	})

	t.Run("invalid or missing web session returns false", func(t *testing.T) {
		t.Parallel()

		gateway := newGRPCGateway(&contractAuthClient{getWebSessionErr: errors.New("missing")})
		if gateway.HasValidWebSession(context.Background(), "ws-1") {
			t.Fatalf("expected invalid session when transport fails")
		}

		gateway = newGRPCGateway(&contractAuthClient{getWebSessionResp: &authv1.GetWebSessionResponse{Session: &authv1.WebSession{Id: "   "}}})
		if gateway.HasValidWebSession(context.Background(), "ws-1") {
			t.Fatalf("expected invalid session when id is empty")
		}
	})
}

type contractAuthClient struct {
	beginPasskeyRegistrationResp    *authv1.BeginPasskeyRegistrationResponse
	beginPasskeyRegistrationErr     error
	lastBeginPasskeyRegistrationReq *authv1.BeginPasskeyRegistrationRequest

	finishPasskeyRegistrationResp *authv1.FinishPasskeyRegistrationResponse
	finishPasskeyRegistrationErr  error

	finishPasskeyLoginResp *authv1.FinishPasskeyLoginResponse
	finishPasskeyLoginErr  error

	createWebSessionResp *authv1.CreateWebSessionResponse
	createWebSessionErr  error

	getWebSessionResp *authv1.GetWebSessionResponse
	getWebSessionErr  error

	revokeWebSessionErr error
}

func (c *contractAuthClient) CreateUser(context.Context, *authv1.CreateUserRequest, ...grpc.CallOption) (*authv1.CreateUserResponse, error) {
	return &authv1.CreateUserResponse{User: &authv1.User{Id: "user-1"}}, nil
}

func (c *contractAuthClient) BeginPasskeyRegistration(_ context.Context, req *authv1.BeginPasskeyRegistrationRequest, _ ...grpc.CallOption) (*authv1.BeginPasskeyRegistrationResponse, error) {
	c.lastBeginPasskeyRegistrationReq = req
	if c.beginPasskeyRegistrationErr != nil {
		return nil, c.beginPasskeyRegistrationErr
	}
	if c.beginPasskeyRegistrationResp != nil {
		return c.beginPasskeyRegistrationResp, nil
	}
	return &authv1.BeginPasskeyRegistrationResponse{SessionId: "register-1", CredentialCreationOptionsJson: json.RawMessage(`{"publicKey":{}}`)}, nil
}

func (c *contractAuthClient) FinishPasskeyRegistration(context.Context, *authv1.FinishPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.FinishPasskeyRegistrationResponse, error) {
	if c.finishPasskeyRegistrationErr != nil {
		return nil, c.finishPasskeyRegistrationErr
	}
	if c.finishPasskeyRegistrationResp != nil {
		return c.finishPasskeyRegistrationResp, nil
	}
	return &authv1.FinishPasskeyRegistrationResponse{User: &authv1.User{Id: "user-1"}}, nil
}

func (c *contractAuthClient) BeginPasskeyLogin(context.Context, *authv1.BeginPasskeyLoginRequest, ...grpc.CallOption) (*authv1.BeginPasskeyLoginResponse, error) {
	return &authv1.BeginPasskeyLoginResponse{SessionId: "login-1", CredentialRequestOptionsJson: json.RawMessage(`{"publicKey":{}}`)}, nil
}

func (c *contractAuthClient) FinishPasskeyLogin(context.Context, *authv1.FinishPasskeyLoginRequest, ...grpc.CallOption) (*authv1.FinishPasskeyLoginResponse, error) {
	if c.finishPasskeyLoginErr != nil {
		return nil, c.finishPasskeyLoginErr
	}
	if c.finishPasskeyLoginResp != nil {
		return c.finishPasskeyLoginResp, nil
	}
	return &authv1.FinishPasskeyLoginResponse{User: &authv1.User{Id: "user-1"}}, nil
}

func (c *contractAuthClient) CreateWebSession(context.Context, *authv1.CreateWebSessionRequest, ...grpc.CallOption) (*authv1.CreateWebSessionResponse, error) {
	if c.createWebSessionErr != nil {
		return nil, c.createWebSessionErr
	}
	if c.createWebSessionResp != nil {
		return c.createWebSessionResp, nil
	}
	return &authv1.CreateWebSessionResponse{Session: &authv1.WebSession{Id: "ws-1"}}, nil
}

func (c *contractAuthClient) GetWebSession(context.Context, *authv1.GetWebSessionRequest, ...grpc.CallOption) (*authv1.GetWebSessionResponse, error) {
	if c.getWebSessionErr != nil {
		return nil, c.getWebSessionErr
	}
	if c.getWebSessionResp != nil {
		return c.getWebSessionResp, nil
	}
	return &authv1.GetWebSessionResponse{Session: &authv1.WebSession{Id: "ws-1"}}, nil
}

func (c *contractAuthClient) RevokeWebSession(context.Context, *authv1.RevokeWebSessionRequest, ...grpc.CallOption) (*authv1.RevokeWebSessionResponse, error) {
	if c.revokeWebSessionErr != nil {
		return nil, c.revokeWebSessionErr
	}
	return &authv1.RevokeWebSessionResponse{}, nil
}
