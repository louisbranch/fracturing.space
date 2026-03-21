package ai

import (
	"context"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// InvokeAgent executes one provider call using an owned active agent auth reference.
func (h *InvocationHandlers) InvokeAgent(ctx context.Context, in *aiv1.InvokeAgentRequest) (*aiv1.InvokeAgentResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "invoke agent request is required")
	}
	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	result, err := h.svc.InvokeAgent(ctx, service.InvokeAgentInput{
		CallerUserID:    userID,
		AgentID:         strings.TrimSpace(in.GetAgentId()),
		Input:           strings.TrimSpace(in.GetInput()),
		ReasoningEffort: strings.TrimSpace(in.GetReasoningEffort()),
	})
	if err != nil {
		return nil, serviceErrorToStatus(err)
	}
	return &aiv1.InvokeAgentResponse{
		OutputText: result.OutputText,
		Provider:   providerToProto(string(result.Provider)),
		Model:      result.Model,
		Usage:      usageToProto(result.Usage),
	}, nil
}
