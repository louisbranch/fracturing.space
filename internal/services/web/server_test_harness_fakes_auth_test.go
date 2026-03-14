package web

import (
	"context"
	"sync"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"google.golang.org/grpc"
)

type fakeWebAuthClient struct {
	mu       sync.Mutex
	sessions map[string]string
}

type countingWebAuthClient struct {
	*fakeWebAuthClient
	countMu            sync.Mutex
	getWebSessionCalls int
}

func newCountingWebAuthClient() *countingWebAuthClient {
	return &countingWebAuthClient{fakeWebAuthClient: newFakeWebAuthClient()}
}

func (f *countingWebAuthClient) GetWebSession(ctx context.Context, req *authv1.GetWebSessionRequest, opts ...grpc.CallOption) (*authv1.GetWebSessionResponse, error) {
	f.countMu.Lock()
	f.getWebSessionCalls++
	f.countMu.Unlock()
	return f.fakeWebAuthClient.GetWebSession(ctx, req, opts...)
}

func (f *countingWebAuthClient) GetWebSessionCalls() int {
	f.countMu.Lock()
	defer f.countMu.Unlock()
	return f.getWebSessionCalls
}

func newFakeWebAuthClient() *fakeWebAuthClient {
	return &fakeWebAuthClient{sessions: map[string]string{}}
}

func (f *fakeWebAuthClient) BeginAccountRegistration(context.Context, *authv1.BeginAccountRegistrationRequest, ...grpc.CallOption) (*authv1.BeginAccountRegistrationResponse, error) {
	return &authv1.BeginAccountRegistrationResponse{SessionId: "register-session", CredentialCreationOptionsJson: []byte(`{"publicKey":{"challenge":"ZmFrZQ","rp":{"name":"web"},"user":{"id":"dXNlcg","name":"louis","displayName":"louis"},"pubKeyCredParams":[{"type":"public-key","alg":-7}]}}`)}, nil
}

func (f *fakeWebAuthClient) CheckUsernameAvailability(_ context.Context, req *authv1.CheckUsernameAvailabilityRequest, _ ...grpc.CallOption) (*authv1.CheckUsernameAvailabilityResponse, error) {
	username := req.GetUsername()
	if username == "" {
		username = "louis"
	}
	return &authv1.CheckUsernameAvailabilityResponse{
		CanonicalUsername: username,
		State:             authv1.UsernameAvailabilityState_USERNAME_AVAILABILITY_STATE_AVAILABLE,
	}, nil
}

func (f *fakeWebAuthClient) FinishAccountRegistration(context.Context, *authv1.FinishAccountRegistrationRequest, ...grpc.CallOption) (*authv1.FinishAccountRegistrationResponse, error) {
	return &authv1.FinishAccountRegistrationResponse{
		User:         &authv1.User{Id: "user-1", Username: "louis"},
		Session:      &authv1.WebSession{Id: "ws-1", UserId: "user-1"},
		RecoveryCode: "ABCD-EFGH",
	}, nil
}

func (f *fakeWebAuthClient) BeginPasskeyLogin(context.Context, *authv1.BeginPasskeyLoginRequest, ...grpc.CallOption) (*authv1.BeginPasskeyLoginResponse, error) {
	return &authv1.BeginPasskeyLoginResponse{SessionId: "login-session", CredentialRequestOptionsJson: []byte(`{"publicKey":{"challenge":"ZmFrZQ","timeout":60000,"userVerification":"preferred"}}`)}, nil
}

func (f *fakeWebAuthClient) BeginPasskeyRegistration(context.Context, *authv1.BeginPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.BeginPasskeyRegistrationResponse, error) {
	return &authv1.BeginPasskeyRegistrationResponse{SessionId: "passkey-session", CredentialCreationOptionsJson: []byte(`{"publicKey":{"challenge":"ZmFrZQ","rp":{"name":"web"},"user":{"id":"dXNlcg","name":"louis","displayName":"louis"},"pubKeyCredParams":[{"type":"public-key","alg":-7}]}}`)}, nil
}

func (f *fakeWebAuthClient) ListPasskeys(context.Context, *authv1.ListPasskeysRequest, ...grpc.CallOption) (*authv1.ListPasskeysResponse, error) {
	return &authv1.ListPasskeysResponse{}, nil
}

