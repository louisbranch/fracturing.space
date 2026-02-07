package domain

import (
	"github.com/louisbranch/fracturing.space/internal/core/dice"
)

// RollAction performs an action roll from the provided request.
// It uses the core dice package for deterministic rolling.
func RollAction(request ActionRequest) (ActionResult, error) {
	rollResult, err := dice.RollDice(dice.Request{
		Dice: []dice.Spec{{Sides: 12, Count: 2}},
		Seed: request.Seed,
	})
	if err != nil {
		// This should be unreachable: the DiceSpec is hardcoded and always valid.
		panic(err)
	}

	hope := rollResult.Rolls[0].Results[0]
	fear := rollResult.Rolls[0].Results[1]

	outcome, err := EvaluateOutcome(OutcomeRequest{
		Hope:       hope,
		Fear:       fear,
		Modifier:   request.Modifier,
		Difficulty: request.Difficulty,
	})
	if err != nil {
		return ActionResult{}, err
	}

	return ActionResult{
		Hope:            outcome.Hope,
		Fear:            outcome.Fear,
		Modifier:        outcome.Modifier,
		Difficulty:      outcome.Difficulty,
		Total:           outcome.Total,
		IsCrit:          outcome.IsCrit,
		MeetsDifficulty: outcome.MeetsDifficulty,
		Outcome:         outcome.Outcome,
	}, nil
}
