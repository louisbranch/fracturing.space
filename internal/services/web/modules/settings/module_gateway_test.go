package settings

import (
	"context"
	"net/http"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	settingsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/app"
	settingsgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/gateway"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestGRPCGatewayLoadAndSaveProfile(t *testing.T) {
	t.Parallel()

	social := &socialClientStub{
		getResp: &socialv1.GetUserProfileResponse{UserProfile: &socialv1.UserProfile{
			Name:          "  Rhea Vale  ",
			AvatarSetId:   "  set-a  ",
			AvatarAssetId: "  asset-1  ",
			Bio:           "  Traveler  ",
		}},
	}
	account := &accountClientStub{getResp: &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{
		Username: "  rhea  ",
		Locale:   commonv1.Locale_LOCALE_EN_US,
	}}}
	gateway := settingsgateway.GRPCGateway{SocialClient: social, AccountClient: account}

	profile, err := gateway.LoadProfile(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("LoadProfile() error = %v", err)
	}
	if profile.Username != "rhea" || profile.Name != "Rhea Vale" {
		t.Fatalf("unexpected profile: %+v", profile)
	}

	err = gateway.SaveProfile(context.Background(), "user-1", settingsapp.SettingsProfile{
		Username:      "rhea",
		Name:          "Rhea Vale",
		AvatarSetID:   "set-a",
		AvatarAssetID: "asset-1",
		Bio:           "Traveler",
	})
	if err != nil {
		t.Fatalf("SaveProfile() error = %v", err)
	}
	if social.lastSetReq.GetUserId() != "user-1" {
		t.Fatalf("SetUserProfile user id = %q, want %q", social.lastSetReq.GetUserId(), "user-1")
	}
}

func TestGRPCGatewayLoadProfileReturnsUsernameWhenSocialProfileMissing(t *testing.T) {
	t.Parallel()

	social := &socialClientStub{getErr: status.Error(codes.NotFound, "missing")}
	account := &accountClientStub{getResp: &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Username: "rhea"}}}
	gateway := settingsgateway.GRPCGateway{SocialClient: social, AccountClient: account}

	profile, err := gateway.LoadProfile(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("LoadProfile() error = %v", err)
	}
	if profile.Username != "rhea" || profile.Name != "" {
		t.Fatalf("unexpected profile: %+v", profile)
	}
}

func TestGRPCGatewayLoadAndSaveLocale(t *testing.T) {
	t.Parallel()

	account := &accountClientStub{getResp: &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Locale: commonv1.Locale_LOCALE_PT_BR}}}
	gateway := settingsgateway.GRPCGateway{AccountClient: account}

	locale, err := gateway.LoadLocale(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("LoadLocale() error = %v", err)
	}
	if locale != "pt-BR" {
		t.Fatalf("locale = %v, want %v", locale, "pt-BR")
	}

	err = gateway.SaveLocale(context.Background(), "user-1", "en-US")
	if err != nil {
		t.Fatalf("SaveLocale() error = %v", err)
	}
	if account.lastUpdateReq.GetLocale() != commonv1.Locale_LOCALE_EN_US {
		t.Fatalf("UpdateProfile locale = %v, want %v", account.lastUpdateReq.GetLocale(), commonv1.Locale_LOCALE_EN_US)
	}
}

func TestGRPCGatewayListCreateAndRevokeAIKeys(t *testing.T) {
	t.Parallel()

	created := timestamppb.New(time.Date(2026, 1, 2, 3, 4, 0, 0, time.UTC))
	credentials := &credentialClientStub{listResp: &aiv1.ListCredentialsResponse{Credentials: []*aiv1.Credential{{Id: "cred-1", Label: "Primary", Provider: aiv1.Provider_PROVIDER_OPENAI, Status: aiv1.CredentialStatus_CREDENTIAL_STATUS_ACTIVE, CreatedAt: created}}}}
	agents := &agentClientStub{}
	gateway := settingsgateway.GRPCGateway{CredentialClient: credentials, AgentClient: agents}

	rows, err := gateway.ListAIKeys(context.Background(), "user-1")
	if err != nil || len(rows) != 1 || !rows[0].CanRevoke {
		t.Fatalf("unexpected rows=%+v err=%v", rows, err)
	}
	if err := gateway.CreateAIKey(context.Background(), "user-1", "Primary", "sk-secret"); err != nil {
		t.Fatalf("CreateAIKey() error = %v", err)
	}
	if err := gateway.RevokeAIKey(context.Background(), "user-1", "cred-1"); err != nil {
		t.Fatalf("RevokeAIKey() error = %v", err)
	}
}

