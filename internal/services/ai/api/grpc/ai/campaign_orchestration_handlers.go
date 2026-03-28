package ai

import (
	"context"
	"fmt"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
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

// RunCampaignTurn validates a game-issued session grant and executes one GM turn.
func (h *CampaignOrchestrationHandlers) RunCampaignTurn(ctx context.Context, in *aiv1.RunCampaignTurnRequest) (*aiv1.RunCampaignTurnResponse, error) {
	if err := requireUnaryRequest(in, "run campaign turn request is required"); err != nil {
		return nil, err
	}

	result, err := h.svc.RunCampaignTurn(ctx, service.RunCampaignTurnInput{
		SessionGrant:    strings.TrimSpace(in.GetSessionGrant()),
		Input:           strings.TrimSpace(in.GetInput()),
		ReasoningEffort: strings.TrimSpace(in.GetReasoningEffort()),
		TurnToken:       strings.TrimSpace(in.GetTurnToken()),
	})
	if err != nil {
		return nil, transportErrorToStatus(err, transportErrorConfig{
			Operation:               "run campaign turn",
			DeadlineExceededCode:    apperrors.CodeAIOrchestrationTimedOut,
			DeadlineExceededMessage: "campaign turn timed out",
			CanceledCode:            apperrors.CodeAIOrchestrationCanceled,
			CanceledMessage:         "campaign turn canceled",
		})
	}
	return &aiv1.RunCampaignTurnResponse{
		OutputText: result.OutputText,
		Provider:   providerToProto(string(result.Provider)),
		Model:      result.Model,
		Usage:      usageToProto(result.Usage),
	}, nil
}
