package public

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc"
)

func TestNewServiceFailsClosedWhenAuthClientMissing(t *testing.T) {
	t.Parallel()

	svc := newService(module.Dependencies{})
	_, err := svc.passkeyLoginStart(context.Background())
	if err == nil {
		t.Fatalf("expected unavailable error when auth client is missing")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}
	if svc.hasValidWebSession(context.Background(), "ws-local") {
		t.Fatalf("expected missing auth client to reject web sessions")
	}
}

func TestNewServiceUsesConfiguredAuthClient(t *testing.T) {
	t.Parallel()

	svc := newService(module.Dependencies{AuthClient: fakeAuthClient{}})
	start, err := svc.passkeyLoginStart(context.Background())
	if err != nil {
		t.Fatalf("passkeyLoginStart() error = %v", err)
	}
	if start.sessionID != "login-session" {
		t.Fatalf("sessionID = %q, want %q", start.sessionID, "login-session")
	}
}

func TestPasskeyLoginFinishValidatesInput(t *testing.T) {
	t.Parallel()

	svc := service{auth: &authGatewayStub{}}
	if _, err := svc.passkeyLoginFinish(context.Background(), "", json.RawMessage(`{"id":"cred-1"}`)); err == nil {
		t.Fatalf("expected session id validation error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
	}

	if _, err := svc.passkeyLoginFinish(context.Background(), "session-1", nil); err == nil {
		t.Fatalf("expected credential validation error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
	}
}

func TestPasskeyLoginFinishRequiresUserIDAndSessionID(t *testing.T) {
	t.Parallel()

	svcMissingUser := service{auth: &authGatewayStub{finishPasskeyLoginResp: &authv1.FinishPasskeyLoginResponse{}}}
	_, err := svcMissingUser.passkeyLoginFinish(context.Background(), "session-1", json.RawMessage(`{"id":"cred-1"}`))
	if err == nil {
		t.Fatalf("expected missing user id error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusInternalServerError {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusInternalServerError)
	}

	svcMissingSession := service{auth: &authGatewayStub{
		finishPasskeyLoginResp: &authv1.FinishPasskeyLoginResponse{User: &authv1.User{Id: "user-1"}},
		createWebSessionResp:   &authv1.CreateWebSessionResponse{Session: &authv1.WebSession{}},
	}}
	_, err = svcMissingSession.passkeyLoginFinish(context.Background(), "session-1", json.RawMessage(`{"id":"cred-1"}`))
	if err == nil {
		t.Fatalf("expected missing web session id error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusInternalServerError {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusInternalServerError)
	}
}

func TestPasskeyRegisterStartValidatesEmailAndAuthResponses(t *testing.T) {
	t.Parallel()

	svc := service{auth: &authGatewayStub{}}
	if _, err := svc.passkeyRegisterStart(context.Background(), "   "); err == nil {
		t.Fatalf("expected email validation error")
	}

	missingUser := service{auth: &authGatewayStub{createUserResp: &authv1.CreateUserResponse{}}}
	if _, err := missingUser.passkeyRegisterStart(context.Background(), "user@example.com"); err == nil {
		t.Fatalf("expected missing user id error")
	}

	missingSession := service{auth: &authGatewayStub{
		createUserResp:               &authv1.CreateUserResponse{User: &authv1.User{Id: "user-1"}},
		beginPasskeyRegistrationResp: &authv1.BeginPasskeyRegistrationResponse{},
	}}
	if _, err := missingSession.passkeyRegisterStart(context.Background(), "user@example.com"); err == nil {
		t.Fatalf("expected missing registration session error")
	}
}

func TestPasskeyRegisterFinishValidatesInput(t *testing.T) {
	t.Parallel()

	svc := service{auth: &authGatewayStub{}}
	if _, err := svc.passkeyRegisterFinish(context.Background(), "", json.RawMessage(`{"id":"cred-1"}`)); err == nil {
		t.Fatalf("expected session id validation error")
	}
	if _, err := svc.passkeyRegisterFinish(context.Background(), "session-1", nil); err == nil {
		t.Fatalf("expected credential validation error")
	}
}

func TestPasskeyRegisterFinishRequiresUserID(t *testing.T) {
	t.Parallel()

	svc := service{auth: &authGatewayStub{finishPasskeyRegistrationResp: &authv1.FinishPasskeyRegistrationResponse{}}}
	_, err := svc.passkeyRegisterFinish(context.Background(), "session-1", json.RawMessage(`{"id":"cred-1"}`))
	if err == nil {
		t.Fatalf("expected missing user id error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusInternalServerError {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusInternalServerError)
	}
}

func TestRevokeWebSessionHandlesEmptyAndGatewayError(t *testing.T) {
	t.Parallel()

	stub := &authGatewayStub{}
	svc := service{auth: stub}
	if err := svc.revokeWebSession(context.Background(), ""); err != nil {
		t.Fatalf("revokeWebSession(empty) error = %v", err)
	}
	if stub.revokeCalled {
		t.Fatalf("expected revoke not called for empty session id")
	}

	svc = service{auth: &authGatewayStub{revokeWebSessionErr: errors.New("boom")}}
	err := svc.revokeWebSession(context.Background(), "session-1")
	if err == nil {
		t.Fatalf("expected revoke failure")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusInternalServerError {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusInternalServerError)
	}
}

type authGatewayStub struct {
	createUserResp                *authv1.CreateUserResponse
	createUserErr                 error
	beginPasskeyRegistrationResp  *authv1.BeginPasskeyRegistrationResponse
	beginPasskeyRegistrationErr   error
	finishPasskeyRegistrationResp *authv1.FinishPasskeyRegistrationResponse
	finishPasskeyRegistrationErr  error
	beginPasskeyLoginResp         *authv1.BeginPasskeyLoginResponse
	beginPasskeyLoginErr          error
	finishPasskeyLoginResp        *authv1.FinishPasskeyLoginResponse
	finishPasskeyLoginErr         error
	createWebSessionResp          *authv1.CreateWebSessionResponse
	createWebSessionErr           error
	getWebSessionResp             *authv1.GetWebSessionResponse
	getWebSessionErr              error
	revokeWebSessionErr           error
	revokeCalled                  bool
}

func (f *authGatewayStub) CreateUser(context.Context, *authv1.CreateUserRequest) (*authv1.CreateUserResponse, error) {
	if f.createUserErr != nil {
		return nil, f.createUserErr
	}
	if f.createUserResp != nil {
		return f.createUserResp, nil
	}
	return &authv1.CreateUserResponse{User: &authv1.User{Id: "user-1"}}, nil
}

func (f *authGatewayStub) BeginPasskeyRegistration(context.Context, *authv1.BeginPasskeyRegistrationRequest) (*authv1.BeginPasskeyRegistrationResponse, error) {
	if f.beginPasskeyRegistrationErr != nil {
		return nil, f.beginPasskeyRegistrationErr
	}
	if f.beginPasskeyRegistrationResp != nil {
		return f.beginPasskeyRegistrationResp, nil
	}
	return &authv1.BeginPasskeyRegistrationResponse{SessionId: "register-session", CredentialCreationOptionsJson: []byte(`{"publicKey":{}}`)}, nil
}

func (f *authGatewayStub) FinishPasskeyRegistration(context.Context, *authv1.FinishPasskeyRegistrationRequest) (*authv1.FinishPasskeyRegistrationResponse, error) {
	if f.finishPasskeyRegistrationErr != nil {
		return nil, f.finishPasskeyRegistrationErr
	}
	if f.finishPasskeyRegistrationResp != nil {
		return f.finishPasskeyRegistrationResp, nil
	}
	return &authv1.FinishPasskeyRegistrationResponse{User: &authv1.User{Id: "user-1"}}, nil
}

func (f *authGatewayStub) BeginPasskeyLogin(context.Context, *authv1.BeginPasskeyLoginRequest) (*authv1.BeginPasskeyLoginResponse, error) {
	if f.beginPasskeyLoginErr != nil {
		return nil, f.beginPasskeyLoginErr
	}
	if f.beginPasskeyLoginResp != nil {
		return f.beginPasskeyLoginResp, nil
	}
	return &authv1.BeginPasskeyLoginResponse{SessionId: "login-session", CredentialRequestOptionsJson: []byte(`{"publicKey":{}}`)}, nil
}

func (f *authGatewayStub) FinishPasskeyLogin(context.Context, *authv1.FinishPasskeyLoginRequest) (*authv1.FinishPasskeyLoginResponse, error) {
	if f.finishPasskeyLoginErr != nil {
		return nil, f.finishPasskeyLoginErr
	}
	if f.finishPasskeyLoginResp != nil {
		return f.finishPasskeyLoginResp, nil
	}
	return &authv1.FinishPasskeyLoginResponse{User: &authv1.User{Id: "user-1"}}, nil
}

func (f *authGatewayStub) CreateWebSession(context.Context, *authv1.CreateWebSessionRequest) (*authv1.CreateWebSessionResponse, error) {
	if f.createWebSessionErr != nil {
		return nil, f.createWebSessionErr
	}
	if f.createWebSessionResp != nil {
		return f.createWebSessionResp, nil
	}
	return &authv1.CreateWebSessionResponse{Session: &authv1.WebSession{Id: "ws-1", UserId: "user-1"}}, nil
}

func (f *authGatewayStub) GetWebSession(context.Context, *authv1.GetWebSessionRequest) (*authv1.GetWebSessionResponse, error) {
	if f.getWebSessionErr != nil {
		return nil, f.getWebSessionErr
	}
	if f.getWebSessionResp != nil {
		return f.getWebSessionResp, nil
	}
	return &authv1.GetWebSessionResponse{Session: &authv1.WebSession{Id: "ws-1", UserId: "user-1"}, User: &authv1.User{Id: "user-1"}}, nil
}

func (f *authGatewayStub) RevokeWebSession(context.Context, *authv1.RevokeWebSessionRequest) (*authv1.RevokeWebSessionResponse, error) {
	f.revokeCalled = true
	if f.revokeWebSessionErr != nil {
		return nil, f.revokeWebSessionErr
	}
	return &authv1.RevokeWebSessionResponse{}, nil
}

func TestPasskeyRegisterStartUsesEnglishLocale(t *testing.T) {
	t.Parallel()

	stub := &createUserCaptureAuthGateway{}
	svc := service{auth: stub}
	_, err := svc.passkeyRegisterStart(context.Background(), "captured@example.com")
	if err != nil {
		t.Fatalf("passkeyRegisterStart() error = %v", err)
	}
	if stub.lastCreateUserReq == nil {
		t.Fatalf("expected CreateUser request")
	}
	if stub.lastCreateUserReq.GetLocale() != commonv1.Locale_LOCALE_EN_US {
		t.Fatalf("locale = %v, want %v", stub.lastCreateUserReq.GetLocale(), commonv1.Locale_LOCALE_EN_US)
	}
}

func TestUnavailableAuthGatewayReturnsUnavailableErrors(t *testing.T) {
	t.Parallel()

	g := unavailableAuthGateway{}
	ctx := context.Background()
	tests := []struct {
		name string
		run  func() error
	}{
		{name: "create user", run: func() error {
			_, err := g.CreateUser(ctx, &authv1.CreateUserRequest{Email: "user@example.com"})
			return err
		}},
		{name: "begin register", run: func() error {
			_, err := g.BeginPasskeyRegistration(ctx, &authv1.BeginPasskeyRegistrationRequest{UserId: "user-1"})
			return err
		}},
		{name: "finish register", run: func() error {
			_, err := g.FinishPasskeyRegistration(ctx, &authv1.FinishPasskeyRegistrationRequest{SessionId: "session-1"})
			return err
		}},
		{name: "begin login", run: func() error {
			_, err := g.BeginPasskeyLogin(ctx, &authv1.BeginPasskeyLoginRequest{})
			return err
		}},
		{name: "finish login", run: func() error {
			_, err := g.FinishPasskeyLogin(ctx, &authv1.FinishPasskeyLoginRequest{SessionId: "session-1"})
			return err
		}},
		{name: "create web session", run: func() error {
			_, err := g.CreateWebSession(ctx, &authv1.CreateWebSessionRequest{UserId: "user-1"})
			return err
		}},
		{name: "get web session", run: func() error {
			_, err := g.GetWebSession(ctx, &authv1.GetWebSessionRequest{SessionId: "session-1"})
			return err
		}},
		{name: "revoke web session", run: func() error {
			_, err := g.RevokeWebSession(ctx, &authv1.RevokeWebSessionRequest{SessionId: "session-1"})
			return err
		}},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.run()
			if err == nil {
				t.Fatalf("expected unavailable error")
			}
			if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
				t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
			}
		})
	}
}

func TestGRPCAuthGatewayForwardsClientCalls(t *testing.T) {
	t.Parallel()

	client := &recordingAuthClient{
		createUserResp:                &authv1.CreateUserResponse{User: &authv1.User{Id: "user-1"}},
		beginPasskeyRegistrationResp:  &authv1.BeginPasskeyRegistrationResponse{SessionId: "reg-1"},
		finishPasskeyRegistrationResp: &authv1.FinishPasskeyRegistrationResponse{User: &authv1.User{Id: "user-1"}},
		beginPasskeyLoginResp:         &authv1.BeginPasskeyLoginResponse{SessionId: "login-1"},
		finishPasskeyLoginResp:        &authv1.FinishPasskeyLoginResponse{User: &authv1.User{Id: "user-1"}},
		createWebSessionResp:          &authv1.CreateWebSessionResponse{Session: &authv1.WebSession{Id: "ws-1", UserId: "user-1"}},
		revokeWebSessionResp:          &authv1.RevokeWebSessionResponse{},
	}
	g := grpcAuthGateway{client: client}

	ctx := context.Background()
	createReq := &authv1.CreateUserRequest{Email: "user@example.com"}
	if resp, err := g.CreateUser(ctx, createReq); err != nil || resp != client.createUserResp {
		t.Fatalf("CreateUser() = (%v, %v), want (%v, nil)", resp, err, client.createUserResp)
	}
	if client.lastCreateUserReq != createReq {
		t.Fatalf("CreateUser request was not forwarded")
	}

	beginRegReq := &authv1.BeginPasskeyRegistrationRequest{UserId: "user-1"}
	if resp, err := g.BeginPasskeyRegistration(ctx, beginRegReq); err != nil || resp != client.beginPasskeyRegistrationResp {
		t.Fatalf("BeginPasskeyRegistration() = (%v, %v)", resp, err)
	}

	finishRegReq := &authv1.FinishPasskeyRegistrationRequest{SessionId: "reg-1"}
	if resp, err := g.FinishPasskeyRegistration(ctx, finishRegReq); err != nil || resp != client.finishPasskeyRegistrationResp {
		t.Fatalf("FinishPasskeyRegistration() = (%v, %v)", resp, err)
	}

	beginLoginReq := &authv1.BeginPasskeyLoginRequest{}
	if resp, err := g.BeginPasskeyLogin(ctx, beginLoginReq); err != nil || resp != client.beginPasskeyLoginResp {
		t.Fatalf("BeginPasskeyLogin() = (%v, %v)", resp, err)
	}

	finishLoginReq := &authv1.FinishPasskeyLoginRequest{SessionId: "login-1"}
	if resp, err := g.FinishPasskeyLogin(ctx, finishLoginReq); err != nil || resp != client.finishPasskeyLoginResp {
		t.Fatalf("FinishPasskeyLogin() = (%v, %v)", resp, err)
	}

	createSessionReq := &authv1.CreateWebSessionRequest{UserId: "user-1"}
	if resp, err := g.CreateWebSession(ctx, createSessionReq); err != nil || resp != client.createWebSessionResp {
		t.Fatalf("CreateWebSession() = (%v, %v)", resp, err)
	}

	getSessionReq := &authv1.GetWebSessionRequest{SessionId: "ws-1"}
	if _, err := g.GetWebSession(ctx, getSessionReq); err != nil {
		t.Fatalf("GetWebSession() error = %v", err)
	}

	revokeReq := &authv1.RevokeWebSessionRequest{SessionId: "ws-1"}
	if resp, err := g.RevokeWebSession(ctx, revokeReq); err != nil || resp != client.revokeWebSessionResp {
		t.Fatalf("RevokeWebSession() = (%v, %v)", resp, err)
	}
	if client.lastRevokeWebSessionReq != revokeReq {
		t.Fatalf("RevokeWebSession request was not forwarded")
	}
}

type createUserCaptureAuthGateway struct {
	authGatewayStub
	lastCreateUserReq *authv1.CreateUserRequest
}

func (f *createUserCaptureAuthGateway) CreateUser(_ context.Context, req *authv1.CreateUserRequest) (*authv1.CreateUserResponse, error) {
	f.lastCreateUserReq = req
	return &authv1.CreateUserResponse{User: &authv1.User{Id: "user-1"}}, nil
}

type recordingAuthClient struct {
	createUserResp                *authv1.CreateUserResponse
	beginPasskeyRegistrationResp  *authv1.BeginPasskeyRegistrationResponse
	finishPasskeyRegistrationResp *authv1.FinishPasskeyRegistrationResponse
	beginPasskeyLoginResp         *authv1.BeginPasskeyLoginResponse
	finishPasskeyLoginResp        *authv1.FinishPasskeyLoginResponse
	createWebSessionResp          *authv1.CreateWebSessionResponse
	revokeWebSessionResp          *authv1.RevokeWebSessionResponse

	lastCreateUserReq       *authv1.CreateUserRequest
	lastRevokeWebSessionReq *authv1.RevokeWebSessionRequest
}

func (f *recordingAuthClient) CreateUser(_ context.Context, req *authv1.CreateUserRequest, _ ...grpc.CallOption) (*authv1.CreateUserResponse, error) {
	f.lastCreateUserReq = req
	return f.createUserResp, nil
}

func (f *recordingAuthClient) BeginPasskeyRegistration(context.Context, *authv1.BeginPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.BeginPasskeyRegistrationResponse, error) {
	return f.beginPasskeyRegistrationResp, nil
}

func (f *recordingAuthClient) FinishPasskeyRegistration(context.Context, *authv1.FinishPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.FinishPasskeyRegistrationResponse, error) {
	return f.finishPasskeyRegistrationResp, nil
}

func (f *recordingAuthClient) BeginPasskeyLogin(context.Context, *authv1.BeginPasskeyLoginRequest, ...grpc.CallOption) (*authv1.BeginPasskeyLoginResponse, error) {
	return f.beginPasskeyLoginResp, nil
}

func (f *recordingAuthClient) FinishPasskeyLogin(context.Context, *authv1.FinishPasskeyLoginRequest, ...grpc.CallOption) (*authv1.FinishPasskeyLoginResponse, error) {
	return f.finishPasskeyLoginResp, nil
}

func (f *recordingAuthClient) CreateWebSession(context.Context, *authv1.CreateWebSessionRequest, ...grpc.CallOption) (*authv1.CreateWebSessionResponse, error) {
	return f.createWebSessionResp, nil
}

func (f *recordingAuthClient) GetWebSession(context.Context, *authv1.GetWebSessionRequest, ...grpc.CallOption) (*authv1.GetWebSessionResponse, error) {
	return &authv1.GetWebSessionResponse{}, nil
}

func (f *recordingAuthClient) RevokeWebSession(_ context.Context, req *authv1.RevokeWebSessionRequest, _ ...grpc.CallOption) (*authv1.RevokeWebSessionResponse, error) {
	f.lastRevokeWebSessionReq = req
	return f.revokeWebSessionResp, nil
}
