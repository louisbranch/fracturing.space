package outcometransport

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ApplyRollOutcome applies the durable side effects of a previously resolved
// roll and returns the updated projection state.
func (h *Handler) ApplyRollOutcome(ctx context.Context, in *pb.ApplyRollOutcomeRequest) (*pb.ApplyRollOutcomeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply roll outcome request is required")
	}
	if err := h.requireRollOutcomeDependencies(); err != nil {
		return nil, err
	}

	request, err := h.loadApplyRollOutcomeRequest(ctx, in)
	if err != nil {
		return nil, err
	}

	alreadyApplied, err := h.outcomeAlreadyAppliedForSessionRequest(
		ctx,
		request.campaignID,
		request.sessionID,
		request.rollSeq,
		request.rollRequestID,
	)
	if err != nil {
		return nil, grpcerror.Internal("check outcome applied", err)
	}
	if alreadyApplied {
		if request.requiresComplication {
			if err := h.openGMConsequenceGate(
				ctx,
				request.campaignID,
				request.sessionID,
				request.sceneID,
				request.rollSeq,
				request.rollRequestID,
			); err != nil {
				return nil, err
			}
		}
		return h.buildApplyRollOutcomeIdempotentResponse(
			ctx,
			request.campaignID,
			request.rollSeq,
			request.targets,
			request.requiresComplication,
			request.hasGMFearGain(),
		)
	}
	if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.SessionGate, request.campaignID, request.sessionID); err != nil {
		return nil, err
	}

	execution := newApplyRollOutcomeExecution(request)
	if err := h.applyRollOutcomeGMFear(ctx, execution); err != nil {
		return nil, err
	}
	if err := h.applyRollOutcomeCharacterStates(ctx, execution); err != nil {
		return nil, err
	}
	if err := h.applyRollOutcomePostEffects(ctx, execution); err != nil {
		return nil, err
	}
	if err := h.persistApplyRollOutcome(ctx, execution); err != nil {
		return nil, err
	}

	return h.buildApplyRollOutcomeResponse(ctx, execution)
}
