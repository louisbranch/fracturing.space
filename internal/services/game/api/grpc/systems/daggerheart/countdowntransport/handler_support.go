package countdowntransport

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// requireDependencies keeps all countdown mutations on the same store and
// executor contract before request-specific validation begins.
func (h *Handler) requireDependencies() error {
	switch {
	case h.deps.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case h.deps.Session == nil:
		return status.Error(codes.Internal, "session store is not configured")
	case h.deps.SessionGate == nil:
		return status.Error(codes.Internal, "session gate store is not configured")
	case h.deps.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	case h.deps.NewID == nil:
		return status.Error(codes.Internal, "countdown id generator is not configured")
	case h.deps.ExecuteDomainCommand == nil:
		return status.Error(codes.Internal, "domain command executor is not configured")
	default:
		return nil
	}
}

// validateCampaignSession centralizes the campaign, system, session, and gate
// checks shared across countdown write operations.
func (h *Handler) validateCampaignSession(ctx context.Context, campaignID, sessionID string, operation campaign.CampaignOperation, unsupportedMessage string) error {
	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, operation); err != nil {
		return grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(record, unsupportedMessage); err != nil {
		return err
	}
	sess, err := h.deps.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return grpcerror.HandleDomainError(err)
	}
	if sess.Status != session.StatusActive {
		return status.Error(codes.FailedPrecondition, "session is not active")
	}
	return daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID)
}

func (h *Handler) validateCampaignMutate(ctx context.Context, campaignID, unsupportedMessage string) error {
	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return grpcerror.HandleDomainError(err)
	}
	return daggerheartguard.RequireDaggerheartSystem(record, unsupportedMessage)
}

func (h *Handler) validateCampaignSessionRead(ctx context.Context, campaignID, sessionID, unsupportedMessage string) error {
	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpRead); err != nil {
		return grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(record, unsupportedMessage); err != nil {
		return err
	}
	sess, err := h.deps.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return grpcerror.HandleDomainError(err)
	}
	if sess.Status != session.StatusActive {
		return status.Error(codes.FailedPrecondition, "session is not active")
	}
	return nil
}
