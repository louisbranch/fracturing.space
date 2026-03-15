package sessionrolltransport

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) SessionActionRoll(ctx context.Context, in *pb.SessionActionRollRequest) (*pb.SessionActionRollResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session action roll request is required")
	}

	actionRoll, err := h.loadSessionActionRollContext(ctx, in)
	if err != nil {
		return nil, err
	}

	latestSeq, err := h.deps.Event.GetLatestEventSeq(ctx, actionRoll.CampaignID)
	if err != nil {
		return nil, err
	}
	rollSeq := latestSeq + uint64(actionRoll.SpendEventCount) + 1

	seed, seedSource, rollMode, err := h.resolveActionRollSeed(in.GetRng())
	if err != nil {
		return nil, err
	}

	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	if err := h.applyActionRollHopeSpends(ctx, actionRoll, rollSeq, requestID, invocationID); err != nil {
		return nil, err
	}

	result, generateHopeFear, triggerGMMove, critNegatesEffects, err := h.resolveActionRoll(seed, actionRoll)
	if err != nil {
		return nil, err
	}

	payloadJSON, flavor, err := h.buildSessionActionRollPayload(
		actionRoll,
		result,
		requestID,
		uint64(seed),
		seedSource,
		rollMode,
		rollSeq,
		generateHopeFear,
		triggerGMMove,
		critNegatesEffects,
	)
	if err != nil {
		return nil, err
	}

	rollSeqValue, err := h.deps.ExecuteActionRollResolve(ctx, RollResolveInput{
		CampaignID:      actionRoll.CampaignID,
		SessionID:       actionRoll.SessionID,
		SceneID:         actionRoll.SceneID,
		RequestID:       requestID,
		InvocationID:    invocationID,
		EntityType:      "roll",
		EntityID:        requestID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "action roll did not emit an event",
	})
	if err != nil {
		return nil, err
	}

	failed := result.Difficulty != nil && !result.MeetsDifficulty
	if err := h.deps.AdvanceBreathCountdown(ctx, actionRoll.CampaignID, actionRoll.SessionID, actionRoll.BreathCountdownID, failed); err != nil {
		return nil, err
	}

	return buildSessionActionRollResponse(
		rollSeqValue,
		result,
		actionRoll.Difficulty,
		flavor,
		uint64(seed),
		seedSource,
		rollMode,
	), nil
}
