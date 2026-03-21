package ai

import (
	"fmt"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/service"
)

// AgentHandlers serves agent RPCs as thin transport wrappers over the agent
// service.
type AgentHandlers struct {
	aiv1.UnimplementedAgentServiceServer
	svc *service.AgentService
}

// AgentHandlersConfig declares the dependencies for agent RPCs.
type AgentHandlersConfig struct {
	AgentService *service.AgentService
}

// NewAgentHandlers builds an agent RPC server from a service.
func NewAgentHandlers(cfg AgentHandlersConfig) (*AgentHandlers, error) {
	if cfg.AgentService == nil {
		return nil, fmt.Errorf("ai: NewAgentHandlers: agent service is required")
	}
	return &AgentHandlers{svc: cfg.AgentService}, nil
}
