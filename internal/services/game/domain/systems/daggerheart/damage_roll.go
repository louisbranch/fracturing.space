package daggerheart

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/core/dice"

// DamageDieSpec describes damage dice to roll.
type DamageDieSpec struct {
	Sides int
	Count int
}

// DamageRollRequest describes a damage roll.
type DamageRollRequest struct {
	Dice     []DamageDieSpec
	Modifier int
	Seed     int64
	Critical bool
}

// DamageRollResult captures damage roll details.
type DamageRollResult struct {
	Rolls         []dice.Roll
	BaseTotal     int
	Modifier      int
	CriticalBonus int
	Total         int
}

// RollDamage rolls damage dice and applies critical damage bonus when requested.
func RollDamage(request DamageRollRequest) (DamageRollResult, error) {
	if len(request.Dice) == 0 {
		return DamageRollResult{}, dice.ErrMissingDice
	}

	specs := make([]dice.Spec, 0, len(request.Dice))
	criticalBonus := 0
	for _, spec := range request.Dice {
		specs = append(specs, dice.Spec{Sides: spec.Sides, Count: spec.Count})
		if request.Critical {
			criticalBonus += spec.Sides * spec.Count
		}
	}

	rollResult, err := dice.RollDice(dice.Request{
		Dice: specs,
		Seed: request.Seed,
	})
	if err != nil {
		return DamageRollResult{}, err
	}

	baseTotal := rollResult.Total + request.Modifier
	total := baseTotal + criticalBonus
	return DamageRollResult{
		Rolls:         rollResult.Rolls,
		BaseTotal:     baseTotal,
		Modifier:      request.Modifier,
		CriticalBonus: criticalBonus,
		Total:         total,
	}, nil
}
