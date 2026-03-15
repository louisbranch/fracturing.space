package mechanicstransport

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

// RollDice handles generic deterministic dice requests through the shared seed
// source and returns both roll results and RNG metadata.
func (h *Handler) RollDice(ctx context.Context, in *pb.RollDiceRequest) (*pb.RollDiceResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "dice roll request is required")
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

	request := dice.Request{
		Dice: make([]dice.Spec, 0, len(in.GetDice())),
		Seed: seed,
	}
	for _, spec := range in.GetDice() {
		request.Dice = append(request.Dice, dice.Spec{
			Sides: int(spec.GetSides()),
			Count: int(spec.GetCount()),
		})
	}

	result, err := dice.RollDice(request)
	if err != nil {
		if errors.Is(err, dice.ErrMissingDice) || errors.Is(err, dice.ErrInvalidDiceSpec) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, grpcerror.Internal("failed to roll dice", err)
	}

	response := &pb.RollDiceResponse{
		Rolls: make([]*pb.DiceRoll, 0, len(result.Rolls)),
		Total: int32(result.Total),
		Rng: &commonv1.RngResponse{
			SeedUsed:   uint64(seed),
			RngAlgo:    random.RngAlgoMathRandV1,
			SeedSource: seedSource,
			RollMode:   rollMode,
		},
	}
	for _, roll := range result.Rolls {
		response.Rolls = append(response.Rolls, &pb.DiceRoll{
			Sides:   int32(roll.Sides),
			Results: int32Slice(roll.Results),
			Total:   int32(roll.Total),
		})
	}

	return response, nil
}
