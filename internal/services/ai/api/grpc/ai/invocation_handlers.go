package ai

import (
	"context"
	"errors"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	aiprovider "github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// InvokeAgent executes one provider call using an owned active agent auth reference.
func (h *InvocationHandlers) InvokeAgent(ctx context.Context, in *aiv1.InvokeAgentRequest) (*aiv1.InvokeAgentResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "invoke agent request is required")
	}
	if h.agentStore == nil {
		return nil, status.Error(codes.Internal, "agent store is not configured")
	}
	if h.sealer == nil {
		return nil, status.Error(codes.Internal, "secret sealer is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}
	agentID := strings.TrimSpace(in.GetAgentId())
	if agentID == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id is required")
	}
	input := strings.TrimSpace(in.GetInput())
	if input == "" {
		return nil, status.Error(codes.InvalidArgument, "input is required")
	}

	agentRecord, err := h.agentStore.GetAgent(ctx, agentID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "agent not found")
		}
		return nil, status.Errorf(codes.Internal, "get agent: %v", err)
	}

	authorized, sharedAccess, accessRequestID, err := newAccessibleAgentResolver(h.agentStore, h.accessRequestStore).isAuthorizedToInvokeAgent(ctx, userID, agentRecord)
	if err != nil {
		return nil, err
	}
	if !authorized {
		return nil, status.Error(codes.NotFound, "agent not found")
	}

	providerID := providerFromString(agentRecord.Provider)
	adapter, ok := h.providerInvocationAdapters[providerID]
	if !ok || adapter == nil {
		return nil, status.Error(codes.FailedPrecondition, "provider invocation adapter is unavailable")
	}

	invokeToken, err := h.resolveAgentInvokeToken(ctx, strings.TrimSpace(agentRecord.OwnerUserID), agentRecord)
	if err != nil {
		return nil, err
	}
	result, err := adapter.Invoke(ctx, aiprovider.InvokeInput{
		Model:            agentRecord.Model,
		Input:            input,
		Instructions:     strings.TrimSpace(agentRecord.Instructions),
		ReasoningEffort:  strings.TrimSpace(in.GetReasoningEffort()),
		CredentialSecret: invokeToken,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "invoke provider: %v", err)
	}
	outputText := strings.TrimSpace(result.OutputText)
	if outputText == "" {
		return nil, status.Error(codes.Internal, "provider returned empty output")
	}
	if sharedAccess {
		if err := putAuditEvent(ctx, h.auditEventStore, storage.AuditEventRecord{
			EventName:       "agent.invoke.shared",
			ActorUserID:     userID,
			OwnerUserID:     strings.TrimSpace(agentRecord.OwnerUserID),
			RequesterUserID: userID,
			AgentID:         strings.TrimSpace(agentRecord.ID),
			AccessRequestID: accessRequestID,
			Outcome:         "success",
			CreatedAt:       h.clock().UTC(),
		}); err != nil {
			return nil, status.Errorf(codes.Internal, "put audit event: %v", err)
		}
	}
	return &aiv1.InvokeAgentResponse{
		OutputText: outputText,
		Provider:   providerToProto(agentRecord.Provider),
		Model:      agentRecord.Model,
		Usage:      usageToProto(result.Usage),
	}, nil
}

func (h *InvocationHandlers) resolveAgentInvokeToken(ctx context.Context, ownerUserID string, agentRecord storage.AgentRecord) (string, error) {
	return h.authTokenResolverForRuntime().resolveAgentInvokeToken(ctx, ownerUserID, agentRecord)
}
