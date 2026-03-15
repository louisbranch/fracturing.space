package sessionrolltransport

import (
	"errors"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/random"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) resolveActionRoll(seed int64, actionRoll actionRollContext) (daggerheartdomain.ActionResult, bool, bool, bool, error) {
	result, generateHopeFear, triggerGMMove, critNegatesEffects, err := resolveRoll(actionRoll.RollKind, daggerheartdomain.ActionRequest{
		Modifier:     actionRoll.ModifierTotal,
		Difficulty:   &actionRoll.Difficulty,
		Seed:         seed,
		Advantage:    actionRoll.Advantage,
		Disadvantage: actionRoll.Disadvantage,
	})
	if err != nil {
		if errors.Is(err, daggerheartdomain.ErrInvalidDifficulty) {
			return daggerheartdomain.ActionResult{}, false, false, false, status.Error(codes.InvalidArgument, err.Error())
		}
		return daggerheartdomain.ActionResult{}, false, false, false, grpcerror.Internal("failed to resolve action roll", err)
	}

	return result, generateHopeFear, triggerGMMove, critNegatesEffects, nil
}

func (h *Handler) resolveActionRollSeed(rng *commonv1.RngRequest) (int64, string, commonv1.RollMode, error) {
	seed, seedSource, rollMode, err := random.ResolveSeed(
		rng,
		h.deps.SeedFunc,
		func(mode commonv1.RollMode) bool { return mode == commonv1.RollMode_REPLAY },
	)
	if err != nil {
		if errors.Is(err, random.ErrSeedOutOfRange()) {
			return 0, "", commonv1.RollMode_ROLL_MODE_UNSPECIFIED, status.Error(codes.InvalidArgument, err.Error())
		}
		return 0, "", commonv1.RollMode_ROLL_MODE_UNSPECIFIED, grpcerror.Internal("failed to resolve seed", err)
	}

	return seed, seedSource, rollMode, nil
}
