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

func (f *fakeWebAuthClient) CreateUser(context.Context, *authv1.CreateUserRequest, ...grpc.CallOption) (*authv1.CreateUserResponse, error) {
	return &authv1.CreateUserResponse{User: &authv1.User{Id: "user-1"}}, nil
}

func (f *fakeWebAuthClient) BeginPasskeyRegistration(context.Context, *authv1.BeginPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.BeginPasskeyRegistrationResponse, error) {
	return &authv1.BeginPasskeyRegistrationResponse{SessionId: "register-session", CredentialCreationOptionsJson: []byte(`{"publicKey":{"challenge":"ZmFrZQ","rp":{"name":"web"},"user":{"id":"dXNlcg","name":"new@example.com","displayName":"new@example.com"},"pubKeyCredParams":[{"type":"public-key","alg":-7}]}}`)}, nil
}

func (f *fakeWebAuthClient) FinishPasskeyRegistration(context.Context, *authv1.FinishPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.FinishPasskeyRegistrationResponse, error) {
	return &authv1.FinishPasskeyRegistrationResponse{User: &authv1.User{Id: "user-1"}}, nil
}

func (f *fakeWebAuthClient) BeginPasskeyLogin(context.Context, *authv1.BeginPasskeyLoginRequest, ...grpc.CallOption) (*authv1.BeginPasskeyLoginResponse, error) {
	return &authv1.BeginPasskeyLoginResponse{SessionId: "login-session", CredentialRequestOptionsJson: []byte(`{"publicKey":{"challenge":"ZmFrZQ","timeout":60000,"userVerification":"preferred"}}`)}, nil
}

func (f *fakeWebAuthClient) FinishPasskeyLogin(context.Context, *authv1.FinishPasskeyLoginRequest, ...grpc.CallOption) (*authv1.FinishPasskeyLoginResponse, error) {
	return &authv1.FinishPasskeyLoginResponse{User: &authv1.User{Id: "user-1"}}, nil
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
