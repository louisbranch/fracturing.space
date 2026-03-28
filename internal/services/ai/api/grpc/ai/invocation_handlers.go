package ai

import (
	"context"
	"fmt"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/service"
)

// InvocationHandlers serves invocation RPCs as thin transport wrappers over
// the invocation service.
type InvocationHandlers struct {
	aiv1.UnimplementedInvocationServiceServer
	svc *service.InvocationService
}

// InvocationHandlersConfig declares the dependencies for invocation RPCs.
type InvocationHandlersConfig struct {
	InvocationService *service.InvocationService
}

// NewInvocationHandlers builds an invocation RPC server from a service.
func NewInvocationHandlers(cfg InvocationHandlersConfig) (*InvocationHandlers, error) {
	if cfg.InvocationService == nil {
		return nil, fmt.Errorf("ai: NewInvocationHandlers: invocation service is required")
	}
	return &InvocationHandlers{svc: cfg.InvocationService}, nil
}

// InvokeAgent executes one provider call using an owned active agent auth reference.
func (h *InvocationHandlers) InvokeAgent(ctx context.Context, in *aiv1.InvokeAgentRequest) (*aiv1.InvokeAgentResponse, error) {
	userID, err := requireUserScopedUnaryRequest(ctx, in, "invoke agent request is required")
	if err != nil {
		return nil, err
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
