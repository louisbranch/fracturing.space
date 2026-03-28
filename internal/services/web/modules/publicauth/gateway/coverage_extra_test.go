package gateway

import (
	"context"
	"net/http"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc"
)

func TestGRPCGatewayNilClientFailsClosedAcrossSurface(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	gateway := NewGRPCGateway(nil)

	checkUnavailable := func(t *testing.T, err error) {
		t.Helper()
		if err == nil {
			t.Fatal("expected unavailable error")
		}
		if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
			t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
		}
	}

	t.Run("registration and login", func(t *testing.T) {
		_, err := gateway.BeginAccountRegistration(ctx, "louis")
		checkUnavailable(t, err)
		_, err = gateway.CheckUsernameAvailability(ctx, "louis")
		checkUnavailable(t, err)
		_, err = gateway.FinishAccountRegistration(ctx, "sess-1", nil)
		checkUnavailable(t, err)
		_, err = gateway.AcknowledgeAccountRegistration(ctx, "sess-1", "pending-1")
		checkUnavailable(t, err)
		_, err = gateway.BeginPasskeyLogin(ctx, "louis")
		checkUnavailable(t, err)
		_, err = gateway.FinishPasskeyLogin(ctx, "sess-1", nil, "pending-1")
		checkUnavailable(t, err)
	})

	t.Run("recovery and session", func(t *testing.T) {
		_, err := gateway.BeginAccountRecovery(ctx, "louis", "code")
		checkUnavailable(t, err)
		_, err = gateway.BeginRecoveryPasskeyRegistration(ctx, "rec-1")
		checkUnavailable(t, err)
		_, err = gateway.FinishRecoveryPasskeyRegistration(ctx, "rec-1", "sess-1", nil, "pending-1")
		checkUnavailable(t, err)
		_, err = gateway.CreateWebSession(ctx, "user-1")
		checkUnavailable(t, err)
		checkUnavailable(t, gateway.RevokeWebSession(ctx, "web-1"))
	})
}

