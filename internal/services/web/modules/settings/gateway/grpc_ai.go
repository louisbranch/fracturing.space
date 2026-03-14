package gateway

import (
	"context"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	settingsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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
				Label:        strings.TrimSpace(agent.GetLabel()),
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
	return mapAIKeyMutationError(err)
}

// CreateAIAgent executes package-scoped creation behavior for this flow.
func (g GRPCGateway) CreateAIAgent(ctx context.Context, userID string, input settingsapp.CreateAIAgentInput) error {
	if g.AgentClient == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.ai_agent_service_client_is_not_configured", "AI agent service client is not configured")
	}
	_, err := g.AgentClient.CreateAgent(ctx, &aiv1.CreateAgentRequest{
		Label:        input.Label,
		Provider:     aiv1.Provider_PROVIDER_OPENAI,
		Model:        input.Model,
		CredentialId: input.CredentialID,
		Instructions: input.Instructions,
	})
	return mapAIAgentMutationError(err)
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

// isSafeCredentialPathID reports whether this package condition is satisfied.
func isSafeCredentialPathID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	return !strings.Contains(value, "/") && !strings.Contains(value, "\\")
}

// mapAIKeyMutationError converts transport-level key mutation conflicts into web errors.
func mapAIKeyMutationError(err error) error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if ok && st.Code() == codes.AlreadyExists {
		return apperrors.EK(apperrors.KindConflict, "web.settings.ai_keys.error_duplicate_label", "AI key label already exists")
	}
	return err
}

// mapAIAgentMutationError converts transport-level agent mutation conflicts into web errors.
func mapAIAgentMutationError(err error) error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if ok && st.Code() == codes.AlreadyExists {
		return apperrors.EK(apperrors.KindConflict, "web.settings.ai_agents.error_duplicate_label", "AI agent label already exists")
	}
	return err
}
