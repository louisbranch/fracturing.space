package gateway

import (
	"context"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	settingsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/app"
	"google.golang.org/grpc"
)

// SocialClient exposes profile lookup and mutation operations.
type SocialClient interface {
	GetUserProfile(context.Context, *socialv1.GetUserProfileRequest, ...grpc.CallOption) (*socialv1.GetUserProfileResponse, error)
	SetUserProfile(context.Context, *socialv1.SetUserProfileRequest, ...grpc.CallOption) (*socialv1.SetUserProfileResponse, error)
}

// AccountClient exposes account profile read/update operations.
type AccountClient interface {
	GetProfile(context.Context, *authv1.GetProfileRequest, ...grpc.CallOption) (*authv1.GetProfileResponse, error)
	UpdateProfile(context.Context, *authv1.UpdateProfileRequest, ...grpc.CallOption) (*authv1.UpdateProfileResponse, error)
}

// PasskeyClient exposes authenticated passkey settings operations.
type PasskeyClient interface {
	ListPasskeys(context.Context, *authv1.ListPasskeysRequest, ...grpc.CallOption) (*authv1.ListPasskeysResponse, error)
	BeginPasskeyRegistration(context.Context, *authv1.BeginPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.BeginPasskeyRegistrationResponse, error)
	FinishPasskeyRegistration(context.Context, *authv1.FinishPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.FinishPasskeyRegistrationResponse, error)
}

// CredentialClient exposes AI credential listing and mutation operations.
type CredentialClient interface {
	ListCredentials(context.Context, *aiv1.ListCredentialsRequest, ...grpc.CallOption) (*aiv1.ListCredentialsResponse, error)
	CreateCredential(context.Context, *aiv1.CreateCredentialRequest, ...grpc.CallOption) (*aiv1.CreateCredentialResponse, error)
	RevokeCredential(context.Context, *aiv1.RevokeCredentialRequest, ...grpc.CallOption) (*aiv1.RevokeCredentialResponse, error)
}

// AgentClient exposes AI agent listing, model discovery, and creation operations.
type AgentClient interface {
	ListAgents(context.Context, *aiv1.ListAgentsRequest, ...grpc.CallOption) (*aiv1.ListAgentsResponse, error)
	ListProviderModels(context.Context, *aiv1.ListProviderModelsRequest, ...grpc.CallOption) (*aiv1.ListProviderModelsResponse, error)
	CreateAgent(context.Context, *aiv1.CreateAgentRequest, ...grpc.CallOption) (*aiv1.CreateAgentResponse, error)
}

// GRPCGateway maps gRPC settings dependencies into the app-layer gateway contract.
type GRPCGateway struct {
	SocialClient     SocialClient
	AccountClient    AccountClient
	PasskeyClient    PasskeyClient
	CredentialClient CredentialClient
	AgentClient      AgentClient
}

// NewGRPCGateway builds the production settings gateway from the configured clients.
// Surface health is derived per settings area so account outages do not hide AI
// settings, and AI outages do not hide account settings.
func NewGRPCGateway(socialClient SocialClient, accountClient AccountClient, passkeyClient PasskeyClient, credentialClient CredentialClient, agentClient AgentClient) settingsapp.Gateway {
	if socialClient == nil && accountClient == nil && passkeyClient == nil && credentialClient == nil && agentClient == nil {
		return settingsapp.NewUnavailableGateway()
	}
	return GRPCGateway{
		SocialClient:     socialClient,
		AccountClient:    accountClient,
		PasskeyClient:    passkeyClient,
		CredentialClient: credentialClient,
		AgentClient:      agentClient,
	}
}

// ProfileGatewayHealthy reports whether the profile surface can serve requests.
func (g GRPCGateway) ProfileGatewayHealthy() bool {
	return g.SocialClient != nil
}

// LocaleGatewayHealthy reports whether the locale surface can serve requests.
func (g GRPCGateway) LocaleGatewayHealthy() bool {
	return g.AccountClient != nil
}

// SecurityGatewayHealthy reports whether the security surface can serve requests.
func (g GRPCGateway) SecurityGatewayHealthy() bool {
	return g.PasskeyClient != nil
}

// AIKeyGatewayHealthy reports whether the AI keys surface can serve requests.
func (g GRPCGateway) AIKeyGatewayHealthy() bool {
	return g.CredentialClient != nil
}

// AIAgentGatewayHealthy reports whether the AI agents surface can serve requests.
func (g GRPCGateway) AIAgentGatewayHealthy() bool {
	return g.CredentialClient != nil && g.AgentClient != nil
}
