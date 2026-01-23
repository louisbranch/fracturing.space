package domain

// DualityProbability computes outcome counts across all d12 pairs.
func DualityProbability(request ProbabilityRequest) (ProbabilityResult, error) {
	if request.Difficulty < 0 {
		return ProbabilityResult{}, ErrInvalidDifficulty
	}

	const totalOutcomes = 12 * 12
	outcomeCounts := make(map[Outcome]int)
	critCount := 0
	successCount := 0
	failureCount := 0

	for hope := 1; hope <= 12; hope++ {
		for fear := 1; fear <= 12; fear++ {
			result, err := EvaluateOutcome(OutcomeRequest{
				Hope:       hope,
				Fear:       fear,
				Modifier:   request.Modifier,
				Difficulty: &request.Difficulty,
			})
			if err != nil {
				return ProbabilityResult{}, err
			}

			outcomeCounts[result.Outcome]++
			if result.IsCrit {
				critCount++
			}
			switch result.Outcome {
			case OutcomeCriticalSuccess, OutcomeSuccessWithHope, OutcomeSuccessWithFear:
				successCount++
			case OutcomeFailureWithHope, OutcomeFailureWithFear:
				failureCount++
			}
		}
	}

	ordered := []Outcome{
		OutcomeCriticalSuccess,
		OutcomeSuccessWithHope,
		OutcomeSuccessWithFear,
		OutcomeFailureWithHope,
		OutcomeFailureWithFear,
	}
	counts := make([]OutcomeCount, 0, len(ordered))
	for _, outcome := range ordered {
		counts = append(counts, OutcomeCount{
			Outcome: outcome,
			Count:   outcomeCounts[outcome],
		})
	}

	return ProbabilityResult{
		TotalOutcomes: totalOutcomes,
		CritCount:     critCount,
		SuccessCount:  successCount,
		FailureCount:  failureCount,
		OutcomeCounts: counts,
	}, nil
}
