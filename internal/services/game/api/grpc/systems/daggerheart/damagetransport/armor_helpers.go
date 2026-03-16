package damagetransport

import (
	"errors"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/dice"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/random"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func rollArmorFeatureDie(seedFunc func() (int64, error), rng *commonv1.RngRequest, sides int) (int, error) {
	if sides <= 0 {
		return 0, status.Error(codes.InvalidArgument, "armor feature die must be positive")
	}
	seed, _, _, err := random.ResolveSeed(
		rng,
		seedFunc,
		func(mode commonv1.RollMode) bool { return mode == commonv1.RollMode_REPLAY },
	)
	if err != nil {
		if errors.Is(err, random.ErrSeedOutOfRange()) {
			return 0, status.Error(codes.InvalidArgument, err.Error())
		}
		return 0, status.Error(codes.Internal, "failed to resolve armor feature seed")
	}
	result, err := dice.RollDice(dice.Request{
		Dice: []dice.Spec{{Sides: sides, Count: 1}},
		Seed: seed,
	})
	if err != nil {
		return 0, status.Error(codes.Internal, "failed to roll armor feature die")
	}
	return result.Total, nil
}
