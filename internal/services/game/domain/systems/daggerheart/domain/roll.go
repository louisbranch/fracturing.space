package domain

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/core/dice"
)

// RollAction performs an action roll from the provided request.
// It uses the core dice package for deterministic rolling.
func RollAction(request ActionRequest) (ActionResult, error) {
	advantage := request.Advantage
	disadvantage := request.Disadvantage
	netAdvantage := 0
	if advantage > disadvantage {
		netAdvantage = 1
	} else if disadvantage > advantage {
		netAdvantage = -1
	}

	rollSpecs := []dice.Spec{{Sides: 12, Count: 2}}
	if netAdvantage != 0 {
		rollSpecs = append(rollSpecs, dice.Spec{Sides: 6, Count: 1})
	}

	rollResult, err := dice.RollDice(dice.Request{
		Dice: rollSpecs,
		Seed: request.Seed,
	})
	if err != nil {
		// This should be unreachable: the DiceSpec is hardcoded and always valid.
		panic(err)
	}

	hope := rollResult.Rolls[0].Results[0]
	fear := rollResult.Rolls[0].Results[1]
	advantageDie := 0
	advantageModifier := 0
	if netAdvantage != 0 {
		advantageDie = rollResult.Rolls[1].Results[0]
		if netAdvantage > 0 {
			advantageModifier = advantageDie
		} else {
			advantageModifier = -advantageDie
		}
	}

	outcome, err := EvaluateOutcome(OutcomeRequest{
		Hope:       hope,
		Fear:       fear,
		Modifier:   request.Modifier + advantageModifier,
		Difficulty: request.Difficulty,
	})
	if err != nil {
		return ActionResult{}, err
	}

	return ActionResult{
		Hope:              outcome.Hope,
		Fear:              outcome.Fear,
		Modifier:          outcome.Modifier,
		AdvantageDie:      advantageDie,
		AdvantageModifier: advantageModifier,
		Difficulty:        outcome.Difficulty,
		Total:             outcome.Total,
		IsCrit:            outcome.IsCrit,
		MeetsDifficulty:   outcome.MeetsDifficulty,
		Outcome:           outcome.Outcome,
	}, nil
}