func TestGRPCGatewayAdditionalMappingBranches(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("username availability states", func(t *testing.T) {
		client := &gatewayAuthClientStub{
			usernameAvailabilityResp: &authv1.CheckUsernameAvailabilityResponse{
				CanonicalUsername: "louis",
				State:             authv1.UsernameAvailabilityState_USERNAME_AVAILABILITY_STATE_UNAVAILABLE,
			},
		}
		gateway := newGRPCGateway(client)

		availability, err := gateway.CheckUsernameAvailability(ctx, "Louis")
		if err != nil {
			t.Fatalf("CheckUsernameAvailability() error = %v", err)
		}
		if client.lastUsernameAvailability.GetUsername() != "Louis" {
			t.Fatalf("username = %q, want %q", client.lastUsernameAvailability.GetUsername(), "Louis")
		}
		if availability.State != "unavailable" || availability.CanonicalUsername != "louis" {
			t.Fatalf("availability = %+v", availability)
		}

		client.usernameAvailabilityResp.State = authv1.UsernameAvailabilityState_USERNAME_AVAILABILITY_STATE_UNSPECIFIED
		availability, err = gateway.CheckUsernameAvailability(ctx, "Louis")
		if err != nil {
			t.Fatalf("CheckUsernameAvailability(unspecified) error = %v", err)
		}
		if availability.State != "invalid" {
			t.Fatalf("availability.State = %q, want invalid", availability.State)
		}
	})

	t.Run("passkey login branches", func(t *testing.T) {
		client := &gatewayAuthClientStub{
			beginPasskeyLoginResp: &authv1.BeginPasskeyLoginResponse{
				SessionId:                    " login-1 ",
				CredentialRequestOptionsJson: []byte(`{"publicKey":{}}`),
			},
			finishPasskeyLoginResp: &authv1.FinishPasskeyLoginResponse{
				User: &authv1.User{Id: " user-1 "},
			},
		}
		gateway := newGRPCGateway(client)

		challenge, err := gateway.BeginPasskeyLogin(ctx, "louis")
		if err != nil {
			t.Fatalf("BeginPasskeyLogin() error = %v", err)
		}
		if challenge.SessionID != "login-1" {
			t.Fatalf("challenge.SessionID = %q, want %q", challenge.SessionID, "login-1")
		}

		userID, err := gateway.FinishPasskeyLogin(ctx, "sess-1", []byte(`{}`), " pending-1 ")
		if err != nil {
			t.Fatalf("FinishPasskeyLogin() error = %v", err)
		}
		if client.lastFinishPasskeyLogin.GetPendingId() != "pending-1" {
			t.Fatalf("pending id = %q, want %q", client.lastFinishPasskeyLogin.GetPendingId(), "pending-1")
		}
		if userID != "user-1" {
			t.Fatalf("userID = %q, want %q", userID, "user-1")
		}

		client.beginPasskeyLoginResp = &authv1.BeginPasskeyLoginResponse{}
		_, err = gateway.BeginPasskeyLogin(ctx, "louis")
		if err == nil || apperrors.HTTPStatus(err) != http.StatusInternalServerError {
			t.Fatalf("BeginPasskeyLogin(missing session) err = %v", err)
		}

		client.finishPasskeyLoginResp = &authv1.FinishPasskeyLoginResponse{}
		_, err = gateway.FinishPasskeyLogin(ctx, "sess-1", []byte(`{}`), "")
		if err == nil || apperrors.HTTPStatus(err) != http.StatusInternalServerError {
			t.Fatalf("FinishPasskeyLogin(missing user) err = %v", err)
		}
	})

	t.Run("recovery branches", func(t *testing.T) {
		client := &gatewayAuthClientStub{
			beginAccountRecoveryResp: &authv1.BeginAccountRecoveryResponse{RecoverySessionId: " recovery-1 "},
			beginRecoveryPasskeyResp: &authv1.BeginPasskeyRegistrationResponse{
				SessionId:                     " passkey-1 ",
				CredentialCreationOptionsJson: []byte(`{"publicKey":{}}`),
			},
			finishRecoveryPasskeyResp: &authv1.FinishAccountRegistrationResponse{
				User:         &authv1.User{Id: " user-1 "},
				Session:      &authv1.WebSession{Id: " web-1 "},
				RecoveryCode: " CODE-1 ",
			},
		}
		gateway := newGRPCGateway(client)

		recoverySessionID, err := gateway.BeginAccountRecovery(ctx, "louis", "code")
		if err != nil {
			t.Fatalf("BeginAccountRecovery() error = %v", err)
		}
		if recoverySessionID != "recovery-1" {
			t.Fatalf("recoverySessionID = %q, want %q", recoverySessionID, "recovery-1")
		}

		challenge, err := gateway.BeginRecoveryPasskeyRegistration(ctx, "recovery-1")
		if err != nil {
			t.Fatalf("BeginRecoveryPasskeyRegistration() error = %v", err)
		}
		if challenge.SessionID != "passkey-1" {
			t.Fatalf("challenge.SessionID = %q, want %q", challenge.SessionID, "passkey-1")
		}

		finish, err := gateway.FinishRecoveryPasskeyRegistration(ctx, "recovery-1", "sess-1", []byte(`{}`), " pending-1 ")
		if err != nil {
			t.Fatalf("FinishRecoveryPasskeyRegistration() error = %v", err)
		}
		if client.lastFinishRecoveryPasskey.GetPendingId() != "pending-1" {
			t.Fatalf("pending id = %q, want %q", client.lastFinishRecoveryPasskey.GetPendingId(), "pending-1")
		}
		if finish.UserID != "user-1" || finish.SessionID != "web-1" || finish.RecoveryCode != "CODE-1" {
			t.Fatalf("finish = %+v", finish)
		}

		client.beginAccountRecoveryResp = &authv1.BeginAccountRecoveryResponse{}
		_, err = gateway.BeginAccountRecovery(ctx, "louis", "code")
		if err == nil || apperrors.HTTPStatus(err) != http.StatusInternalServerError {
			t.Fatalf("BeginAccountRecovery(missing session) err = %v", err)
		}

		client.beginRecoveryPasskeyResp = &authv1.BeginPasskeyRegistrationResponse{}
		_, err = gateway.BeginRecoveryPasskeyRegistration(ctx, "recovery-1")
		if err == nil || apperrors.HTTPStatus(err) != http.StatusInternalServerError {
			t.Fatalf("BeginRecoveryPasskeyRegistration(missing session) err = %v", err)
		}

		client.finishRecoveryPasskeyResp = &authv1.FinishAccountRegistrationResponse{Session: &authv1.WebSession{Id: "web-1"}}
		_, err = gateway.FinishRecoveryPasskeyRegistration(ctx, "recovery-1", "sess-1", []byte(`{}`), "")
		if err == nil || apperrors.HTTPStatus(err) != http.StatusInternalServerError {
			t.Fatalf("FinishRecoveryPasskeyRegistration(missing user) err = %v", err)
		}

		client.finishRecoveryPasskeyResp = &authv1.FinishAccountRegistrationResponse{User: &authv1.User{Id: "user-1"}}
		_, err = gateway.FinishRecoveryPasskeyRegistration(ctx, "recovery-1", "sess-1", []byte(`{}`), "")
		if err == nil || apperrors.HTTPStatus(err) != http.StatusInternalServerError {
			t.Fatalf("FinishRecoveryPasskeyRegistration(missing web session) err = %v", err)
		}
	})

	t.Run("web session branches", func(t *testing.T) {
		client := &gatewayAuthClientStub{
			createWebSessionResp: &authv1.CreateWebSessionResponse{
				Session: &authv1.WebSession{Id: " web-1 "},
			},
		}
		gateway := newGRPCGateway(client)

		sessionID, err := gateway.CreateWebSession(ctx, "user-1")
		if err != nil {
			t.Fatalf("CreateWebSession() error = %v", err)
		}
		if client.lastCreateWebSession.GetUserId() != "user-1" {
			t.Fatalf("user id = %q, want %q", client.lastCreateWebSession.GetUserId(), "user-1")
		}
		if sessionID != "web-1" {
			t.Fatalf("sessionID = %q, want %q", sessionID, "web-1")
		}

		if err := gateway.RevokeWebSession(ctx, "web-1"); err != nil {
			t.Fatalf("RevokeWebSession() error = %v", err)
		}
		if client.lastRevokeWebSession.GetSessionId() != "web-1" {
			t.Fatalf("session id = %q, want %q", client.lastRevokeWebSession.GetSessionId(), "web-1")
		}

		client.createWebSessionResp = &authv1.CreateWebSessionResponse{}
		_, err = gateway.CreateWebSession(ctx, "user-1")
		if err == nil || apperrors.HTTPStatus(err) != http.StatusInternalServerError {
			t.Fatalf("CreateWebSession(missing session) err = %v", err)
		}
	})
}

