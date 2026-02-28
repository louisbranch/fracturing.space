package public

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewServiceFailsClosedWhenAuthClientMissing(t *testing.T) {
	t.Parallel()

	svc := newServiceWithGateway(nil)
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

	svc := newServiceWithGateway(NewGRPCAuthGateway(fakeAuthClient{}))
	start, err := svc.passkeyLoginStart(context.Background())
	if err != nil {
		t.Fatalf("passkeyLoginStart() error = %v", err)
	}
	if start.SessionID != "login-session" {
		t.Fatalf("SessionID = %q, want %q", start.SessionID, "login-session")
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

func TestPasskeyLoginFinishPropagatesGatewayErrors(t *testing.T) {
	t.Parallel()

	svcLoginFail := service{auth: &authGatewayStub{finishPasskeyLoginErr: errors.New("auth failed")}}
	_, err := svcLoginFail.passkeyLoginFinish(context.Background(), "session-1", json.RawMessage(`{"id":"cred-1"}`))
	if err == nil {
		t.Fatalf("expected finish passkey login error")
	}

	svcSessionFail := service{auth: &authGatewayStub{createWebSessionErr: errors.New("session failed")}}
	_, err = svcSessionFail.passkeyLoginFinish(context.Background(), "session-1", json.RawMessage(`{"id":"cred-1"}`))
	if err == nil {
		t.Fatalf("expected create web session error")
	}
}

func TestPasskeyRegisterStartValidatesEmailAndGatewayErrors(t *testing.T) {
	t.Parallel()

	svc := service{auth: &authGatewayStub{}}
	if _, err := svc.passkeyRegisterStart(context.Background(), "   "); err == nil {
		t.Fatalf("expected email validation error")
	}

	svcCreateUserFail := service{auth: &authGatewayStub{createUserErr: errors.New("boom")}}
	if _, err := svcCreateUserFail.passkeyRegisterStart(context.Background(), "user@example.com"); err == nil {
		t.Fatalf("expected create user error")
	}

	svcRegisterFail := service{auth: &authGatewayStub{beginPasskeyRegistrationErr: errors.New("boom")}}
	if _, err := svcRegisterFail.passkeyRegisterStart(context.Background(), "user@example.com"); err == nil {
		t.Fatalf("expected begin registration error")
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

func TestPasskeyRegisterFinishPropagatesGatewayError(t *testing.T) {
	t.Parallel()

	svc := service{auth: &authGatewayStub{finishPasskeyRegistrationErr: errors.New("boom")}}
	_, err := svc.passkeyRegisterFinish(context.Background(), "session-1", json.RawMessage(`{"id":"cred-1"}`))
	if err == nil {
		t.Fatalf("expected finish registration error")
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
}

// --- Gateway stub (domain-typed) ---

type authGatewayStub struct {
	createUserResult                string
	createUserErr                   error
	beginPasskeyRegistrationResult  passkeyChallenge
	beginPasskeyRegistrationErr     error
	finishPasskeyRegistrationResult string
	finishPasskeyRegistrationErr    error
	beginPasskeyLoginResult         passkeyChallenge
	beginPasskeyLoginErr            error
	finishPasskeyLoginResult        string
	finishPasskeyLoginErr           error
	createWebSessionResult          string
	createWebSessionErr             error
	hasValidWebSessionResult        bool
	revokeWebSessionErr             error
	revokeCalled                    bool
}

func (f *authGatewayStub) CreateUser(context.Context, string) (string, error) {
	if f.createUserErr != nil {
		return "", f.createUserErr
	}
	if f.createUserResult != "" {
		return f.createUserResult, nil
	}
	return "user-1", nil
}

func (f *authGatewayStub) BeginPasskeyRegistration(context.Context, string) (passkeyChallenge, error) {
	if f.beginPasskeyRegistrationErr != nil {
		return passkeyChallenge{}, f.beginPasskeyRegistrationErr
	}
	if f.beginPasskeyRegistrationResult.SessionID != "" {
		return f.beginPasskeyRegistrationResult, nil
	}
	return passkeyChallenge{SessionID: "register-session", PublicKey: json.RawMessage(`{"publicKey":{}}`)}, nil
}

func (f *authGatewayStub) FinishPasskeyRegistration(context.Context, string, json.RawMessage) (string, error) {
	if f.finishPasskeyRegistrationErr != nil {
		return "", f.finishPasskeyRegistrationErr
	}
	if f.finishPasskeyRegistrationResult != "" {
		return f.finishPasskeyRegistrationResult, nil
	}
	return "user-1", nil
}

func (f *authGatewayStub) BeginPasskeyLogin(context.Context) (passkeyChallenge, error) {
	if f.beginPasskeyLoginErr != nil {
		return passkeyChallenge{}, f.beginPasskeyLoginErr
	}
	if f.beginPasskeyLoginResult.SessionID != "" {
		return f.beginPasskeyLoginResult, nil
	}
	return passkeyChallenge{SessionID: "login-session", PublicKey: json.RawMessage(`{"publicKey":{}}`)}, nil
}

func (f *authGatewayStub) FinishPasskeyLogin(context.Context, string, json.RawMessage) (string, error) {
	if f.finishPasskeyLoginErr != nil {
		return "", f.finishPasskeyLoginErr
	}
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
	return f.hasValidWebSessionResult
}

func (f *authGatewayStub) RevokeWebSession(context.Context, string) error {
	f.revokeCalled = true
	return f.revokeWebSessionErr
}

// --- Gateway tests ---

func TestUnavailableAuthGatewayReturnsUnavailableErrors(t *testing.T) {
	t.Parallel()

	g := unavailableAuthGateway{}
	ctx := context.Background()
	tests := []struct {
		name string
		run  func() error
	}{
		{name: "create user", run: func() error {
			_, err := g.CreateUser(ctx, "user@example.com")
			return err
		}},
		{name: "begin register", run: func() error {
			_, err := g.BeginPasskeyRegistration(ctx, "user-1")
			return err
		}},
		{name: "finish register", run: func() error {
			_, err := g.FinishPasskeyRegistration(ctx, "session-1", json.RawMessage(`{}`))
			return err
		}},
		{name: "begin login", run: func() error {
			_, err := g.BeginPasskeyLogin(ctx)
			return err
		}},
		{name: "finish login", run: func() error {
			_, err := g.FinishPasskeyLogin(ctx, "session-1", json.RawMessage(`{}`))
			return err
		}},
		{name: "create web session", run: func() error {
			_, err := g.CreateWebSession(ctx, "user-1")
			return err
		}},
		{name: "revoke web session", run: func() error {
			return g.RevokeWebSession(ctx, "session-1")
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

	if g.HasValidWebSession(ctx, "session-1") {
		t.Fatalf("expected unavailable gateway to reject sessions")
	}
}

func TestGRPCAuthGatewayMapsProtoToDomainTypes(t *testing.T) {
	t.Parallel()

	client := &recordingAuthClient{
		createUserResp:                &authv1.CreateUserResponse{User: &authv1.User{Id: "user-1"}},
		beginPasskeyRegistrationResp:  &authv1.BeginPasskeyRegistrationResponse{SessionId: "reg-1", CredentialCreationOptionsJson: []byte(`{"publicKey":{}}`)},
		finishPasskeyRegistrationResp: &authv1.FinishPasskeyRegistrationResponse{User: &authv1.User{Id: "user-1"}},
		beginPasskeyLoginResp:         &authv1.BeginPasskeyLoginResponse{SessionId: "login-1", CredentialRequestOptionsJson: []byte(`{"publicKey":{}}`)},
		finishPasskeyLoginResp:        &authv1.FinishPasskeyLoginResponse{User: &authv1.User{Id: "user-1"}},
		createWebSessionResp:          &authv1.CreateWebSessionResponse{Session: &authv1.WebSession{Id: "ws-1", UserId: "user-1"}},
		revokeWebSessionResp:          &authv1.RevokeWebSessionResponse{},
	}
	g := newGRPCAuthGateway(client)

	ctx := context.Background()

	userID, err := g.CreateUser(ctx, "user@example.com")
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

	regChallenge, err := g.BeginPasskeyRegistration(ctx, "user-1")
	if err != nil {
		t.Fatalf("BeginPasskeyRegistration() error = %v", err)
	}
	if regChallenge.SessionID != "reg-1" {
		t.Fatalf("BeginPasskeyRegistration() SessionID = %q, want %q", regChallenge.SessionID, "reg-1")
	}

	finishRegUserID, err := g.FinishPasskeyRegistration(ctx, "reg-1", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("FinishPasskeyRegistration() error = %v", err)
	}
	if finishRegUserID != "user-1" {
		t.Fatalf("FinishPasskeyRegistration() = %q, want %q", finishRegUserID, "user-1")
	}

	loginChallenge, err := g.BeginPasskeyLogin(ctx)
	if err != nil {
		t.Fatalf("BeginPasskeyLogin() error = %v", err)
	}
	if loginChallenge.SessionID != "login-1" {
		t.Fatalf("BeginPasskeyLogin() SessionID = %q, want %q", loginChallenge.SessionID, "login-1")
	}

	finishLoginUserID, err := g.FinishPasskeyLogin(ctx, "login-1", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("FinishPasskeyLogin() error = %v", err)
	}
	if finishLoginUserID != "user-1" {
		t.Fatalf("FinishPasskeyLogin() = %q, want %q", finishLoginUserID, "user-1")
	}

	sessionID, err := g.CreateWebSession(ctx, "user-1")
	if err != nil {
		t.Fatalf("CreateWebSession() error = %v", err)
	}
	if sessionID != "ws-1" {
		t.Fatalf("CreateWebSession() = %q, want %q", sessionID, "ws-1")
	}

	if err := g.RevokeWebSession(ctx, "ws-1"); err != nil {
		t.Fatalf("RevokeWebSession() error = %v", err)
	}
	if client.lastRevokeWebSessionReq == nil || client.lastRevokeWebSessionReq.GetSessionId() != "ws-1" {
		t.Fatalf("RevokeWebSession request was not forwarded")
	}
}

func TestGRPCAuthGatewayRejectsMissingProtoFields(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("CreateUser empty user id", func(t *testing.T) {
		g := newGRPCAuthGateway(&recordingAuthClient{createUserResp: &authv1.CreateUserResponse{}})
		_, err := g.CreateUser(ctx, "user@example.com")
		if err == nil {
			t.Fatalf("expected missing user id error")
		}
	})

	t.Run("BeginPasskeyLogin empty session id", func(t *testing.T) {
		g := newGRPCAuthGateway(&recordingAuthClient{beginPasskeyLoginResp: &authv1.BeginPasskeyLoginResponse{}})
		_, err := g.BeginPasskeyLogin(ctx)
		if err == nil {
			t.Fatalf("expected missing session id error")
		}
	})

	t.Run("FinishPasskeyLogin empty user id", func(t *testing.T) {
		g := newGRPCAuthGateway(&recordingAuthClient{finishPasskeyLoginResp: &authv1.FinishPasskeyLoginResponse{}})
		_, err := g.FinishPasskeyLogin(ctx, "s1", json.RawMessage(`{}`))
		if err == nil {
			t.Fatalf("expected missing user id error")
		}
	})

	t.Run("CreateWebSession empty session id", func(t *testing.T) {
		g := newGRPCAuthGateway(&recordingAuthClient{createWebSessionResp: &authv1.CreateWebSessionResponse{Session: &authv1.WebSession{}}})
		_, err := g.CreateWebSession(ctx, "user-1")
		if err == nil {
			t.Fatalf("expected missing session id error")
		}
	})

	t.Run("BeginPasskeyRegistration empty session id", func(t *testing.T) {
		g := newGRPCAuthGateway(&recordingAuthClient{beginPasskeyRegistrationResp: &authv1.BeginPasskeyRegistrationResponse{}})
		_, err := g.BeginPasskeyRegistration(ctx, "user-1")
		if err == nil {
			t.Fatalf("expected missing session id error")
		}
	})

	t.Run("FinishPasskeyRegistration empty user id", func(t *testing.T) {
		g := newGRPCAuthGateway(&recordingAuthClient{finishPasskeyRegistrationResp: &authv1.FinishPasskeyRegistrationResponse{}})
		_, err := g.FinishPasskeyRegistration(ctx, "s1", json.RawMessage(`{}`))
		if err == nil {
			t.Fatalf("expected missing user id error")
		}
	})
}

func TestGRPCAuthGatewayMapsGrpcErrorsToDomainKinds(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := &recordingAuthClient{
		createUserErr:               status.Error(codes.InvalidArgument, "bad email"),
		finishPasskeyLoginErr:       status.Error(codes.Unauthenticated, "session expired"),
		beginPasskeyRegistrationErr: status.Error(codes.PermissionDenied, "forbidden"),
		beginPasskeyLoginErr:        status.Error(codes.Unavailable, "down"),
	}
	g := newGRPCAuthGateway(client)

	t.Run("create user invalid argument maps to bad input", func(t *testing.T) {
		t.Parallel()
		_, err := g.CreateUser(ctx, "user@example.com")
		if err == nil {
			t.Fatalf("CreateUser() error = nil")
		}
		if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
			t.Fatalf("create user status = %d, want %d", got, http.StatusBadRequest)
		}
		if got := apperrors.LocalizationKey(err); got != "error.http.failed_to_create_user" {
			t.Fatalf("create user localization key = %q, want %q", got, "error.http.failed_to_create_user")
		}
	})

	t.Run("login finish unauthenticated maps to unauthorized", func(t *testing.T) {
		t.Parallel()
		_, err := g.FinishPasskeyLogin(ctx, "session-id", json.RawMessage(`{}`))
		if err == nil {
			t.Fatalf("FinishPasskeyLogin() error = nil")
		}
		if got := apperrors.HTTPStatus(err); got != http.StatusUnauthorized {
			t.Fatalf("login finish status = %d, want %d", got, http.StatusUnauthorized)
		}
	})

	t.Run("registration start permission denied maps to forbidden", func(t *testing.T) {
		t.Parallel()
		_, err := g.BeginPasskeyRegistration(ctx, "user-id")
		if err == nil {
			t.Fatalf("BeginPasskeyRegistration() error = nil")
		}
		if got := apperrors.HTTPStatus(err); got != http.StatusForbidden {
			t.Fatalf("register start status = %d, want %d", got, http.StatusForbidden)
		}
	})

	t.Run("login start unavailable maps to service unavailable", func(t *testing.T) {
		t.Parallel()
		_, err := g.BeginPasskeyLogin(ctx)
		if err == nil {
			t.Fatalf("BeginPasskeyLogin() error = nil")
		}
		if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
			t.Fatalf("login start status = %d, want %d", got, http.StatusServiceUnavailable)
		}
	})
}

// --- Recording gRPC client (for gateway tests) ---

type recordingAuthClient struct {
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
	revokeWebSessionResp          *authv1.RevokeWebSessionResponse
	revokeWebSessionErr           error

	lastCreateUserReq       *authv1.CreateUserRequest
	lastRevokeWebSessionReq *authv1.RevokeWebSessionRequest
}

func (f *recordingAuthClient) CreateUser(_ context.Context, req *authv1.CreateUserRequest, _ ...grpc.CallOption) (*authv1.CreateUserResponse, error) {
	if f.createUserErr != nil {
		return nil, f.createUserErr
	}
	f.lastCreateUserReq = req
	return f.createUserResp, nil
}

func (f *recordingAuthClient) BeginPasskeyRegistration(context.Context, *authv1.BeginPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.BeginPasskeyRegistrationResponse, error) {
	if f.beginPasskeyRegistrationErr != nil {
		return nil, f.beginPasskeyRegistrationErr
	}
	return f.beginPasskeyRegistrationResp, nil
}

func (f *recordingAuthClient) FinishPasskeyRegistration(context.Context, *authv1.FinishPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.FinishPasskeyRegistrationResponse, error) {
	if f.finishPasskeyRegistrationErr != nil {
		return nil, f.finishPasskeyRegistrationErr
	}
	return f.finishPasskeyRegistrationResp, nil
}

func (f *recordingAuthClient) BeginPasskeyLogin(context.Context, *authv1.BeginPasskeyLoginRequest, ...grpc.CallOption) (*authv1.BeginPasskeyLoginResponse, error) {
	if f.beginPasskeyLoginErr != nil {
		return nil, f.beginPasskeyLoginErr
	}
	return f.beginPasskeyLoginResp, nil
}

func (f *recordingAuthClient) FinishPasskeyLogin(context.Context, *authv1.FinishPasskeyLoginRequest, ...grpc.CallOption) (*authv1.FinishPasskeyLoginResponse, error) {
	if f.finishPasskeyLoginErr != nil {
		return nil, f.finishPasskeyLoginErr
	}
	return f.finishPasskeyLoginResp, nil
}

func (f *recordingAuthClient) CreateWebSession(context.Context, *authv1.CreateWebSessionRequest, ...grpc.CallOption) (*authv1.CreateWebSessionResponse, error) {
	if f.createWebSessionErr != nil {
		return nil, f.createWebSessionErr
	}
	return f.createWebSessionResp, nil
}

func (f *recordingAuthClient) GetWebSession(context.Context, *authv1.GetWebSessionRequest, ...grpc.CallOption) (*authv1.GetWebSessionResponse, error) {
	return &authv1.GetWebSessionResponse{}, nil
}

func (f *recordingAuthClient) RevokeWebSession(_ context.Context, req *authv1.RevokeWebSessionRequest, _ ...grpc.CallOption) (*authv1.RevokeWebSessionResponse, error) {
	if f.revokeWebSessionErr != nil {
		return nil, f.revokeWebSessionErr
	}
	f.lastRevokeWebSessionReq = req
	return f.revokeWebSessionResp, nil
}
