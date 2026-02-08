package domain

// ExplainOutcome returns a deterministic explanation for the provided outcome request.
func ExplainOutcome(request OutcomeRequest) (ExplainResult, error) {
	result, err := EvaluateOutcome(request)
	if err != nil {
		return ExplainResult{}, err
	}

	baseTotal := request.Hope + request.Fear
	total := baseTotal + request.Modifier
	intermediates := ExplainIntermediates{
		BaseTotal:       baseTotal,
		Total:           total,
		IsCrit:          result.IsCrit,
		MeetsDifficulty: result.MeetsDifficulty,
		HopeGtFear:      request.Hope > request.Fear,
		FearGtHope:      request.Fear > request.Hope,
	}

	checkDifficultyData := map[string]any{
		"total":             total,
		"meets_difficulty":  result.MeetsDifficulty,
		"critical_override": result.IsCrit && request.Difficulty != nil,
	}
	if request.Difficulty != nil {
		checkDifficultyData["difficulty"] = *request.Difficulty
	}

	steps := []ExplainStep{
		{
			Code:    "SUM_DICE",
			Message: "Sum Hope and Fear dice",
			Data: map[string]any{
				"hope":       request.Hope,
				"fear":       request.Fear,
				"base_total": baseTotal,
			},
		},
		{
			Code:    "APPLY_MODIFIER",
			Message: "Apply modifier to base total",
			Data: map[string]any{
				"base_total": baseTotal,
				"modifier":   request.Modifier,
				"total":      total,
			},
		},
		{
			Code:    "CHECK_CRIT",
			Message: "Check for critical success",
			Data: map[string]any{
				"hope":    request.Hope,
				"fear":    request.Fear,
				"is_crit": result.IsCrit,
			},
		},
		{
			Code:    "CHECK_DIFFICULTY",
			Message: "Compare total to difficulty",
			Data:    checkDifficultyData,
		},
		{
			Code:    "SELECT_OUTCOME",
			Message: "Select outcome based on comparison",
			Data: map[string]any{
				"outcome_code":  int(result.Outcome),
				"outcome_label": result.Outcome.String(),
			},
		},
	}

	return ExplainResult{
		OutcomeResult: result,
		RulesVersion:  RulesVersion().RulesVersion,
		Intermediates: intermediates,
		Steps:         steps,
	}, nil
}
