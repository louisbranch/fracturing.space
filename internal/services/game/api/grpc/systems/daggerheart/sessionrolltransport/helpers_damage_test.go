package sessionrolltransport

import (
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/dice"
)

func TestDamageDiceFromProto(t *testing.T) {
	t.Run("empty specs", func(t *testing.T) {
		_, err := damageDiceFromProto(nil)
		if err != dice.ErrMissingDice {
			t.Errorf("expected ErrMissingDice, got %v", err)
		}
	})

	t.Run("nil spec", func(t *testing.T) {
		_, err := damageDiceFromProto([]*pb.DiceSpec{nil})
		if err != dice.ErrInvalidDiceSpec {
			t.Errorf("expected ErrInvalidDiceSpec, got %v", err)
		}
	})

	t.Run("invalid sides", func(t *testing.T) {
		_, err := damageDiceFromProto([]*pb.DiceSpec{{Sides: 0, Count: 1}})
		if err != dice.ErrInvalidDiceSpec {
			t.Errorf("expected ErrInvalidDiceSpec, got %v", err)
		}
	})

	t.Run("valid spec", func(t *testing.T) {
		result, err := damageDiceFromProto([]*pb.DiceSpec{{Sides: 6, Count: 2}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 || result[0].Sides != 6 || result[0].Count != 2 {
			t.Errorf("unexpected result: %v", result)
		}
	})
}

func TestDiceRollsToProto(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		if got := diceRollsToProto(nil); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("valid rolls", func(t *testing.T) {
		rolls := []dice.Roll{
			{Sides: 6, Results: []int{3, 4}, Total: 7},
		}
		protos := diceRollsToProto(rolls)
		if len(protos) != 1 {
			t.Fatalf("expected 1 roll, got %d", len(protos))
		}
		if protos[0].Sides != 6 || protos[0].Total != 7 {
			t.Errorf("roll mismatch: %v", protos[0])
		}
	})
}
