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

// ErrInvalidDualityDie indicates hope or fear dice are outside the 1-12 range.
var ErrInvalidDualityDie = errors.New("duality dice must be between 1 and 12")

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

// ActionRequest describes an action roll request.
type ActionRequest struct {
	Modifier   int
	Difficulty *int
	Seed       int64
}

// ActionResult contains the outcome of an action roll.
type ActionResult struct {
	Hope            int
	Fear            int
	Modifier        int
	Difficulty      *int
	Total           int
	IsCrit          bool
	MeetsDifficulty bool
	Outcome         Outcome
}

// OutcomeRequest describes a deterministic duality outcome evaluation.
type OutcomeRequest struct {
	Hope       int
	Fear       int
	Modifier   int
	Difficulty *int
}

// OutcomeResult captures the deterministic outcome evaluation.
type OutcomeResult struct {
	Hope            int
	Fear            int
	Modifier        int
	Difficulty      *int
	Total           int
	IsCrit          bool
	MeetsDifficulty bool
	Outcome         Outcome
}

// EvaluateOutcome deterministically resolves an action roll outcome.
func EvaluateOutcome(request OutcomeRequest) (OutcomeResult, error) {
	if request.Hope < 1 || request.Hope > 12 || request.Fear < 1 || request.Fear > 12 {
		return OutcomeResult{}, ErrInvalidDualityDie
	}
	if request.Difficulty != nil && *request.Difficulty < 0 {
		return OutcomeResult{}, ErrInvalidDifficulty
	}

	total := request.Hope + request.Fear + request.Modifier
	isCrit := request.Hope == request.Fear
	meetsDifficulty := false
	if request.Difficulty != nil {
		meetsDifficulty = total >= *request.Difficulty
	}
	if isCrit && request.Difficulty != nil {
		meetsDifficulty = true
	}

	outcome := OutcomeUnspecified
	switch {
	case isCrit:
		outcome = OutcomeCriticalSuccess
	case request.Difficulty == nil && request.Hope > request.Fear:
		outcome = OutcomeRollWithHope
	case request.Difficulty == nil && request.Fear > request.Hope:
		outcome = OutcomeRollWithFear
	case meetsDifficulty && request.Hope > request.Fear:
		outcome = OutcomeSuccessWithHope
	case meetsDifficulty && request.Fear > request.Hope:
		outcome = OutcomeSuccessWithFear
	case !meetsDifficulty && request.Hope > request.Fear:
		outcome = OutcomeFailureWithHope
	case !meetsDifficulty && request.Fear > request.Hope:
		outcome = OutcomeFailureWithFear
	}

	return OutcomeResult{
		Hope:            request.Hope,
		Fear:            request.Fear,
		Modifier:        request.Modifier,
		Difficulty:      request.Difficulty,
		Total:           total,
		IsCrit:          isCrit,
		MeetsDifficulty: meetsDifficulty,
		Outcome:         outcome,
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
