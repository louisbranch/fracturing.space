package dice

import apperrors "github.com/louisbranch/fracturing.space/internal/errors"

// ErrMissingDice indicates a roll request had no dice specified.
var ErrMissingDice = apperrors.New(apperrors.CodeDiceMissing, "at least one die must be provided")

// ErrInvalidDiceSpec indicates a die specification has invalid fields.
var ErrInvalidDiceSpec = apperrors.New(apperrors.CodeDiceInvalidSpec, "dice must have positive sides and count")

// Spec describes a die to roll and how many times to roll it.
// For example, 2d6 would be Spec{Sides: 6, Count: 2}.
type Spec struct {
	Sides int
	Count int
}

// Roll captures the results for rolling a single dice specification.
type Roll struct {
	Sides   int   // The number of sides on the die
	Results []int // Individual roll results
	Total   int   // Sum of all results
}

// Request describes a request to roll one or more dice.
type Request struct {
	Dice []Spec
	Seed int64
}

// Result captures the results from rolling multiple dice.
type Result struct {
	Rolls []Roll
	Total int // Sum of all rolls
}
