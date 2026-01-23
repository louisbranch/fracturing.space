package duality

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
