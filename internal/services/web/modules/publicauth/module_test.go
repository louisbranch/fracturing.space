package publicauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	publicauthgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	"google.golang.org/grpc"
)

func TestModuleHealthyReflectsGatewayState(t *testing.T) {
	if got := New(Config{}).ID(); got != "public" {
		t.Fatalf("ID() = %q, want %q", got, "public")
	}
	if got := newModuleFromGateway(publicauthgateway.NewGRPCGateway(fakeAuthClient{}), "").ID(); got != "public" {
		t.Fatalf("ID() = %q, want %q", got, "public")
	}
}

func TestPasskeyRegisterStartAcceptsUsername(t *testing.T) {
	m := newModuleFromGateway(publicauthgateway.NewGRPCGateway(fakeAuthClient{}), "")
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

func TestSurfaceSelectionControlsModuleIdentityAndPrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		surface    Surface
		wantID     string
		wantPrefix string
	}{
		{name: "default", surface: "", wantID: "public", wantPrefix: routepath.Root},
		{name: "shell", surface: SurfaceShell, wantID: "public", wantPrefix: routepath.Root},
		{name: "passkeys", surface: SurfacePasskeys, wantID: "public-passkeys", wantPrefix: routepath.PasskeysPrefix},
		{name: "auth redirect", surface: SurfaceAuthRedirect, wantID: "public-auth-redirect", wantPrefix: routepath.AuthPrefix},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := newModuleFromGateway(nil, "", withRequestMeta(requestmeta.SchemePolicy{}), withSurface(tc.surface))
			if got := m.ID(); got != tc.wantID {
				t.Fatalf("ID() = %q, want %q", got, tc.wantID)
			}
			mount, err := m.Mount()
			if err != nil {
				t.Fatalf("Mount() error = %v", err)
			}
			if got := mount.Prefix; got != tc.wantPrefix {
				t.Fatalf("prefix = %q, want %q", got, tc.wantPrefix)
			}
		})
	}
}

type fakeAuthClient struct{}

func (fakeAuthClient) BeginAccountRegistration(context.Context, *authv1.BeginAccountRegistrationRequest, ...grpc.CallOption) (*authv1.BeginAccountRegistrationResponse, error) {
	return &authv1.BeginAccountRegistrationResponse{SessionId: "reg-1", CredentialCreationOptionsJson: []byte(`{"publicKey":{}}`)}, nil
}

func (fakeAuthClient) CheckUsernameAvailability(context.Context, *authv1.CheckUsernameAvailabilityRequest, ...grpc.CallOption) (*authv1.CheckUsernameAvailabilityResponse, error) {
	return &authv1.CheckUsernameAvailabilityResponse{
		CanonicalUsername: "louis",
		State:             authv1.UsernameAvailabilityState_USERNAME_AVAILABILITY_STATE_AVAILABLE,
	}, nil
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

func (fakeAuthClient) BeginAccountRecovery(context.Context, *authv1.BeginAccountRecoveryRequest, ...grpc.CallOption) (*authv1.BeginAccountRecoveryResponse, error) {
	return &authv1.BeginAccountRecoveryResponse{RecoverySessionId: "recover-1"}, nil
}

func (fakeAuthClient) BeginRecoveryPasskeyRegistration(context.Context, *authv1.BeginRecoveryPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.BeginPasskeyRegistrationResponse, error) {
	return &authv1.BeginPasskeyRegistrationResponse{SessionId: "recovery-passkey-1", CredentialCreationOptionsJson: []byte(`{"publicKey":{}}`)}, nil
}

func (fakeAuthClient) FinishRecoveryPasskeyRegistration(context.Context, *authv1.FinishRecoveryPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.FinishAccountRegistrationResponse, error) {
	return &authv1.FinishAccountRegistrationResponse{
		User:         &authv1.User{Id: "user-1"},
		Session:      &authv1.WebSession{Id: "web-1"},
		RecoveryCode: "WXYZ-1234",
	}, nil
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
