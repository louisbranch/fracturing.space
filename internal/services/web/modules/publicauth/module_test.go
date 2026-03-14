package publicauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	publicauthgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	"google.golang.org/grpc"
)

func TestModuleHealthyReflectsGatewayState(t *testing.T) {
	if got := New(Config{}).ID(); got != "public" {
		t.Fatalf("ID() = %q, want %q", got, "public")
	}
	if got := New(Config{Gateway: publicauthgateway.NewGRPCGateway(fakeAuthClient{})}).ID(); got != "public" {
		t.Fatalf("ID() = %q, want %q", got, "public")
	}
}

func TestPasskeyRegisterStartAcceptsUsername(t *testing.T) {
	m := New(Config{Gateway: publicauthgateway.NewGRPCGateway(fakeAuthClient{})})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, routepath.PasskeyRegisterStart, strings.NewReader(`{"username":"louis"}`))
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

type fakeAuthClient struct{}

func (fakeAuthClient) BeginAccountRegistration(context.Context, *authv1.BeginAccountRegistrationRequest, ...grpc.CallOption) (*authv1.BeginAccountRegistrationResponse, error) {
	return &authv1.BeginAccountRegistrationResponse{SessionId: "reg-1", CredentialCreationOptionsJson: []byte(`{"publicKey":{}}`)}, nil
}

func (fakeAuthClient) FinishAccountRegistration(context.Context, *authv1.FinishAccountRegistrationRequest, ...grpc.CallOption) (*authv1.FinishAccountRegistrationResponse, error) {
	return &authv1.FinishAccountRegistrationResponse{User: &authv1.User{Id: "user-1"}, Session: &authv1.WebSession{Id: "web-1"}, RecoveryCode: "ABCD-EFGH"}, nil
}

func (fakeAuthClient) BeginPasskeyLogin(context.Context, *authv1.BeginPasskeyLoginRequest, ...grpc.CallOption) (*authv1.BeginPasskeyLoginResponse, error) {
	return &authv1.BeginPasskeyLoginResponse{SessionId: "login-1", CredentialRequestOptionsJson: []byte(`{"publicKey":{}}`)}, nil
}

func (fakeAuthClient) FinishPasskeyLogin(context.Context, *authv1.FinishPasskeyLoginRequest, ...grpc.CallOption) (*authv1.FinishPasskeyLoginResponse, error) {
	return &authv1.FinishPasskeyLoginResponse{User: &authv1.User{Id: "user-1"}}, nil
}

func (fakeAuthClient) CreateWebSession(context.Context, *authv1.CreateWebSessionRequest, ...grpc.CallOption) (*authv1.CreateWebSessionResponse, error) {
	return &authv1.CreateWebSessionResponse{Session: &authv1.WebSession{Id: "web-1"}}, nil
}

func (fakeAuthClient) GetWebSession(context.Context, *authv1.GetWebSessionRequest, ...grpc.CallOption) (*authv1.GetWebSessionResponse, error) {
	return &authv1.GetWebSessionResponse{}, nil
}

func (fakeAuthClient) RevokeWebSession(context.Context, *authv1.RevokeWebSessionRequest, ...grpc.CallOption) (*authv1.RevokeWebSessionResponse, error) {
	return &authv1.RevokeWebSessionResponse{}, nil
}
