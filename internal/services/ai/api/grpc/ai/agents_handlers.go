package ai

import (
	"context"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateAgent creates a user-owned AI agent profile.
func (h *AgentHandlers) CreateAgent(ctx context.Context, in *aiv1.CreateAgentRequest) (*aiv1.CreateAgentResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create agent request is required")
	}
	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}
	providerID, err := providerFromProto(in.GetProvider())
	if err != nil {
		return nil, err
	}

	record, err := h.svc.Create(ctx, service.CreateAgentInput{
		OwnerUserID:     userID,
		Label:           in.GetLabel(),
		Instructions:    in.GetInstructions(),
		Provider:        providerID,
		Model:           in.GetModel(),
		CredentialID:    in.GetCredentialId(),
		ProviderGrantID: in.GetProviderGrantId(),
	})
	if err != nil {
		return nil, serviceErrorToStatus(err)
	}
	return &aiv1.CreateAgentResponse{Agent: h.agentProtoWithAuthState(ctx, record)}, nil
}

// ListAgents returns a page of agents owned by the caller.
func (h *AgentHandlers) ListAgents(ctx context.Context, in *aiv1.ListAgentsRequest) (*aiv1.ListAgentsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list agents request is required")
	}
	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	page, err := h.svc.List(ctx, userID, clampPageSize(in.GetPageSize()), in.GetPageToken())
	if err != nil {
		return nil, serviceErrorToStatus(err)
	}

	resp := &aiv1.ListAgentsResponse{
		NextPageToken: page.NextPageToken,
		Agents:        make([]*aiv1.Agent, 0, len(page.Agents)),
	}
	for _, rec := range page.Agents {
		proto, err := h.agentProtoWithUsage(ctx, rec)
		if err != nil {
			return nil, serviceErrorToStatus(err)
		}
		resp.Agents = append(resp.Agents, proto)
	}
	return resp, nil
}

// ListProviderModels returns provider-backed model options for one owned auth reference.
func (h *AgentHandlers) ListProviderModels(ctx context.Context, in *aiv1.ListProviderModelsRequest) (*aiv1.ListProviderModelsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list provider models request is required")
	}
	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}
	providerID, err := providerFromProto(in.GetProvider())
	if err != nil {
		return nil, err
	}

	models, err := h.svc.ListProviderModels(ctx, service.ListProviderModelsInput{
		OwnerUserID:     userID,
		Provider:        providerID,
		CredentialID:    in.GetCredentialId(),
		ProviderGrantID: in.GetProviderGrantId(),
	})
	if err != nil {
		return nil, serviceErrorToStatus(err)
	}

	resp := &aiv1.ListProviderModelsResponse{Models: make([]*aiv1.ProviderModel, 0, len(models))}
	for _, model := range models {
		resp.Models = append(resp.Models, &aiv1.ProviderModel{
			Id:      model.ID,
			OwnedBy: model.OwnedBy,
		})
	}
	return resp, nil
}

// ListAccessibleAgents returns a page of agents the caller can invoke, combining
// owned agents with approved shared invoke access.
func (h *AgentHandlers) ListAccessibleAgents(ctx context.Context, in *aiv1.ListAccessibleAgentsRequest) (*aiv1.ListAccessibleAgentsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list accessible agents request is required")
	}
	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	page, err := h.svc.ListAccessible(ctx, userID, clampPageSize(in.GetPageSize()), in.GetPageToken())
	if err != nil {
		return nil, serviceErrorToStatus(err)
	}

	resp := &aiv1.ListAccessibleAgentsResponse{
		NextPageToken: page.NextPageToken,
		Agents:        make([]*aiv1.Agent, 0, len(page.Agents)),
	}
	for _, rec := range page.Agents {
		resp.Agents = append(resp.Agents, h.agentProtoWithAuthState(ctx, rec))
	}
	return resp, nil
}

