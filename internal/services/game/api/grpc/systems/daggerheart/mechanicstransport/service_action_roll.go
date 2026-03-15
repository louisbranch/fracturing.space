package mechanicstransport

import (
	"context"
	"errors"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/random"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ActionRoll handles deterministic action roll requests against the mechanics
// domain model while surfacing seed metadata to callers.
func (h *Handler) ActionRoll(ctx context.Context, in *pb.ActionRollRequest) (*pb.ActionRollResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "action roll request is required")
	}
	if h == nil || h.seedFunc == nil {
		return nil, status.Error(codes.Internal, "seed generator is not configured")
	}

	seed, seedSource, rollMode, err := random.ResolveSeed(
		in.GetRng(),
		h.seedFunc,
		func(mode commonv1.RollMode) bool {
			return mode == commonv1.RollMode_REPLAY
		},
	)
	if err != nil {
		if errors.Is(err, random.ErrSeedOutOfRange()) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, grpcerror.Internal("failed to generate seed", err)
	}

	var difficulty *int
	if in.Difficulty != nil {
		value := int(*in.Difficulty)
		difficulty = &value
	}

	result, err := daggerheartdomain.RollAction(daggerheartdomain.ActionRequest{
		Modifier:     int(in.GetModifier()),
		Difficulty:   difficulty,
		Seed:         seed,
		Advantage:    int(in.GetAdvantage()),
		Disadvantage: int(in.GetDisadvantage()),
	})
	if err != nil {
		if errors.Is(err, daggerheartdomain.ErrInvalidDifficulty) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, grpcerror.Internal("failed to roll action", err)
	}

	response := &pb.ActionRollResponse{
		Hope:              int32(result.Hope),
		Fear:              int32(result.Fear),
		Modifier:          int32(result.Modifier),
		AdvantageDie:      int32(result.AdvantageDie),
		AdvantageModifier: int32(result.AdvantageModifier),
		Total:             int32(result.Total),
		IsCrit:            result.IsCrit,
		MeetsDifficulty:   result.MeetsDifficulty,
		Outcome:           outcomeToProto(result.Outcome),
		Rng: &commonv1.RngResponse{
			SeedUsed:   uint64(seed),
			RngAlgo:    random.RngAlgoMathRandV1,
			SeedSource: seedSource,
			RollMode:   rollMode,
		},
	}
	if result.Difficulty != nil {
		value := int32(*result.Difficulty)
		response.Difficulty = &value
	}

	return response, nil
}
