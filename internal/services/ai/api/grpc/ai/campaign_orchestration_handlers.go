package ai

import (
	"context"
	"errors"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/ai/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RunCampaignTurn validates a game-issued session grant and executes one GM turn.
func (h *CampaignOrchestrationHandlers) RunCampaignTurn(ctx context.Context, in *aiv1.RunCampaignTurnRequest) (*aiv1.RunCampaignTurnResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "run campaign turn request is required")
	}

	result, err := h.svc.RunCampaignTurn(ctx, service.RunCampaignTurnInput{
		SessionGrant:    strings.TrimSpace(in.GetSessionGrant()),
		Input:           strings.TrimSpace(in.GetInput()),
		ReasoningEffort: strings.TrimSpace(in.GetReasoningEffort()),
		TurnToken:       strings.TrimSpace(in.GetTurnToken()),
	})
	if err != nil {
		// Service-layer errors are mapped first; remaining errors go through
		// the orchestration error mapper which handles app-error codes and
		// context errors.
		var svcErr *service.Error
		if errors.As(err, &svcErr) {
			return nil, serviceErrorToStatus(err)
		}
		return nil, campaignTurnGRPCError(err)
	}
	return &aiv1.RunCampaignTurnResponse{
		OutputText: result.OutputText,
		Provider:   providerToProto(string(result.Provider)),
		Model:      result.Model,
		Usage:      usageToProto(result.Usage),
	}, nil
}

// campaignTurnGRPCError converts orchestration errors to gRPC status errors.
// The orchestration layer wraps most context errors with app error codes, but
// raw context errors may still escape (e.g. from transport-level timeouts), so
// we fall back to context detection after the app-error branch.
func campaignTurnGRPCError(err error) error {
	if err == nil {
		return nil
	}
	if apperrors.GetCode(err) != apperrors.CodeUnknown {
		return apperrors.HandleError(err, apperrors.DefaultLocale)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return apperrors.HandleError(
			apperrors.Wrap(apperrors.CodeAIOrchestrationTimedOut, "campaign turn timed out", err),
			apperrors.DefaultLocale,
		)
	}
	if errors.Is(err, context.Canceled) {
		return apperrors.HandleError(
			apperrors.Wrap(apperrors.CodeAIOrchestrationCanceled, "campaign turn canceled", err),
			apperrors.DefaultLocale,
		)
	}
	return status.Errorf(codes.Internal, "run campaign turn: %v", err)
}
