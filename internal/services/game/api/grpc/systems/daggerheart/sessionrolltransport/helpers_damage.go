package sessionrolltransport

import (
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/dice"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
)

func damageDiceFromProto(specs []*pb.DiceSpec) ([]bridge.DamageDieSpec, error) {
	if len(specs) == 0 {
		return nil, dice.ErrMissingDice
	}
	converted := make([]bridge.DamageDieSpec, 0, len(specs))
	for _, spec := range specs {
		if spec == nil {
			return nil, dice.ErrInvalidDiceSpec
		}
		sides := int(spec.GetSides())
		count := int(spec.GetCount())
		if sides <= 0 || count <= 0 {
			return nil, dice.ErrInvalidDiceSpec
		}
		converted = append(converted, bridge.DamageDieSpec{Sides: sides, Count: count})
	}
	return converted, nil
}

func diceRollsToProto(rolls []dice.Roll) []*pb.DiceRoll {
	if len(rolls) == 0 {
		return nil
	}
	converted := make([]*pb.DiceRoll, 0, len(rolls))
	for _, roll := range rolls {
		results := make([]int32, 0, len(roll.Results))
		for _, value := range roll.Results {
			results = append(results, int32(value))
		}
		converted = append(converted, &pb.DiceRoll{
			Sides:   int32(roll.Sides),
			Results: results,
			Total:   int32(roll.Total),
		})
	}
	return converted
}
