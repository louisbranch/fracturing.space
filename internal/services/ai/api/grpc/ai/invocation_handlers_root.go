package ai

import (
	"fmt"

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
