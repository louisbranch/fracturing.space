package sessionrolltransport

import (
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/dice"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/domain"
)

// outcomeToProto maps a domain Outcome to the corresponding proto enum value.
func outcomeToProto(outcome daggerheartdomain.Outcome) pb.Outcome {
	switch outcome {
	case daggerheartdomain.OutcomeRollWithHope:
		return pb.Outcome_ROLL_WITH_HOPE
	case daggerheartdomain.OutcomeRollWithFear:
		return pb.Outcome_ROLL_WITH_FEAR
	case daggerheartdomain.OutcomeSuccessWithHope:
		return pb.Outcome_SUCCESS_WITH_HOPE
	case daggerheartdomain.OutcomeSuccessWithFear:
		return pb.Outcome_SUCCESS_WITH_FEAR
	case daggerheartdomain.OutcomeFailureWithHope:
		return pb.Outcome_FAILURE_WITH_HOPE
	case daggerheartdomain.OutcomeFailureWithFear:
		return pb.Outcome_FAILURE_WITH_FEAR
	case daggerheartdomain.OutcomeCriticalSuccess:
		return pb.Outcome_CRITICAL_SUCCESS
	default:
		return pb.Outcome_OUTCOME_UNSPECIFIED
	}
}

// damageDiceFromProto converts proto DiceSpec messages into the domain
// DamageDieSpec slice consumed by RollDamage.
func damageDiceFromProto(specs []*pb.DiceSpec) ([]bridge.DamageDieSpec, error) {
	if len(specs) == 0 {
		return nil, dice.ErrMissingDice
	}
	out := make([]bridge.DamageDieSpec, 0, len(specs))
	for _, spec := range specs {
		if spec == nil || spec.GetSides() <= 0 || spec.GetCount() <= 0 {
			return nil, dice.ErrInvalidDiceSpec
		}
		out = append(out, bridge.DamageDieSpec{
			Sides: int(spec.GetSides()),
			Count: int(spec.GetCount()),
		})
	}
	return out, nil
}

// diceRollsToProto converts domain dice Roll results into proto DiceRoll
// messages for the response.
func diceRollsToProto(rolls []dice.Roll) []*pb.DiceRoll {
	if len(rolls) == 0 {
		return nil
	}
	out := make([]*pb.DiceRoll, 0, len(rolls))
	for _, roll := range rolls {
		results := make([]int32, 0, len(roll.Results))
		for _, r := range roll.Results {
			results = append(results, int32(r))
		}
		out = append(out, &pb.DiceRoll{
			Sides:   int32(roll.Sides),
			Results: results,
			Total:   int32(roll.Total),
		})
	}
	return out
}
