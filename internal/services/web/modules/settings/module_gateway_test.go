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
			Username:      "  rhea  ",
			Name:          "  Rhea Vale  ",
			AvatarSetId:   "  set-a  ",
			AvatarAssetId: "  asset-1  ",
			Bio:           "  Traveler  ",
		}},
	}
	gateway := settingsgateway.GRPCGateway{SocialClient: social}

	profile, err := gateway.LoadProfile(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("LoadProfile() error = %v", err)
	}
	if profile.Username != "rhea" {
		t.Fatalf("Username = %q, want %q", profile.Username, "rhea")
	}
	if profile.Name != "Rhea Vale" {
		t.Fatalf("Name = %q, want %q", profile.Name, "Rhea Vale")
	}

	err = gateway.SaveProfile(context.Background(), "user-1", SettingsProfile{
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
	if social.lastSetReq.GetUsername() != "rhea" {
		t.Fatalf("SetUserProfile username = %q, want %q", social.lastSetReq.GetUsername(), "rhea")
	}
}

func TestGRPCGatewayLoadProfileReturnsEmptyWhenSocialProfileMissing(t *testing.T) {
	t.Parallel()

	social := &socialClientStub{getErr: status.Error(codes.NotFound, "user profile not found")}
	gateway := settingsgateway.GRPCGateway{SocialClient: social}

	profile, err := gateway.LoadProfile(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("LoadProfile() error = %v", err)
	}
	if profile != (SettingsProfile{}) {
		t.Fatalf("profile = %#v, want empty profile", profile)
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
	if account.lastUpdateReq.GetUserId() != "user-1" {
		t.Fatalf("UpdateProfile user id = %q, want %q", account.lastUpdateReq.GetUserId(), "user-1")
	}
	if account.lastUpdateReq.GetLocale() != commonv1.Locale_LOCALE_EN_US {
		t.Fatalf("UpdateProfile locale = %v, want %v", account.lastUpdateReq.GetLocale(), "en-US")
	}
}

func TestGRPCGatewayListCreateAndRevokeAIKeys(t *testing.T) {
	t.Parallel()

	created := timestamppb.New(time.Date(2026, 1, 2, 3, 4, 0, 0, time.UTC))
	credentials := &credentialClientStub{listResp: &aiv1.ListCredentialsResponse{Credentials: []*aiv1.Credential{
		{
			Id:        "cred-1",
			Label:     "Primary",
			Provider:  aiv1.Provider_PROVIDER_OPENAI,
			Status:    aiv1.CredentialStatus_CREDENTIAL_STATUS_ACTIVE,
			CreatedAt: created,
		},
		{
			Id:     "unsafe/id",
			Label:  "Unsafe",
			Status: aiv1.CredentialStatus_CREDENTIAL_STATUS_ACTIVE,
		},
		{
			Id:        "cred-3",
			Label:     "Unknown Provider",
			Provider:  aiv1.Provider_PROVIDER_UNSPECIFIED,
			Status:    aiv1.CredentialStatus_CREDENTIAL_STATUS_UNSPECIFIED,
			CreatedAt: &timestamppb.Timestamp{Seconds: 1, Nanos: 2_000_000_000},
		},
	}}}
	agents := &agentClientStub{}
	gateway := settingsgateway.GRPCGateway{CredentialClient: credentials, AgentClient: agents}

	rows, err := gateway.ListAIKeys(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListAIKeys() error = %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("len(rows) = %d, want 3", len(rows))
	}
	if rows[0].Provider != "OpenAI" {
		t.Fatalf("provider = %q, want %q", rows[0].Provider, "OpenAI")
	}
	if rows[0].Status != "Active" {
		t.Fatalf("status = %q, want %q", rows[0].Status, "Active")
	}
	if rows[0].CreatedAt != "2026-01-02 03:04 UTC" {
		t.Fatalf("created at = %q, want %q", rows[0].CreatedAt, "2026-01-02 03:04 UTC")
	}
	if !rows[0].CanRevoke {
		t.Fatalf("expected first key to be revocable")
	}
	if rows[1].ID != "" || rows[1].CanRevoke {
		t.Fatalf("expected unsafe key path id to be disabled: %#v", rows[1])
	}
	if rows[2].Provider != "Unknown" {
		t.Fatalf("provider = %q, want %q", rows[2].Provider, "Unknown")
	}
	if rows[2].Status != "Unspecified" {
		t.Fatalf("status = %q, want %q", rows[2].Status, "Unspecified")
	}
	if rows[2].CreatedAt != "-" {
		t.Fatalf("created at = %q, want %q", rows[2].CreatedAt, "-")
	}

	err = gateway.CreateAIKey(context.Background(), "user-1", "Primary", "sk-secret")
	if err != nil {
		t.Fatalf("CreateAIKey() error = %v", err)
	}
	if credentials.lastCreateReq.GetProvider() != aiv1.Provider_PROVIDER_OPENAI {
		t.Fatalf("provider = %v, want %v", credentials.lastCreateReq.GetProvider(), aiv1.Provider_PROVIDER_OPENAI)
	}
	if credentials.lastCreateReq.GetLabel() != "Primary" {
		t.Fatalf("label = %q, want %q", credentials.lastCreateReq.GetLabel(), "Primary")
	}

	err = gateway.RevokeAIKey(context.Background(), "user-1", "cred-1")
	if err != nil {
		t.Fatalf("RevokeAIKey() error = %v", err)
	}
	if credentials.lastRevokeReq.GetCredentialId() != "cred-1" {
		t.Fatalf("credential id = %q, want %q", credentials.lastRevokeReq.GetCredentialId(), "cred-1")
	}

	credentialOptions, err := gateway.ListAIAgentCredentials(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListAIAgentCredentials() error = %v", err)
	}
	if len(credentialOptions) != 1 || credentialOptions[0].ID != "cred-1" {
		t.Fatalf("credential options = %+v", credentialOptions)
	}

	models, err := gateway.ListAIProviderModels(context.Background(), "user-1", "cred-1")
	if err != nil {
		t.Fatalf("ListAIProviderModels() error = %v", err)
	}
	if len(models) != 1 || models[0].ID != "gpt-4o-mini" {
		t.Fatalf("models = %+v", models)
	}

	agentRows, err := gateway.ListAIAgents(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListAIAgents() error = %v", err)
	}
	if len(agentRows) != 1 || agentRows[0].Name != "Narrator" {
		t.Fatalf("agent rows = %+v", agentRows)
	}

	err = gateway.CreateAIAgent(context.Background(), "user-1", CreateAIAgentInput{
		Name:         "Narrator",
		CredentialID: "cred-1",
		Model:        "gpt-4o-mini",
		Instructions: "Keep the session moving.",
	})
	if err != nil {
		t.Fatalf("CreateAIAgent() error = %v", err)
	}
	if agents.lastCreateReq == nil || agents.lastCreateReq.GetCredentialId() != "cred-1" || agents.lastCreateReq.GetInstructions() != "Keep the session moving." {
		t.Fatalf("agent create request = %+v", agents.lastCreateReq)
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
		{name: "save profile", run: func() error { return gateway.SaveProfile(context.Background(), "user-1", SettingsProfile{}) }},
		{name: "load locale", run: func() error { _, err := gateway.LoadLocale(context.Background(), "user-1"); return err }},
		{name: "save locale", run: func() error { return gateway.SaveLocale(context.Background(), "user-1", "en-US") }},
		{name: "list ai keys", run: func() error { _, err := gateway.ListAIKeys(context.Background(), "user-1"); return err }},
		{name: "list agent credentials", run: func() error { _, err := gateway.ListAIAgentCredentials(context.Background(), "user-1"); return err }},
		{name: "list ai agents", run: func() error { _, err := gateway.ListAIAgents(context.Background(), "user-1"); return err }},
		{name: "list provider models", run: func() error {
			_, err := gateway.ListAIProviderModels(context.Background(), "user-1", "cred-1")
			return err
		}},
		{name: "create ai key", run: func() error { return gateway.CreateAIKey(context.Background(), "user-1", "label", "secret") }},
		{name: "create ai agent", run: func() error {
			return gateway.CreateAIAgent(context.Background(), "user-1", CreateAIAgentInput{Name: "Narrator", CredentialID: "cred-1", Model: "gpt-4o-mini"})
		}},
		{name: "revoke ai key", run: func() error { return gateway.RevokeAIKey(context.Background(), "user-1", "cred-1") }},
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
	lookupResp *socialv1.LookupUserProfileResponse
	lookupErr  error
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

func (f *socialClientStub) LookupUserProfile(context.Context, *socialv1.LookupUserProfileRequest, ...grpc.CallOption) (*socialv1.LookupUserProfileResponse, error) {
	if f.lookupErr != nil {
		return nil, f.lookupErr
	}
	if f.lookupResp != nil {
		return f.lookupResp, nil
	}
	return &socialv1.LookupUserProfileResponse{}, nil
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
	listAgentsResp    *aiv1.ListAgentsResponse
	listAgentsErr     error
	listModelsResp    *aiv1.ListProviderModelsResponse
	listModelsErr     error
	createErr         error
	lastCreateReq     *aiv1.CreateAgentRequest
	lastListModelsReq *aiv1.ListProviderModelsRequest
}

func (f *agentClientStub) ListAgents(context.Context, *aiv1.ListAgentsRequest, ...grpc.CallOption) (*aiv1.ListAgentsResponse, error) {
	if f.listAgentsErr != nil {
		return nil, f.listAgentsErr
	}
	if f.listAgentsResp != nil {
		return f.listAgentsResp, nil
	}
	return &aiv1.ListAgentsResponse{Agents: []*aiv1.Agent{{
		Id:           "agent-1",
		Name:         "Narrator",
		Provider:     aiv1.Provider_PROVIDER_OPENAI,
		Model:        "gpt-4o-mini",
		Status:       aiv1.AgentStatus_AGENT_STATUS_ACTIVE,
		Instructions: "Keep the session moving.",
	}}}, nil
}

func (f *agentClientStub) ListProviderModels(_ context.Context, req *aiv1.ListProviderModelsRequest, _ ...grpc.CallOption) (*aiv1.ListProviderModelsResponse, error) {
	f.lastListModelsReq = req
	if f.listModelsErr != nil {
		return nil, f.listModelsErr
	}
	if f.listModelsResp != nil {
		return f.listModelsResp, nil
	}
	return &aiv1.ListProviderModelsResponse{Models: []*aiv1.ProviderModel{{
		Id:      "gpt-4o-mini",
		OwnedBy: "openai",
	}}}, nil
}

func (f *agentClientStub) CreateAgent(_ context.Context, req *aiv1.CreateAgentRequest, _ ...grpc.CallOption) (*aiv1.CreateAgentResponse, error) {
	f.lastCreateReq = req
	if f.createErr != nil {
		return nil, f.createErr
	}
	return &aiv1.CreateAgentResponse{}, nil
}
