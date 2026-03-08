package gateway

import (
	"context"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	settingsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// SocialClient exposes profile lookup and mutation operations.
type SocialClient interface {
	GetUserProfile(context.Context, *socialv1.GetUserProfileRequest, ...grpc.CallOption) (*socialv1.GetUserProfileResponse, error)
	LookupUserProfile(context.Context, *socialv1.LookupUserProfileRequest, ...grpc.CallOption) (*socialv1.LookupUserProfileResponse, error)
	SetUserProfile(context.Context, *socialv1.SetUserProfileRequest, ...grpc.CallOption) (*socialv1.SetUserProfileResponse, error)
}

// AccountClient exposes account profile read/update operations.
type AccountClient interface {
	GetProfile(context.Context, *authv1.GetProfileRequest, ...grpc.CallOption) (*authv1.GetProfileResponse, error)
	UpdateProfile(context.Context, *authv1.UpdateProfileRequest, ...grpc.CallOption) (*authv1.UpdateProfileResponse, error)
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
	CredentialClient CredentialClient
	AgentClient      AgentClient
}

// NewGRPCGateway builds the production settings gateway from the required clients.
// All four clients are required — a partial set would report healthy while
// individual settings pages 503.
func NewGRPCGateway(socialClient SocialClient, accountClient AccountClient, credentialClient CredentialClient, agentClient AgentClient) settingsapp.Gateway {
	if socialClient == nil || accountClient == nil || credentialClient == nil || agentClient == nil {
		return settingsapp.NewUnavailableGateway()
	}
	return GRPCGateway{
		SocialClient:     socialClient,
		AccountClient:    accountClient,
		CredentialClient: credentialClient,
		AgentClient:      agentClient,
	}
}

// LoadProfile loads the package state needed for this request path.
func (g GRPCGateway) LoadProfile(ctx context.Context, userID string) (settingsapp.SettingsProfile, error) {
	if g.SocialClient == nil {
		return settingsapp.SettingsProfile{}, apperrors.EK(apperrors.KindUnavailable, "error.web.message.social_service_is_not_configured", "social service client is not configured")
	}
	resp, err := g.SocialClient.GetUserProfile(ctx, &socialv1.GetUserProfileRequest{UserId: userID})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return settingsapp.SettingsProfile{}, nil
		}
		return settingsapp.SettingsProfile{}, err
	}
	if resp == nil || resp.GetUserProfile() == nil {
		return settingsapp.SettingsProfile{}, nil
	}
	profile := resp.GetUserProfile()
	return settingsapp.SettingsProfile{
		Username:      strings.TrimSpace(profile.GetUsername()),
		Name:          strings.TrimSpace(profile.GetName()),
		Pronouns:      pronouns.FromProto(profile.GetPronouns()),
		Bio:           strings.TrimSpace(profile.GetBio()),
		AvatarSetID:   strings.TrimSpace(profile.GetAvatarSetId()),
		AvatarAssetID: strings.TrimSpace(profile.GetAvatarAssetId()),
	}, nil
}

// SaveProfile centralizes this web behavior in one helper seam.
func (g GRPCGateway) SaveProfile(ctx context.Context, userID string, profile settingsapp.SettingsProfile) error {
	if g.SocialClient == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.social_service_is_not_configured", "social service client is not configured")
	}
	_, err := g.SocialClient.SetUserProfile(ctx, &socialv1.SetUserProfileRequest{
		UserId:        userID,
		Username:      profile.Username,
		Name:          profile.Name,
		Pronouns:      pronouns.ToProto(profile.Pronouns),
		Bio:           profile.Bio,
		AvatarSetId:   profile.AvatarSetID,
		AvatarAssetId: profile.AvatarAssetID,
	})
	return err
}

// LoadLocale loads the package state needed for this request path.
func (g GRPCGateway) LoadLocale(ctx context.Context, userID string) (string, error) {
	if g.AccountClient == nil {
		return "", apperrors.EK(apperrors.KindUnavailable, "error.web.message.account_service_client_is_not_configured", "account service client is not configured")
	}
	resp, err := g.AccountClient.GetProfile(ctx, &authv1.GetProfileRequest{UserId: userID})
	if err != nil {
		return "", err
	}
	if resp == nil || resp.GetProfile() == nil {
		return settingsapp.NormalizeLocale(""), nil
	}
	return mapSettingsLocaleFromProto(resp.GetProfile().GetLocale()), nil
}

