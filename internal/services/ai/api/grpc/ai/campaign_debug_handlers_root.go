package ai

import (
	"fmt"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/service"
)

// CampaignDebugHandlers serves campaign debug read RPCs with campaign auth checks.
type CampaignDebugHandlers struct {
	aiv1.UnimplementedCampaignDebugServiceServer

	svc                      *service.CampaignDebugService
	campaignContextValidator campaignContextValidator
}

// CampaignDebugHandlersConfig declares the dependencies for campaign debug RPCs.
type CampaignDebugHandlersConfig struct {
	CampaignDebugService     *service.CampaignDebugService
	AuthorizationClient      gamev1.AuthorizationServiceClient
	InternalServiceAllowlist map[string]struct{}
}

// NewCampaignDebugHandlers builds a campaign-debug RPC server from the read service.
func NewCampaignDebugHandlers(cfg CampaignDebugHandlersConfig) (*CampaignDebugHandlers, error) {
	if cfg.CampaignDebugService == nil {
		return nil, fmt.Errorf("ai: NewCampaignDebugHandlers: campaign debug service is required")
	}
	return &CampaignDebugHandlers{
		svc:                      cfg.CampaignDebugService,
		campaignContextValidator: newCampaignContextValidator(cfg.AuthorizationClient, cfg.InternalServiceAllowlist),
	}, nil
}
