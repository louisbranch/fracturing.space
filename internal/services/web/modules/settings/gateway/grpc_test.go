package gateway

import (
	"context"
	"net/http"
	"strings"
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
	return &socialv1.GetUserProfileResponse{UserProfile: &socialv1.UserProfile{Name: "  Rhea  "}}, nil
}
func (socialStub) SetUserProfile(context.Context, *socialv1.SetUserProfileRequest, ...grpc.CallOption) (*socialv1.SetUserProfileResponse, error) {
	return &socialv1.SetUserProfileResponse{}, nil
}

type accountStub struct {
	lastUpdateReq *authv1.UpdateProfileRequest
}

func (accountStub) GetProfile(context.Context, *authv1.GetProfileRequest, ...grpc.CallOption) (*authv1.GetProfileResponse, error) {
	return &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Username: "  rhea  ", Locale: commonv1.Locale_LOCALE_PT_BR}}, nil
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

type agentStub struct {
	lastCreateReq     *aiv1.CreateAgentRequest
	lastListModelsReq *aiv1.ListProviderModelsRequest
	lastListAgentsReq *aiv1.ListAgentsRequest
	listAgentsCalls   int
}

func (a *agentStub) ListAgents(_ context.Context, req *aiv1.ListAgentsRequest, _ ...grpc.CallOption) (*aiv1.ListAgentsResponse, error) {
	a.lastListAgentsReq = req
	a.listAgentsCalls++
	if strings.TrimSpace(req.GetPageToken()) == "" {
		return &aiv1.ListAgentsResponse{Agents: []*aiv1.Agent{{Id: "agent-1", Name: "Narrator", Provider: aiv1.Provider_PROVIDER_OPENAI, Model: "gpt-4o-mini", Status: aiv1.AgentStatus_AGENT_STATUS_ACTIVE, Instructions: "Keep the session moving."}}, NextPageToken: "page-2"}, nil
	}
	return &aiv1.ListAgentsResponse{Agents: []*aiv1.Agent{{Id: "agent-2", Name: "Oracle", Provider: aiv1.Provider_PROVIDER_OPENAI, Model: "gpt-4o", Status: aiv1.AgentStatus_AGENT_STATUS_ACTIVE, Instructions: "Answer briefly."}}}, nil
}

func (a *agentStub) ListProviderModels(_ context.Context, req *aiv1.ListProviderModelsRequest, _ ...grpc.CallOption) (*aiv1.ListProviderModelsResponse, error) {
	a.lastListModelsReq = req
	return &aiv1.ListProviderModelsResponse{Models: []*aiv1.ProviderModel{{Id: "gpt-4o-mini", OwnedBy: "openai"}}}, nil
}

func (a *agentStub) CreateAgent(_ context.Context, req *aiv1.CreateAgentRequest, _ ...grpc.CallOption) (*aiv1.CreateAgentResponse, error) {
	a.lastCreateReq = req
	return &aiv1.CreateAgentResponse{}, nil
}

func TestNewGRPCGatewayWithoutRequiredClientsFailsClosed(t *testing.T) {
	t.Parallel()

	gateway := NewGRPCGateway(nil, nil, nil, nil)
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
	agents := &agentStub{}
	gateway := NewGRPCGateway(socialStub{}, account, credentials, agents)
	profile, err := gateway.LoadProfile(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("LoadProfile() error = %v", err)
	}
	if profile.Username != "rhea" || profile.Name != "Rhea" {
		t.Fatalf("unexpected profile: %+v", profile)
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
		t.Fatalf("unexpected update req: %+v", account.lastUpdateReq)
	}
	if rows, err := gateway.ListAIKeys(context.Background(), "user-1"); err != nil || len(rows) != 1 || rows[0].CanRevoke {
		t.Fatalf("unexpected AI keys: rows=%+v err=%v", rows, err)
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