func TestGRPCGatewayMissingClientBehavior(t *testing.T) {
	t.Parallel()

	gateway := settingsgateway.GRPCGateway{}
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
		{name: "list ai keys", run: func() error { _, err := gateway.ListAIKeys(context.Background(), "user-1"); return err }},
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

type socialClientStub struct {
	getResp    *socialv1.GetUserProfileResponse
	getErr     error
	setErr     error
	lastSetReq *socialv1.SetUserProfileRequest
}

func (f *socialClientStub) GetUserProfile(context.Context, *socialv1.GetUserProfileRequest, ...grpc.CallOption) (*socialv1.GetUserProfileResponse, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.getResp != nil {
		return f.getResp, nil
	}
	return &socialv1.GetUserProfileResponse{}, nil
}

func (f *socialClientStub) SetUserProfile(_ context.Context, req *socialv1.SetUserProfileRequest, _ ...grpc.CallOption) (*socialv1.SetUserProfileResponse, error) {
	f.lastSetReq = req
	if f.setErr != nil {
		return nil, f.setErr
	}
	return &socialv1.SetUserProfileResponse{}, nil
}

type accountClientStub struct {
	getResp       *authv1.GetProfileResponse
	getErr        error
	updateErr     error
	lastUpdateReq *authv1.UpdateProfileRequest
}

func (f *accountClientStub) GetProfile(context.Context, *authv1.GetProfileRequest, ...grpc.CallOption) (*authv1.GetProfileResponse, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.getResp != nil {
		return f.getResp, nil
	}
	return &authv1.GetProfileResponse{}, nil
}

func (f *accountClientStub) UpdateProfile(_ context.Context, req *authv1.UpdateProfileRequest, _ ...grpc.CallOption) (*authv1.UpdateProfileResponse, error) {
	f.lastUpdateReq = req
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	return &authv1.UpdateProfileResponse{}, nil
}

type passkeyClientStub struct {
	listResp   *authv1.ListPasskeysResponse
	listErr    error
	beginErr   error
	finishErr  error
	lastBegin  *authv1.BeginPasskeyRegistrationRequest
	lastFinish *authv1.FinishPasskeyRegistrationRequest
}

func (f *passkeyClientStub) ListPasskeys(context.Context, *authv1.ListPasskeysRequest, ...grpc.CallOption) (*authv1.ListPasskeysResponse, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.listResp != nil {
		return f.listResp, nil
	}
	return &authv1.ListPasskeysResponse{}, nil
}

func (f *passkeyClientStub) BeginPasskeyRegistration(_ context.Context, req *authv1.BeginPasskeyRegistrationRequest, _ ...grpc.CallOption) (*authv1.BeginPasskeyRegistrationResponse, error) {
	f.lastBegin = req
	if f.beginErr != nil {
		return nil, f.beginErr
	}
	return &authv1.BeginPasskeyRegistrationResponse{SessionId: "passkey-session-1", CredentialCreationOptionsJson: []byte(`{"publicKey":{}}`)}, nil
}

func (f *passkeyClientStub) FinishPasskeyRegistration(_ context.Context, req *authv1.FinishPasskeyRegistrationRequest, _ ...grpc.CallOption) (*authv1.FinishPasskeyRegistrationResponse, error) {
	f.lastFinish = req
	if f.finishErr != nil {
		return nil, f.finishErr
	}
	return &authv1.FinishPasskeyRegistrationResponse{}, nil
}

type credentialClientStub struct {
	listResp      *aiv1.ListCredentialsResponse
	listErr       error
	createErr     error
	revokeErr     error
	lastCreateReq *aiv1.CreateCredentialRequest
	lastRevokeReq *aiv1.RevokeCredentialRequest
}

func (f *credentialClientStub) ListCredentials(context.Context, *aiv1.ListCredentialsRequest, ...grpc.CallOption) (*aiv1.ListCredentialsResponse, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.listResp != nil {
		return f.listResp, nil
	}
	return &aiv1.ListCredentialsResponse{}, nil
}

func (f *credentialClientStub) CreateCredential(_ context.Context, req *aiv1.CreateCredentialRequest, _ ...grpc.CallOption) (*aiv1.CreateCredentialResponse, error) {
	f.lastCreateReq = req
	if f.createErr != nil {
		return nil, f.createErr
	}
	return &aiv1.CreateCredentialResponse{}, nil
}

func (f *credentialClientStub) RevokeCredential(_ context.Context, req *aiv1.RevokeCredentialRequest, _ ...grpc.CallOption) (*aiv1.RevokeCredentialResponse, error) {
	f.lastRevokeReq = req
	if f.revokeErr != nil {
		return nil, f.revokeErr
	}
	return &aiv1.RevokeCredentialResponse{}, nil
}

type agentClientStub struct {
	lastCreateReq *aiv1.CreateAgentRequest
}

func (*agentClientStub) ListAgents(context.Context, *aiv1.ListAgentsRequest, ...grpc.CallOption) (*aiv1.ListAgentsResponse, error) {
	return &aiv1.ListAgentsResponse{}, nil
}
func (*agentClientStub) ListProviderModels(context.Context, *aiv1.ListProviderModelsRequest, ...grpc.CallOption) (*aiv1.ListProviderModelsResponse, error) {
	return &aiv1.ListProviderModelsResponse{}, nil
}
func (f *agentClientStub) CreateAgent(_ context.Context, req *aiv1.CreateAgentRequest, _ ...grpc.CallOption) (*aiv1.CreateAgentResponse, error) {
	f.lastCreateReq = req
	return &aiv1.CreateAgentResponse{}, nil
}
