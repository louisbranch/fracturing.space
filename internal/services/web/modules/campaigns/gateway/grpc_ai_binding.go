package gateway

import (
	"context"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const campaignAIAgentsPageSize = 50

// CampaignAIAgents returns selectable AI agents for owner-only campaign binding.
func (g GRPCGateway) CampaignAIAgents(ctx context.Context) ([]campaignapp.CampaignAIAgentOption, error) {
	if g.Read.Agent == nil {
		return nil, apperrors.EK(apperrors.KindUnavailable, "error.web.message.ai_agent_service_client_is_not_configured", "AI agent service client is not configured")
	}

	resp, err := g.Read.Agent.ListAgents(ctx, &aiv1.ListAgentsRequest{PageSize: campaignAIAgentsPageSize})
	if err != nil {
		return nil, apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnknown,
			FallbackKey:     "error.web.message.failed_to_list_ai_agents",
			FallbackMessage: "failed to list AI agents",
		})
	}

	options := make([]campaignapp.CampaignAIAgentOption, 0, len(resp.GetAgents()))
	for _, agent := range resp.GetAgents() {
		if agent == nil {
			continue
		}
		agentID := strings.TrimSpace(agent.GetId())
		if agentID == "" {
			continue
		}
		options = append(options, campaignapp.CampaignAIAgentOption{
			ID:      agentID,
			Name:    campaignAIAgentDisplayName(agent),
			Enabled: agent.GetStatus() == aiv1.AgentStatus_AGENT_STATUS_ACTIVE,
		})
	}
	return options, nil
}

// UpdateCampaignAIBinding applies this package workflow transition.
func (g GRPCGateway) UpdateCampaignAIBinding(ctx context.Context, campaignID string, input campaignapp.UpdateCampaignAIBindingInput) error {
	if g.Mutation.Campaign == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.campaign_service_client_is_not_configured", "campaign service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}

	aiAgentID := strings.TrimSpace(input.AIAgentID)
	if aiAgentID == "" {
		_, err := g.Mutation.Campaign.ClearCampaignAIBinding(ctx, &statev1.ClearCampaignAIBindingRequest{CampaignId: campaignID})
		return mapCampaignAIBindingMutationError(err)
	}

	_, err := g.Mutation.Campaign.SetCampaignAIBinding(ctx, &statev1.SetCampaignAIBindingRequest{
		CampaignId: campaignID,
		AiAgentId:  aiAgentID,
	})
	return mapCampaignAIBindingMutationError(err)
}

// campaignAIAgentDisplayName resolves a user-facing name with durable fallback.
func campaignAIAgentDisplayName(agent *aiv1.Agent) string {
	if agent == nil {
		return ""
	}
	if name := strings.TrimSpace(agent.GetName()); name != "" {
		return name
	}
	if handle := strings.TrimSpace(agent.GetHandle()); handle != "" {
		return handle
	}
	return strings.TrimSpace(agent.GetId())
}

// mapCampaignAIBindingMutationError keeps invalid input and failed
// precondition responses distinct so the web transport returns 400 vs 409.
func mapCampaignAIBindingMutationError(err error) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if ok {
		switch st.Code() {
		case codes.InvalidArgument, codes.OutOfRange:
			return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.failed_to_update_campaign_ai_binding", "failed to update campaign AI binding")
		case codes.FailedPrecondition, codes.AlreadyExists, codes.Aborted:
			return apperrors.EK(apperrors.KindConflict, "error.web.message.failed_to_update_campaign_ai_binding", "failed to update campaign AI binding")
		case codes.Unauthenticated:
			return apperrors.E(apperrors.KindUnauthorized, "authentication required")
		case codes.PermissionDenied:
			return apperrors.E(apperrors.KindForbidden, "access denied")
		case codes.NotFound:
			return apperrors.E(apperrors.KindNotFound, "resource not found")
		case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted, codes.Canceled:
			return apperrors.E(apperrors.KindUnavailable, "dependency is temporarily unavailable")
		}
	}

	return apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
		FallbackKind:    apperrors.KindUnknown,
		FallbackKey:     "error.web.message.failed_to_update_campaign_ai_binding",
		FallbackMessage: "failed to update campaign AI binding",
	})
}
