package ai

import (
	"fmt"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/service"
)

// CampaignOrchestrationHandlers serves campaign-turn orchestration RPCs as
// thin transport wrappers over the campaign orchestration service.
type CampaignOrchestrationHandlers struct {
	aiv1.UnimplementedCampaignOrchestrationServiceServer
	svc *service.CampaignOrchestrationService
}

// CampaignOrchestrationHandlersConfig declares the dependencies for campaign
// orchestration RPCs.
type CampaignOrchestrationHandlersConfig struct {
	CampaignOrchestrationService *service.CampaignOrchestrationService
}

// NewCampaignOrchestrationHandlers builds a campaign-orchestration RPC server
// from a service.
func NewCampaignOrchestrationHandlers(cfg CampaignOrchestrationHandlersConfig) (*CampaignOrchestrationHandlers, error) {
	if cfg.CampaignOrchestrationService == nil {
		return nil, fmt.Errorf("ai: NewCampaignOrchestrationHandlers: campaign orchestration service is required")
	}
	return &CampaignOrchestrationHandlers{svc: cfg.CampaignOrchestrationService}, nil
}