func (f *fakeWebAuthClient) FinishPasskeyRegistration(context.Context, *authv1.FinishPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.FinishPasskeyRegistrationResponse, error) {
	return &authv1.FinishPasskeyRegistrationResponse{User: &authv1.User{Id: "user-1", Username: "louis"}}, nil
}

func (f *fakeWebAuthClient) FinishPasskeyLogin(context.Context, *authv1.FinishPasskeyLoginRequest, ...grpc.CallOption) (*authv1.FinishPasskeyLoginResponse, error) {
	return &authv1.FinishPasskeyLoginResponse{User: &authv1.User{Id: "user-1", Username: "louis"}}, nil
}

func (f *fakeWebAuthClient) BeginAccountRecovery(context.Context, *authv1.BeginAccountRecoveryRequest, ...grpc.CallOption) (*authv1.BeginAccountRecoveryResponse, error) {
	return &authv1.BeginAccountRecoveryResponse{RecoverySessionId: "recovery-session"}, nil
}

func (f *fakeWebAuthClient) BeginRecoveryPasskeyRegistration(context.Context, *authv1.BeginRecoveryPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.BeginPasskeyRegistrationResponse, error) {
	return &authv1.BeginPasskeyRegistrationResponse{SessionId: "recovery-passkey-session", CredentialCreationOptionsJson: []byte(`{"publicKey":{"challenge":"ZmFrZQ","rp":{"name":"web"},"user":{"id":"dXNlcg","name":"louis","displayName":"louis"},"pubKeyCredParams":[{"type":"public-key","alg":-7}]}}`)}, nil
}

func (f *fakeWebAuthClient) FinishRecoveryPasskeyRegistration(context.Context, *authv1.FinishRecoveryPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.FinishAccountRegistrationResponse, error) {
	return &authv1.FinishAccountRegistrationResponse{
		User:         &authv1.User{Id: "user-1", Username: "louis"},
		Session:      &authv1.WebSession{Id: "ws-1", UserId: "user-1"},
		RecoveryCode: "WXYZ-1234",
	}, nil
}

func (f *fakeWebAuthClient) CreateWebSession(_ context.Context, req *authv1.CreateWebSessionRequest, _ ...grpc.CallOption) (*authv1.CreateWebSessionResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := "ws-1"
	f.sessions[id] = req.GetUserId()
	return &authv1.CreateWebSessionResponse{
		Session: &authv1.WebSession{Id: id, UserId: req.GetUserId()},
		User:    &authv1.User{Id: req.GetUserId()},
	}, nil
}

func (f *fakeWebAuthClient) GetWebSession(_ context.Context, req *authv1.GetWebSessionRequest, _ ...grpc.CallOption) (*authv1.GetWebSessionResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	userID, ok := f.sessions[req.GetSessionId()]
	if !ok {
		return nil, context.Canceled
	}
	return &authv1.GetWebSessionResponse{
		Session: &authv1.WebSession{Id: req.GetSessionId(), UserId: userID},
		User:    &authv1.User{Id: userID},
	}, nil
}

func (f *fakeWebAuthClient) RevokeWebSession(_ context.Context, req *authv1.RevokeWebSessionRequest, _ ...grpc.CallOption) (*authv1.RevokeWebSessionResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.sessions, req.GetSessionId())
	return &authv1.RevokeWebSessionResponse{}, nil
}

func (f *fakeWebAuthClient) LookupUserByUsername(_ context.Context, req *authv1.LookupUserByUsernameRequest, _ ...grpc.CallOption) (*authv1.LookupUserByUsernameResponse, error) {
	username := req.GetUsername()
	if username == "" {
		username = "louis"
	}
	return &authv1.LookupUserByUsernameResponse{
		User: &authv1.User{Id: "user-1", Username: username},
	}, nil
}

func (f *fakeWebAuthClient) GetUser(_ context.Context, req *authv1.GetUserRequest, _ ...grpc.CallOption) (*authv1.GetUserResponse, error) {
	return &authv1.GetUserResponse{User: &authv1.User{Id: req.GetUserId(), Username: "viewer"}}, nil
}

func (f *fakeWebAuthClient) IssueJoinGrant(context.Context, *authv1.IssueJoinGrantRequest, ...grpc.CallOption) (*authv1.IssueJoinGrantResponse, error) {
	return &authv1.IssueJoinGrantResponse{JoinGrant: "grant"}, nil
}
