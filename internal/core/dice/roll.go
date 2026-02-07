package dice

import "math/rand"

// RollDice rolls dice based on the provided request.
//
// # Determinism
//
// RollDice is deterministic with respect to the Seed field on Request.
// Given the same Seed and the same Dice slice (including order and values),
// RollDice will always produce the same Result.
//
// # Ordering
//
// Dice specs in Request.Dice are processed in slice order. The resulting
// Roll entries in Result.Rolls appear in the same order as the
// corresponding Spec entries in Request.Dice.
//
// # Totals
//
// For each Roll in Result.Rolls, the Total field is the sum of all
// values in Results for that dice specification. The Result.Total field
// is the sum of Total for all Roll entries (i.e., the sum of every die
// rolled across the entire request).
//
// # Errors
//
//   - At least one Spec must be provided in Request.Dice, otherwise
//     ErrMissingDice is returned.
//   - Each Spec must have Sides > 0 and Count > 0, otherwise
//     ErrInvalidDiceSpec is returned.
//
// Example:
//
//	req := Request{
//	    Dice: []Spec{
//	        {Sides: 6, Count: 2}, // roll 2d6
//	        {Sides: 8, Count: 1}, // roll 1d8
//	    },
//	    Seed: 1,
//	}
//	result, err := RollDice(req)
func RollDice(request Request) (Result, error) {
	if len(request.Dice) == 0 {
		return Result{}, ErrMissingDice
	}

	rng := rand.New(rand.NewSource(request.Seed))
	rolls := make([]Roll, 0, len(request.Dice))
	total := 0

	for _, spec := range request.Dice {
		if spec.Sides <= 0 || spec.Count <= 0 {
			return Result{}, ErrInvalidDiceSpec
		}

		results := make([]int, spec.Count)
		rollTotal := 0
		for i := 0; i < spec.Count; i++ {
			value := rollDie(rng, spec.Sides)
			results[i] = value
			rollTotal += value
		}

		rolls = append(rolls, Roll{
			Sides:   spec.Sides,
			Results: results,
			Total:   rollTotal,
		})
		total += rollTotal
	}

	return Result{
		Rolls: rolls,
		Total: total,
	}, nil
}

// RollWithRng rolls dice using a provided random source.
// This is useful when you want to control the RNG directly.
func RollWithRng(rng *rand.Rand, specs []Spec) (Result, error) {
	if len(specs) == 0 {
		return Result{}, ErrMissingDice
	}

	rolls := make([]Roll, 0, len(specs))
	total := 0

	for _, spec := range specs {
		if spec.Sides <= 0 || spec.Count <= 0 {
			return Result{}, ErrInvalidDiceSpec
		}

		results := make([]int, spec.Count)
		rollTotal := 0
		for i := 0; i < spec.Count; i++ {
			value := rollDie(rng, spec.Sides)
			results[i] = value
			rollTotal += value
		}

		rolls = append(rolls, Roll{
			Sides:   spec.Sides,
			Results: results,
			Total:   rollTotal,
		})
		total += rollTotal
	}

	return Result{
		Rolls: rolls,
		Total: total,
	}, nil
}

// rollDie rolls a single die with the provided number of sides.
func rollDie(rng *rand.Rand, sides int) int {
	return rng.Intn(sides) + 1
}
