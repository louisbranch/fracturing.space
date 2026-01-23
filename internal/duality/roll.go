package duality

import "math/rand"

// RollDice rolls dice based on the provided request.
//
// # Determinism
//
// RollDice is deterministic with respect to the Seed field on RollRequest.
// Given the same Seed and the same Dice slice (including order and values),
// RollDice will always produce the same RollResult.
//
// # Ordering
//
// Dice specs in RollRequest.Dice are processed in slice order. The resulting
// DieRoll entries in RollResult.Rolls appear in the same order as the
// corresponding DiceSpec entries in RollRequest.Dice.
//
// # Totals
//
// For each DieRoll in RollResult.Rolls, the Total field is the sum of all
// values in Results for that dice specification. The RollResult.Total field
// is the sum of Total for all DieRoll entries (i.e., the sum of every die
// rolled across the entire request).
//
// Constraints and errors
//
//   - At least one DiceSpec must be provided in RollRequest.Dice, otherwise
//     ErrMissingDice is returned.
//   - Each DiceSpec must have Sides > 0 and Count > 0, otherwise
//     ErrInvalidDiceSpec is returned.
//
// Example:
//
//	req := RollRequest{
//	    Dice: []DiceSpec{
//	        {Sides: 6, Count: 2}, // roll 2d6
//	        {Sides: 8, Count: 1}, // roll 1d8
//	    },
//	    Seed: 1,
//	}
//	result, err := RollDice(req)
//
// After a successful call, result.Rolls will contain two DieRoll entries
// (one for the d6s, one for the d8), and result.Total will equal the sum
// of all dice rolled in those entries.
func RollDice(request RollRequest) (RollResult, error) {
	if len(request.Dice) == 0 {
		return RollResult{}, ErrMissingDice
	}

	rng := rand.New(rand.NewSource(request.Seed))
	rolls := make([]DieRoll, 0, len(request.Dice))
	total := 0

	for _, spec := range request.Dice {
		if spec.Sides <= 0 || spec.Count <= 0 {
			return RollResult{}, ErrInvalidDiceSpec
		}

		results := make([]int, spec.Count)
		rollTotal := 0
		for i := 0; i < spec.Count; i++ {
			value := rollDie(rng, spec.Sides)
			results[i] = value
			rollTotal += value
		}

		rolls = append(rolls, DieRoll{
			Sides:   spec.Sides,
			Results: results,
			Total:   rollTotal,
		})
		total += rollTotal
	}

	return RollResult{
		Rolls: rolls,
		Total: total,
	}, nil
}

// RollAction performs an action roll from the provided request.
func RollAction(request ActionRequest) (ActionResult, error) {
	rollResult, err := RollDice(RollRequest{
		Dice: []DiceSpec{{Sides: 12, Count: 2}},
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

// rollDie rolls a die with the provided number of sides.
func rollDie(rng *rand.Rand, sides int) int {
	return rng.Intn(sides) + 1
}
