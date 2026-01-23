package duality

import "errors"

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

// RulesMetadata captures the ruleset semantics for duality roll interpretation.
type RulesMetadata struct {
	System         string
	Module         string
	RulesVersion   string
	DiceModel      string
	TotalFormula   string
	CritRule       string
	DifficultyRule string
	Outcomes       []Outcome
}

// ExplainIntermediates captures derived values for a deterministic outcome.
type ExplainIntermediates struct {
	BaseTotal       int
	Total           int
	IsCrit          bool
	MeetsDifficulty bool
	HopeGtFear      bool
	FearGtHope      bool
}

// ExplainStep represents a deterministic evaluation step.
type ExplainStep struct {
	Code    string
	Message string
	Data    map[string]any
}

// ExplainResult captures outcome details alongside explanation context.
type ExplainResult struct {
	OutcomeResult
	RulesVersion  string
	Intermediates ExplainIntermediates
	Steps         []ExplainStep
}

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

// ProbabilityRequest describes a deterministic probability evaluation.
type ProbabilityRequest struct {
	Modifier   int
	Difficulty int
}

// OutcomeCount captures a count for a specific outcome.
type OutcomeCount struct {
	Outcome Outcome
	Count   int
}

// ProbabilityResult captures exact outcome counts across the dice space.
type ProbabilityResult struct {
	TotalOutcomes int
	CritCount     int
	SuccessCount  int
	FailureCount  int
	OutcomeCounts []OutcomeCount
}