// SaveLocale centralizes this web behavior in one helper seam.
func (g GRPCGateway) SaveLocale(ctx context.Context, userID string, locale string) error {
	if g.AccountClient == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.account_service_client_is_not_configured", "account service client is not configured")
	}
	_, err := g.AccountClient.UpdateProfile(ctx, &authv1.UpdateProfileRequest{UserId: userID, Locale: mapSettingsLocaleToProto(locale)})
	return err
}

// ListAIKeys returns the package view collection for this workflow.
func (g GRPCGateway) ListAIKeys(ctx context.Context, userID string) ([]settingsapp.SettingsAIKey, error) {
	if g.CredentialClient == nil {
		return nil, apperrors.EK(apperrors.KindUnavailable, "error.web.message.credential_service_client_is_not_configured", "credential service client is not configured")
	}
	resp, err := g.CredentialClient.ListCredentials(ctx, &aiv1.ListCredentialsRequest{PageSize: 50})
	if err != nil {
		return nil, err
	}
	keys := make([]settingsapp.SettingsAIKey, 0, len(resp.GetCredentials()))
	for _, credential := range resp.GetCredentials() {
		if credential == nil {
			continue
		}
		credentialID := strings.TrimSpace(credential.GetId())
		statusValue := credential.GetStatus()
		safeCredentialID := credentialID
		canRevoke := credentialID != "" && statusValue == aiv1.CredentialStatus_CREDENTIAL_STATUS_ACTIVE
		if !isSafeCredentialPathID(credentialID) {
			safeCredentialID = ""
			canRevoke = false
		}
		keys = append(keys, settingsapp.SettingsAIKey{
			ID:        safeCredentialID,
			Label:     strings.TrimSpace(credential.GetLabel()),
			Provider:  providerDisplayLabel(credential.GetProvider()),
			Status:    credentialStatusDisplayLabel(statusValue),
			CreatedAt: formatProtoTimestamp(credential.GetCreatedAt()),
			RevokedAt: formatProtoTimestamp(credential.GetRevokedAt()),
			CanRevoke: canRevoke,
		})
	}
	return keys, nil
}

// ListAIAgentCredentials returns active credential options for agent creation.
func (g GRPCGateway) ListAIAgentCredentials(ctx context.Context, userID string) ([]settingsapp.SettingsAICredentialOption, error) {
	keys, err := g.ListAIKeys(ctx, userID)
	if err != nil {
		return nil, err
	}
	options := make([]settingsapp.SettingsAICredentialOption, 0, len(keys))
	for _, key := range keys {
		if !key.CanRevoke {
			continue
		}
		options = append(options, settingsapp.SettingsAICredentialOption{
			ID:       key.ID,
			Label:    key.Label,
			Provider: key.Provider,
		})
	}
	return options, nil
}

// ListAIAgents returns the package view collection for the AI agents page.
func (g GRPCGateway) ListAIAgents(ctx context.Context, userID string) ([]settingsapp.SettingsAIAgent, error) {
	if g.AgentClient == nil {
		return nil, apperrors.EK(apperrors.KindUnavailable, "error.web.message.ai_agent_service_client_is_not_configured", "AI agent service client is not configured")
	}
	agents := make([]settingsapp.SettingsAIAgent, 0, 50)
	pageToken := ""
	for {
		resp, err := g.AgentClient.ListAgents(ctx, &aiv1.ListAgentsRequest{
			PageSize:  50,
			PageToken: pageToken,
		})
		if err != nil {
			return nil, err
		}
		for _, agent := range resp.GetAgents() {
			if agent == nil {
				continue
			}
			agents = append(agents, settingsapp.SettingsAIAgent{
				ID:           strings.TrimSpace(agent.GetId()),
				Name:         strings.TrimSpace(agent.GetName()),
				Provider:     providerDisplayLabel(agent.GetProvider()),
				Model:        strings.TrimSpace(agent.GetModel()),
				Status:       agentStatusDisplayLabel(agent.GetStatus()),
				CreatedAt:    formatProtoTimestamp(agent.GetCreatedAt()),
				Instructions: strings.TrimSpace(agent.GetInstructions()),
			})
		}
		pageToken = strings.TrimSpace(resp.GetNextPageToken())
		if pageToken == "" {
			break
		}
	}
	return agents, nil
}

