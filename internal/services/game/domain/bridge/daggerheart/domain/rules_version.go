package domain

// RulesVersion returns the static ruleset metadata for the Duality system.
func RulesVersion() RulesMetadata {
	return RulesMetadata{
		System:         "Daggerheart",
		Module:         "Duality",
		RulesVersion:   "1.0.0",
		DiceModel:      "2d12",
		TotalFormula:   "hope + fear + modifier",
		CritRule:       "critical success on matching hope/fear; always succeeds",
		DifficultyRule: "difficulty optional; total >= difficulty succeeds; critical success always succeeds",
		Outcomes: []Outcome{
			OutcomeRollWithHope,
			OutcomeRollWithFear,
			OutcomeSuccessWithHope,
			OutcomeSuccessWithFear,
			OutcomeFailureWithHope,
			OutcomeFailureWithFear,
			OutcomeCriticalSuccess,
		},
	}
}