type gatewayAuthClientStub struct {
	usernameAvailabilityResp  *authv1.CheckUsernameAvailabilityResponse
	beginPasskeyLoginResp     *authv1.BeginPasskeyLoginResponse
	finishPasskeyLoginResp    *authv1.FinishPasskeyLoginResponse
	beginAccountRecoveryResp  *authv1.BeginAccountRecoveryResponse
	beginRecoveryPasskeyResp  *authv1.BeginPasskeyRegistrationResponse
	finishRecoveryPasskeyResp *authv1.FinishAccountRegistrationResponse
	createWebSessionResp      *authv1.CreateWebSessionResponse

	lastUsernameAvailability  *authv1.CheckUsernameAvailabilityRequest
	lastBeginPasskeyLogin     *authv1.BeginPasskeyLoginRequest
	lastFinishPasskeyLogin    *authv1.FinishPasskeyLoginRequest
	lastBeginAccountRecovery  *authv1.BeginAccountRecoveryRequest
	lastBeginRecoveryPasskey  *authv1.BeginRecoveryPasskeyRegistrationRequest
	lastFinishRecoveryPasskey *authv1.FinishRecoveryPasskeyRegistrationRequest
	lastCreateWebSession      *authv1.CreateWebSessionRequest
	lastRevokeWebSession      *authv1.RevokeWebSessionRequest
}

func (*gatewayAuthClientStub) BeginAccountRegistration(context.Context, *authv1.BeginAccountRegistrationRequest, ...grpc.CallOption) (*authv1.BeginAccountRegistrationResponse, error) {
	return &authv1.BeginAccountRegistrationResponse{SessionId: "reg-1"}, nil
}

