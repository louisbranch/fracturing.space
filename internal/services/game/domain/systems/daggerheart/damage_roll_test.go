package daggerheart

import (
	"errors"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/core/dice"
)

func TestRollDamageMissingDice(t *testing.T) {
	_, err := RollDamage(DamageRollRequest{})
	if !errors.Is(err, dice.ErrMissingDice) {
		t.Fatalf("expected missing dice error, got %v", err)
	}
}

func TestRollDamageAppliesModifierAndCritical(t *testing.T) {
	request := DamageRollRequest{
		Dice: []DamageDieSpec{
			{Sides: 6, Count: 2},
		},
		Modifier: 3,
		Seed:     424242,
		Critical: true,
	}

	rollResult, err := RollDamage(request)
	if err != nil {
		t.Fatalf("RollDamage failed: %v", err)
	}

	expectedRoll, err := dice.RollDice(dice.Request{
		Dice: []dice.Spec{{Sides: request.Dice[0].Sides, Count: request.Dice[0].Count}},
		Seed: request.Seed,
	})
	if err != nil {
		t.Fatalf("expected dice roll: %v", err)
	}
	expectedBase := expectedRoll.Total + request.Modifier
	expectedCritical := request.Dice[0].Sides * request.Dice[0].Count
	if rollResult.BaseTotal != expectedBase {
		t.Fatalf("base total = %d, want %d", rollResult.BaseTotal, expectedBase)
	}
	if rollResult.CriticalBonus != expectedCritical {
		t.Fatalf("critical bonus = %d, want %d", rollResult.CriticalBonus, expectedCritical)
	}
	if rollResult.Total != expectedBase+expectedCritical {
		t.Fatalf("total = %d, want %d", rollResult.Total, expectedBase+expectedCritical)
	}
}

func TestRollDamageInvalidDice(t *testing.T) {
	_, err := RollDamage(DamageRollRequest{
		Dice: []DamageDieSpec{{Sides: 6, Count: 0}},
	})
	if !errors.Is(err, dice.ErrInvalidDiceSpec) {
		t.Fatalf("expected invalid dice spec error, got %v", err)
	}
}
