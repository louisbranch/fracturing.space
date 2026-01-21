// Package dice implements the dice-rolling logic for the Duality Engine.
package dice

import (
	"errors"
	"math/rand"
)

// Outcome represents the outcome of an action roll.
type Outcome int

const (
	OutcomeUnspecified Outcome = iota
	OutcomeRollWithHope
	OutcomeRollWithFear
	OutcomeSuccessWithHope
	OutcomeSuccessWithFear
	OutcomeFailureWithHope
	OutcomeFailureWithFear
	OutcomeCriticalSuccess
)

func (o Outcome) String() string {
	switch o {
	case OutcomeUnspecified:
		return "Unspecified"
	case OutcomeRollWithHope:
		return "Roll with hope"
	case OutcomeRollWithFear:
		return "Roll with fear"
	case OutcomeSuccessWithHope:
		return "Success with hope"
	case OutcomeSuccessWithFear:
		return "Success with fear"
	case OutcomeFailureWithHope:
		return "Failure with hope"
	case OutcomeFailureWithFear:
		return "Failure with fear"
	case OutcomeCriticalSuccess:
		return "Critical success"
	default:
		return "Unknown"
	}
}

// ErrInvalidDifficulty indicates the difficulty is invalid for a roll.
var ErrInvalidDifficulty = errors.New("difficulty must be non-negative")

// ErrMissingDice indicates a roll request had no dice specified.
var ErrMissingDice = errors.New("at least one die must be provided")

// ErrInvalidDiceSpec indicates a die specification has invalid fields.
var ErrInvalidDiceSpec = errors.New("dice must have positive sides and count")

// DiceSpec describes a die to roll and how many times to roll it.
type DiceSpec struct {
	Sides int
	Count int
}

// DieRoll captures the results for a single dice spec.
type DieRoll struct {
	Sides   int
	Results []int
	Total   int
}

// RollRequest describes a request to roll one or more dice.
type RollRequest struct {
	Dice []DiceSpec
	Seed int64
}

// RollResult captures the results from rolling multiple dice.
type RollResult struct {
	Rolls []DieRoll
	Total int
}

// RollDice rolls dice based on the provided request.
//
// Determinism
//
// RollDice is deterministic with respect to the Seed field on RollRequest.
// Given the same Seed and the same Dice slice (including order and values),
// RollDice will always produce the same RollResult.
//
// Ordering
//
// Dice specs in RollRequest.Dice are processed in slice order. The resulting
// DieRoll entries in RollResult.Rolls appear in the same order as the
// corresponding DiceSpec entries in RollRequest.Dice.
//
// Totals
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
//   req := RollRequest{
//       Dice: []DiceSpec{
//           {Sides: 6, Count: 2}, // roll 2d6
//           {Sides: 8, Count: 1}, // roll 1d8
//       },
//       Seed: 1,
//   }
//   result, err := RollDice(req)
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

// ActionRequest describes an action roll request.
type ActionRequest struct {
	Modifier   int
	Difficulty *int
	Seed       int64
}

// ActionResult contains the outcome of an action roll.
type ActionResult struct {
	Hope    int
	Fear    int
	Total   int
	Outcome Outcome
}

// RollAction performs an action roll from the provided request.
func RollAction(request ActionRequest) (ActionResult, error) {
	if request.Difficulty != nil && *request.Difficulty < 0 {
		return ActionResult{}, ErrInvalidDifficulty
	}

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
	total := rollResult.Total + request.Modifier

	return ActionResult{
		Hope:    hope,
		Fear:    fear,
		Total:   total,
		Outcome: outcomeFor(hope, fear, total, request.Difficulty),
	}, nil
}

// rollDie rolls a die with the provided number of sides.
func rollDie(rng *rand.Rand, sides int) int {
	return rng.Intn(sides) + 1
}

// outcomeFor determines the roll outcome based on totals and difficulty.
func outcomeFor(hope int, fear int, total int, difficulty *int) Outcome {
	unopposed := difficulty == nil
	success := !unopposed && total >= *difficulty

	switch {
	case hope == fear:
		return OutcomeCriticalSuccess
	case hope > fear && unopposed:
		return OutcomeRollWithHope
	case fear > hope && unopposed:
		return OutcomeRollWithFear
	case hope > fear && success:
		return OutcomeSuccessWithHope
	case hope < fear && success:
		return OutcomeSuccessWithFear
	case hope > fear:
		return OutcomeFailureWithHope
	case hope < fear:
		return OutcomeFailureWithFear
	default:
		return OutcomeUnspecified
	}
}