// ListAIProviderModels returns provider-backed model options for one credential.
func (g GRPCGateway) ListAIProviderModels(ctx context.Context, userID string, credentialID string) ([]settingsapp.SettingsAIModelOption, error) {
	if g.AgentClient == nil {
		return nil, apperrors.EK(apperrors.KindUnavailable, "error.web.message.ai_agent_service_client_is_not_configured", "AI agent service client is not configured")
	}
	resp, err := g.AgentClient.ListProviderModels(ctx, &aiv1.ListProviderModelsRequest{
		Provider:     aiv1.Provider_PROVIDER_OPENAI,
		CredentialId: credentialID,
	})
	if err != nil {
		return nil, err
	}
	models := make([]settingsapp.SettingsAIModelOption, 0, len(resp.GetModels()))
	for _, model := range resp.GetModels() {
		if model == nil {
			continue
		}
		models = append(models, settingsapp.SettingsAIModelOption{
			ID:      strings.TrimSpace(model.GetId()),
			OwnedBy: strings.TrimSpace(model.GetOwnedBy()),
		})
	}
	return models, nil
}

// CreateAIKey executes package-scoped creation behavior for this flow.
func (g GRPCGateway) CreateAIKey(ctx context.Context, userID string, label string, secret string) error {
	if g.CredentialClient == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.credential_service_client_is_not_configured", "credential service client is not configured")
	}
	_, err := g.CredentialClient.CreateCredential(ctx, &aiv1.CreateCredentialRequest{
		Provider: aiv1.Provider_PROVIDER_OPENAI,
		Label:    label,
		Secret:   secret,
	})
	return err
}

// CreateAIAgent executes package-scoped creation behavior for this flow.
func (g GRPCGateway) CreateAIAgent(ctx context.Context, userID string, input settingsapp.CreateAIAgentInput) error {
	if g.AgentClient == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.ai_agent_service_client_is_not_configured", "AI agent service client is not configured")
	}
	_, err := g.AgentClient.CreateAgent(ctx, &aiv1.CreateAgentRequest{
		Name:         input.Name,
		Provider:     aiv1.Provider_PROVIDER_OPENAI,
		Model:        input.Model,
		CredentialId: input.CredentialID,
		Instructions: input.Instructions,
	})
	return err
}

// RevokeAIKey applies this package workflow transition.
func (g GRPCGateway) RevokeAIKey(ctx context.Context, userID string, credentialID string) error {
	if g.CredentialClient == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.credential_service_client_is_not_configured", "credential service client is not configured")
	}
	_, err := g.CredentialClient.RevokeCredential(ctx, &aiv1.RevokeCredentialRequest{CredentialId: credentialID})
	return err
}

// providerDisplayLabel centralizes this web behavior in one helper seam.
func providerDisplayLabel(provider aiv1.Provider) string {
	switch provider {
	case aiv1.Provider_PROVIDER_OPENAI:
		return "OpenAI"
	default:
		return "Unknown"
	}
}

// credentialStatusDisplayLabel centralizes this web behavior in one helper seam.
func credentialStatusDisplayLabel(statusValue aiv1.CredentialStatus) string {
	switch statusValue {
	case aiv1.CredentialStatus_CREDENTIAL_STATUS_ACTIVE:
		return "Active"
	case aiv1.CredentialStatus_CREDENTIAL_STATUS_REVOKED:
		return "Revoked"
	default:
		return "Unspecified"
	}
}

// agentStatusDisplayLabel centralizes this web behavior in one helper seam.
func agentStatusDisplayLabel(statusValue aiv1.AgentStatus) string {
	switch statusValue {
	case aiv1.AgentStatus_AGENT_STATUS_ACTIVE:
		return "Active"
	default:
		return "Unspecified"
	}
}

// formatProtoTimestamp centralizes this web behavior in one helper seam.
func formatProtoTimestamp(value *timestamppb.Timestamp) string {
	if value == nil {
		return "-"
	}
	if err := value.CheckValid(); err != nil {
		return "-"
	}
	return value.AsTime().UTC().Format("2006-01-02 15:04 UTC")
}

// mapSettingsLocaleToProto maps values across transport and domain boundaries.
func mapSettingsLocaleToProto(locale string) commonv1.Locale {
	s := settingsapp.NormalizeLocale(locale)
	switch s {
	case "pt-BR":
		return commonv1.Locale_LOCALE_PT_BR
	default:
		return commonv1.Locale_LOCALE_EN_US
	}
}

// mapSettingsLocaleFromProto maps values across transport and domain boundaries.
func mapSettingsLocaleFromProto(locale commonv1.Locale) string {
	switch locale {
	case commonv1.Locale_LOCALE_PT_BR:
		return "pt-BR"
	default:
		return "en-US"
	}
}

// isSafeCredentialPathID reports whether this package condition is satisfied.
func isSafeCredentialPathID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	return !strings.Contains(value, "/") && !strings.Contains(value, "\\")
}
