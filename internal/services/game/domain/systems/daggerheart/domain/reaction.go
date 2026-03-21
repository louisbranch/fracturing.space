package domain

// RollReaction performs a reaction roll using action roll mechanics
// with reaction-specific semantics encoded in the result.
func RollReaction(request ReactionRequest) (ReactionResult, error) {
	result, err := RollAction(ActionRequest{
		Modifier:     request.Modifier,
		Difficulty:   request.Difficulty,
		Seed:         request.Seed,
		Advantage:    request.Advantage,
		Disadvantage: request.Disadvantage,
	})
	if err != nil {
		return ReactionResult{}, err
	}

	return ReactionResult{
		ActionResult:       result,
		GeneratesHopeFear:  false,
		AidAllowed:         false,
		TriggersGMMove:     false,
		CritNegatesEffects: true,
	}, nil
}
