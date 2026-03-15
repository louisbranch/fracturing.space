package daggerheart

import (
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/dice"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/random"
)

// assertRollDiceResponse validates roll dice response fields against expectations.
func assertRollDiceResponse(t *testing.T, response *pb.RollDiceResponse, seed int64, seedSource string, rollMode commonv1.RollMode, specs []dice.Spec) {
	t.Helper()

	if response == nil {
		t.Fatal("RollDice response is nil")
	}
	if response.GetRng() == nil {
		t.Fatal("RollDice rng is nil")
	}
	if response.GetRng().GetSeedUsed() != uint64(seed) {
		t.Fatalf("RollDice seed_used = %d, want %d", response.GetRng().GetSeedUsed(), seed)
	}
	if response.GetRng().GetRngAlgo() != random.RngAlgoMathRandV1 {
		t.Fatalf("RollDice rng_algo = %q, want %q", response.GetRng().GetRngAlgo(), random.RngAlgoMathRandV1)
	}
	if response.GetRng().GetSeedSource() != seedSource {
		t.Fatalf("RollDice seed_source = %q, want %q", response.GetRng().GetSeedSource(), seedSource)
	}
	if response.GetRng().GetRollMode() != rollMode {
		t.Fatalf("RollDice roll_mode = %v, want %v", response.GetRng().GetRollMode(), rollMode)
	}

	result, err := dice.RollDice(dice.Request{
		Dice: specs,
		Seed: seed,
	})
	if err != nil {
		t.Fatalf("RollDice returned error: %v", err)
	}

	if len(response.GetRolls()) != len(result.Rolls) {
		t.Fatalf("RollDice roll count = %d, want %d", len(response.GetRolls()), len(result.Rolls))
	}
	if response.Total != int32(result.Total) {
		t.Fatalf("RollDice total = %d, want %d", response.Total, result.Total)
	}

	for i, roll := range response.GetRolls() {
		want := result.Rolls[i]
		if roll.GetSides() != int32(want.Sides) {
			t.Fatalf("RollDice roll[%d] sides = %d, want %d", i, roll.GetSides(), want.Sides)
		}
		if roll.GetTotal() != int32(want.Total) {
			t.Fatalf("RollDice roll[%d] total = %d, want %d", i, roll.GetTotal(), want.Total)
		}
		if len(roll.GetResults()) != len(want.Results) {
			t.Fatalf("RollDice roll[%d] results = %v, want %v", i, roll.GetResults(), want.Results)
		}
		for j, value := range roll.GetResults() {
			if value != int32(want.Results[j]) {
				t.Fatalf("RollDice roll[%d] result[%d] = %d, want %d", i, j, value, want.Results[j])
			}
		}
	}
}
