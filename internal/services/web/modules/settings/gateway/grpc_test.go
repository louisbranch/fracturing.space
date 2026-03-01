package gateway

import (
	"context"
	"net/http"
	"testing"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	settingsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc"
)

type socialStub struct{}

func (socialStub) GetUserProfile(context.Context, *socialv1.GetUserProfileRequest, ...grpc.CallOption) (*socialv1.GetUserProfileResponse, error) {
	return &socialv1.GetUserProfileResponse{UserProfile: &socialv1.UserProfile{Username: "  rhea  "}}, nil
}
func (socialStub) LookupUserProfile(context.Context, *socialv1.LookupUserProfileRequest, ...grpc.CallOption) (*socialv1.LookupUserProfileResponse, error) {
	return &socialv1.LookupUserProfileResponse{}, nil
}
func (socialStub) SetUserProfile(context.Context, *socialv1.SetUserProfileRequest, ...grpc.CallOption) (*socialv1.SetUserProfileResponse, error) {
	return &socialv1.SetUserProfileResponse{}, nil
}

type accountStub struct {
	lastUpdateReq *authv1.UpdateProfileRequest
}

func (accountStub) GetProfile(context.Context, *authv1.GetProfileRequest, ...grpc.CallOption) (*authv1.GetProfileResponse, error) {
	return &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Locale: commonv1.Locale_LOCALE_PT_BR}}, nil
}
func (a *accountStub) UpdateProfile(_ context.Context, req *authv1.UpdateProfileRequest, _ ...grpc.CallOption) (*authv1.UpdateProfileResponse, error) {
	a.lastUpdateReq = req
	return &authv1.UpdateProfileResponse{}, nil
}

type credentialStub struct {
	lastCreateReq *aiv1.CreateCredentialRequest
	lastRevokeReq *aiv1.RevokeCredentialRequest
}

func (credentialStub) ListCredentials(context.Context, *aiv1.ListCredentialsRequest, ...grpc.CallOption) (*aiv1.ListCredentialsResponse, error) {
	return &aiv1.ListCredentialsResponse{Credentials: []*aiv1.Credential{{
		Id:       "unsafe/id",
		Label:    "Primary",
		Provider: aiv1.Provider_PROVIDER_OPENAI,
		Status:   aiv1.CredentialStatus_CREDENTIAL_STATUS_ACTIVE,
	}}}, nil
}
func (c *credentialStub) CreateCredential(_ context.Context, req *aiv1.CreateCredentialRequest, _ ...grpc.CallOption) (*aiv1.CreateCredentialResponse, error) {
	c.lastCreateReq = req
	return &aiv1.CreateCredentialResponse{}, nil
}
func (c *credentialStub) RevokeCredential(_ context.Context, req *aiv1.RevokeCredentialRequest, _ ...grpc.CallOption) (*aiv1.RevokeCredentialResponse, error) {
	c.lastRevokeReq = req
	return &aiv1.RevokeCredentialResponse{}, nil
}

func TestNewGRPCGatewayWithoutRequiredClientsFailsClosed(t *testing.T) {
	t.Parallel()

	gateway := NewGRPCGateway(nil, nil, nil)
	_, err := gateway.LoadProfile(context.Background(), "user-1")
	if err == nil {
		t.Fatalf("expected unavailable error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

func TestGRPCGatewayMapsProfileAndLocale(t *testing.T) {
	t.Parallel()

	account := &accountStub{}
	credentials := &credentialStub{}
	gateway := NewGRPCGateway(socialStub{}, account, credentials)
	profile, err := gateway.LoadProfile(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("LoadProfile() error = %v", err)
	}
	if profile.Username != "rhea" {
		t.Fatalf("Username = %q, want %q", profile.Username, "rhea")
	}
	locale, err := gateway.LoadLocale(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("LoadLocale() error = %v", err)
	}
	if locale != "pt-BR" {
		t.Fatalf("locale = %q, want %q", locale, "pt-BR")
	}

	if err := gateway.SaveLocale(context.Background(), "user-1", "en-US"); err != nil {
		t.Fatalf("SaveLocale() error = %v", err)
	}
	if account.lastUpdateReq == nil || account.lastUpdateReq.GetLocale() != commonv1.Locale_LOCALE_EN_US {
		t.Fatalf("UpdateProfile locale = %v, want %v", account.lastUpdateReq.GetLocale(), commonv1.Locale_LOCALE_EN_US)
	}

	rows, err := gateway.ListAIKeys(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListAIKeys() error = %v", err)
	}
	if len(rows) != 1 || rows[0].ID != "" || rows[0].CanRevoke {
		t.Fatalf("expected unsafe key to be normalized as non-revocable: %+v", rows)
	}

	if err := gateway.CreateAIKey(context.Background(), "user-1", "Primary", "secret"); err != nil {
		t.Fatalf("CreateAIKey() error = %v", err)
	}
	if credentials.lastCreateReq == nil || credentials.lastCreateReq.GetProvider() != aiv1.Provider_PROVIDER_OPENAI {
		t.Fatalf("CreateCredential request not captured as expected")
	}

	if err := gateway.RevokeAIKey(context.Background(), "user-1", "cred-1"); err != nil {
		t.Fatalf("RevokeAIKey() error = %v", err)
	}
	if credentials.lastRevokeReq == nil || credentials.lastRevokeReq.GetCredentialId() != "cred-1" {
		t.Fatalf("RevokeCredential credential id = %q, want %q", credentials.lastRevokeReq.GetCredentialId(), "cred-1")
	}
}

func TestGRPCGatewayMissingClientBehavior(t *testing.T) {
	t.Parallel()

	gateway := GRPCGateway{}
	tests := []struct {
		name string
		run  func() error
	}{
		{name: "load profile", run: func() error { _, err := gateway.LoadProfile(context.Background(), "user-1"); return err }},
		{name: "save profile", run: func() error {
			return gateway.SaveProfile(context.Background(), "user-1", settingsapp.SettingsProfile{})
		}},
		{name: "load locale", run: func() error { _, err := gateway.LoadLocale(context.Background(), "user-1"); return err }},
		{name: "save locale", run: func() error { return gateway.SaveLocale(context.Background(), "user-1", "en-US") }},
		{name: "list keys", run: func() error { _, err := gateway.ListAIKeys(context.Background(), "user-1"); return err }},
		{name: "create key", run: func() error { return gateway.CreateAIKey(context.Background(), "user-1", "label", "secret") }},
		{name: "revoke key", run: func() error { return gateway.RevokeAIKey(context.Background(), "user-1", "cred-1") }},
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
