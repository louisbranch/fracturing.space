package session

func deriveReadyCheckResolution(progress *GateProgress) {
	if progress == nil {
		return
	}
	switch {
	case progress.WaitCount > 0:
		progress.ResolutionState = GateResolutionStateBlocked
		progress.ResolutionReason = "wait_response_present"
		progress.SuggestedDecision = "wait"
	case progress.AllReady:
		progress.ResolutionState = GateResolutionStateReadyToResolve
		progress.ResolutionReason = "all_ready"
		progress.SuggestedDecision = "ready"
	default:
		progress.ResolutionState = GateResolutionStatePendingResponses
		progress.ResolutionReason = "waiting_on_participants"
	}
}

func deriveVoteResolution(progress *GateProgress) {
	if progress == nil {
		return
	}
	if progress.EligibleCount == 0 {
		switch {
		case progress.RespondedCount == 0:
			progress.ResolutionState = GateResolutionStatePendingResponses
			progress.ResolutionReason = "waiting_on_participants"
		case progress.LeadingOptionCount == 0:
			progress.ResolutionState = GateResolutionStateManualReview
			progress.ResolutionReason = "no_votes_recorded"
		case progress.LeadingTie:
			progress.ResolutionState = GateResolutionStateManualReview
			progress.ResolutionReason = "vote_tied"
		default:
			progress.ResolutionState = GateResolutionStateManualReview
			progress.ResolutionReason = "open_ended_vote"
			if len(progress.LeadingOptions) == 1 {
				progress.SuggestedDecision = progress.LeadingOptions[0]
			}
		}
		return
	}
	switch {
	case !progress.AllResponded:
		progress.ResolutionState = GateResolutionStatePendingResponses
		progress.ResolutionReason = "waiting_on_participants"
	case progress.LeadingOptionCount == 0:
		progress.ResolutionState = GateResolutionStateManualReview
		progress.ResolutionReason = "no_votes_recorded"
	case progress.LeadingTie:
		progress.ResolutionState = GateResolutionStateManualReview
		progress.ResolutionReason = "vote_tied"
	case len(progress.LeadingOptions) == 1:
		progress.ResolutionState = GateResolutionStateReadyToResolve
		progress.ResolutionReason = "leader_selected"
		progress.SuggestedDecision = progress.LeadingOptions[0]
	default:
		progress.ResolutionState = GateResolutionStateManualReview
		progress.ResolutionReason = "manual_resolution_required"
	}
}