// GetAccessibleAgent returns one agent by ID when the caller can invoke it.
func (h *AgentHandlers) GetAccessibleAgent(ctx context.Context, in *aiv1.GetAccessibleAgentRequest) (*aiv1.GetAccessibleAgentResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get accessible agent request is required")
	}
	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}
	agentID := strings.TrimSpace(in.GetAgentId())
	if agentID == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id is required")
	}

	record, err := h.svc.GetAccessible(ctx, userID, agentID)
	if err != nil {
		return nil, serviceErrorToStatus(err)
	}
	return &aiv1.GetAccessibleAgentResponse{Agent: h.agentProtoWithAuthState(ctx, record)}, nil
}

// ValidateCampaignAgentBinding verifies owner-scoped bind eligibility for one agent.
func (h *AgentHandlers) ValidateCampaignAgentBinding(ctx context.Context, in *aiv1.ValidateCampaignAgentBindingRequest) (*aiv1.ValidateCampaignAgentBindingResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "validate campaign agent binding request is required")
	}
	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	record, err := h.svc.ValidateCampaignAgentBinding(ctx, userID, strings.TrimSpace(in.GetAgentId()))
	if err != nil {
		return nil, serviceErrorToStatus(err)
	}
	return &aiv1.ValidateCampaignAgentBindingResponse{Agent: h.agentProtoWithAuthState(ctx, record)}, nil
}

// UpdateAgent updates mutable fields on one user-owned agent.
func (h *AgentHandlers) UpdateAgent(ctx context.Context, in *aiv1.UpdateAgentRequest) (*aiv1.UpdateAgentResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "update agent request is required")
	}
	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	record, err := h.svc.Update(ctx, service.UpdateAgentInput{
		OwnerUserID:     userID,
		AgentID:         strings.TrimSpace(in.GetAgentId()),
		Label:           strings.TrimSpace(in.GetLabel()),
		Instructions:    strings.TrimSpace(in.GetInstructions()),
		Model:           strings.TrimSpace(in.GetModel()),
		CredentialID:    strings.TrimSpace(in.GetCredentialId()),
		ProviderGrantID: strings.TrimSpace(in.GetProviderGrantId()),
	})
	if err != nil {
		return nil, serviceErrorToStatus(err)
	}
	return &aiv1.UpdateAgentResponse{Agent: h.agentProtoWithAuthState(ctx, record)}, nil
}

// DeleteAgent deletes one user-owned agent profile.
func (h *AgentHandlers) DeleteAgent(ctx context.Context, in *aiv1.DeleteAgentRequest) (*aiv1.DeleteAgentResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "delete agent request is required")
	}
	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	if err := h.svc.Delete(ctx, userID, strings.TrimSpace(in.GetAgentId())); err != nil {
		return nil, serviceErrorToStatus(err)
	}
	return &aiv1.DeleteAgentResponse{}, nil
}

// agentProtoWithUsage enriches one agent read with auth health and usage metadata.
func (h *AgentHandlers) agentProtoWithUsage(ctx context.Context, a agent.Agent) (*aiv1.Agent, error) {
	proto := agentToProto(a)
	proto.AuthState = agentAuthStateToProto(h.svc.GetAuthState(ctx, a))

	activeCampaignCount, err := h.svc.GetActiveCampaignCount(ctx, a.ID)
	if err != nil {
		return nil, err
	}
	proto.ActiveCampaignCount = activeCampaignCount
	return proto, nil
}

// agentProtoWithAuthState enriches one agent read with auth health only.
func (h *AgentHandlers) agentProtoWithAuthState(ctx context.Context, a agent.Agent) *aiv1.Agent {
	proto := agentToProto(a)
	proto.AuthState = agentAuthStateToProto(h.svc.GetAuthState(ctx, a))
	return proto
}

// agentAuthStateToProto maps a service-layer agent auth state to the proto enum.
func agentAuthStateToProto(state service.AgentAuthState) aiv1.AgentAuthState {
	switch state {
	case service.AgentAuthStateReady:
		return aiv1.AgentAuthState_AGENT_AUTH_STATE_READY
	case service.AgentAuthStateRevoked:
		return aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_REVOKED
	case service.AgentAuthStateUnavailable:
		return aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_UNAVAILABLE
	default:
		return aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_UNAVAILABLE
	}
}
