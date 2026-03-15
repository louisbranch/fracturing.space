package sessionrolltransport

import (
	"context"
	"errors"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/dice"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/random"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) sessionAdversaryActionCheck(ctx context.Context, in *pb.SessionAdversaryActionCheckRequest) (*pb.SessionAdversaryActionCheckResponse, error) {
	if err := h.requireAdversaryActionCheckDependencies(); err != nil {
		return nil, err
	}

	rollCtx, err := h.loadAdversaryRollContext(
		ctx,
		in.GetCampaignId(),
		in.GetSessionId(),
		in.GetSceneId(),
		in.GetAdversaryId(),
		"campaign system does not support daggerheart rolls",
	)
	if err != nil {
		return nil, err
	}
	if in.GetDifficulty() < 0 {
		return nil, status.Error(codes.InvalidArgument, "difficulty must be non-negative")
	}

	rollSeq, err := h.nextAdversaryRollSeq(ctx, rollCtx.CampaignID)
	if err != nil {
		return nil, err
	}

	if !in.GetDramatic() {
		return &pb.SessionAdversaryActionCheckResponse{
			RollSeq:     rollSeq,
			AutoSuccess: true,
			Success:     true,
			Roll:        0,
			Modifier:    in.GetModifier(),
			Total:       in.GetModifier(),
		}, nil
	}

	seed, seedSource, rollMode, err := random.ResolveSeed(
		in.GetRng(),
		h.deps.SeedFunc,
		func(mode commonv1.RollMode) bool { return mode == commonv1.RollMode_REPLAY },
	)
	if err != nil {
		if errors.Is(err, random.ErrSeedOutOfRange()) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, grpcerror.Internal("failed to resolve seed", err)
	}

	rollResult, err := resolveD20Roll(seed)
	if err != nil {
		return nil, grpcerror.Internal("failed to resolve dramatic action check", err)
	}

	total := rollResult + int(in.GetModifier())
	difficulty := int(in.GetDifficulty())

	return &pb.SessionAdversaryActionCheckResponse{
		RollSeq:  rollSeq,
		Success:  total >= difficulty,
		Roll:     int32(rollResult),
		Modifier: in.GetModifier(),
		Total:    int32(total),
		Rng: &commonv1.RngResponse{
			SeedUsed:   uint64(seed),
			RngAlgo:    random.RngAlgoMathRandV1,
			SeedSource: seedSource,
			RollMode:   rollMode,
		},
	}, nil
}

func resolveD20Roll(seed int64) (int, error) {
	result, err := dice.RollDice(dice.Request{
		Dice: []dice.Spec{{Sides: 20, Count: 1}},
		Seed: seed,
	})
	if err != nil {
		return 0, err
	}
	if len(result.Rolls) != 1 || len(result.Rolls[0].Results) != 1 {
		return 0, status.Error(codes.Internal, "invalid d20 roll result")
	}

	return result.Rolls[0].Results[0], nil
}