func (s *gatewayAuthClientStub) CheckUsernameAvailability(_ context.Context, req *authv1.CheckUsernameAvailabilityRequest, _ ...grpc.CallOption) (*authv1.CheckUsernameAvailabilityResponse, error) {
	s.lastUsernameAvailability = req
	if s.usernameAvailabilityResp != nil {
		return s.usernameAvailabilityResp, nil
	}
	return &authv1.CheckUsernameAvailabilityResponse{}, nil
}

func (*gatewayAuthClientStub) FinishAccountRegistration(context.Context, *authv1.FinishAccountRegistrationRequest, ...grpc.CallOption) (*authv1.FinishAccountRegistrationResponse, error) {
	return &authv1.FinishAccountRegistrationResponse{}, nil
}

func (*gatewayAuthClientStub) AcknowledgeAccountRegistration(context.Context, *authv1.AcknowledgeAccountRegistrationRequest, ...grpc.CallOption) (*authv1.AcknowledgeAccountRegistrationResponse, error) {
	return &authv1.AcknowledgeAccountRegistrationResponse{}, nil
}

func (s *gatewayAuthClientStub) BeginPasskeyLogin(_ context.Context, req *authv1.BeginPasskeyLoginRequest, _ ...grpc.CallOption) (*authv1.BeginPasskeyLoginResponse, error) {
	s.lastBeginPasskeyLogin = req
	if s.beginPasskeyLoginResp != nil {
		return s.beginPasskeyLoginResp, nil
	}
	return &authv1.BeginPasskeyLoginResponse{}, nil
}

func (s *gatewayAuthClientStub) FinishPasskeyLogin(_ context.Context, req *authv1.FinishPasskeyLoginRequest, _ ...grpc.CallOption) (*authv1.FinishPasskeyLoginResponse, error) {
	s.lastFinishPasskeyLogin = req
	if s.finishPasskeyLoginResp != nil {
		return s.finishPasskeyLoginResp, nil
	}
	return &authv1.FinishPasskeyLoginResponse{}, nil
}

func (s *gatewayAuthClientStub) BeginAccountRecovery(_ context.Context, req *authv1.BeginAccountRecoveryRequest, _ ...grpc.CallOption) (*authv1.BeginAccountRecoveryResponse, error) {
	s.lastBeginAccountRecovery = req
	if s.beginAccountRecoveryResp != nil {
		return s.beginAccountRecoveryResp, nil
	}
	return &authv1.BeginAccountRecoveryResponse{}, nil
}

func (s *gatewayAuthClientStub) BeginRecoveryPasskeyRegistration(_ context.Context, req *authv1.BeginRecoveryPasskeyRegistrationRequest, _ ...grpc.CallOption) (*authv1.BeginPasskeyRegistrationResponse, error) {
	s.lastBeginRecoveryPasskey = req
	if s.beginRecoveryPasskeyResp != nil {
		return s.beginRecoveryPasskeyResp, nil
	}
	return &authv1.BeginPasskeyRegistrationResponse{}, nil
}

func (s *gatewayAuthClientStub) FinishRecoveryPasskeyRegistration(_ context.Context, req *authv1.FinishRecoveryPasskeyRegistrationRequest, _ ...grpc.CallOption) (*authv1.FinishAccountRegistrationResponse, error) {
	s.lastFinishRecoveryPasskey = req
	if s.finishRecoveryPasskeyResp != nil {
		return s.finishRecoveryPasskeyResp, nil
	}
	return &authv1.FinishAccountRegistrationResponse{}, nil
}

func (s *gatewayAuthClientStub) CreateWebSession(_ context.Context, req *authv1.CreateWebSessionRequest, _ ...grpc.CallOption) (*authv1.CreateWebSessionResponse, error) {
	s.lastCreateWebSession = req
	if s.createWebSessionResp != nil {
		return s.createWebSessionResp, nil
	}
	return &authv1.CreateWebSessionResponse{}, nil
}

func (s *gatewayAuthClientStub) RevokeWebSession(_ context.Context, req *authv1.RevokeWebSessionRequest, _ ...grpc.CallOption) (*authv1.RevokeWebSessionResponse, error) {
	s.lastRevokeWebSession = req
	return &authv1.RevokeWebSessionResponse{}, nil
}
