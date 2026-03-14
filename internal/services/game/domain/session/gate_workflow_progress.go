package session

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func buildGateProgress(gateType string, metadataJSON []byte, progressJSON []byte) (GateProgress, error) {
	progress := GateProgress{}
	if len(progressJSON) > 0 {
		if err := json.Unmarshal(progressJSON, &progress); err != nil {
			return GateProgress{}, fmt.Errorf("decode gate progress: %w", err)
		}
	}
	progress.WorkflowType = strings.TrimSpace(gateType)

	workflow, err := decodeGateWorkflow(gateType, metadataJSON)
	if err != nil {
		return GateProgress{}, err
	}
	workflow.applyProgressMetadata(&progress)
	recomputeGateProgress(&progress, workflow)
	return progress, nil
}

func recomputeGateProgress(progress *GateProgress, workflow gateWorkflow) {
	if progress == nil {
		return
	}
	if len(progress.Responses) == 0 {
		progress.Responses = nil
	} else {
		sort.SliceStable(progress.Responses, func(i, j int) bool {
			return progress.Responses[i].ParticipantID < progress.Responses[j].ParticipantID
		})
	}

	decisionCounts := map[string]int{}
	eligibleSet := map[string]struct{}{}
	for _, participantID := range progress.EligibleParticipantIDs {
		eligibleSet[participantID] = struct{}{}
	}
	respondedSet := map[string]struct{}{}

	respondedCount := 0
	for _, response := range progress.Responses {
		participantID := strings.TrimSpace(response.ParticipantID)
		if len(eligibleSet) == 0 {
			respondedCount++
			respondedSet[participantID] = struct{}{}
		} else if _, ok := eligibleSet[participantID]; ok {
			respondedCount++
			respondedSet[participantID] = struct{}{}
		}
		if decision := strings.TrimSpace(response.Decision); decision != "" {
			decisionCounts[decision]++
		}
	}

	progress.EligibleCount = len(progress.EligibleParticipantIDs)
	progress.RespondedCount = respondedCount
	if progress.EligibleCount > 0 {
		progress.PendingCount = progress.EligibleCount - progress.RespondedCount
		if progress.PendingCount < 0 {
			progress.PendingCount = 0
		}
		progress.PendingParticipantIDs = gateWorkflowPendingParticipantIDs(progress.EligibleParticipantIDs, respondedSet)
		progress.AllResponded = progress.PendingCount == 0
	} else {
		progress.PendingCount = 0
		progress.PendingParticipantIDs = nil
		progress.AllResponded = false
	}
	if len(decisionCounts) == 0 {
		progress.DecisionCounts = nil
	} else {
		progress.DecisionCounts = decisionCounts
	}
	progress.ReadyCount = 0
	progress.WaitCount = 0
	progress.AllReady = false
	progress.LeadingOptions = nil
	progress.LeadingOptionCount = 0
	progress.LeadingTie = false
	progress.ResolutionState = ""
	progress.ResolutionReason = ""
	progress.SuggestedDecision = ""
	if workflow == nil {
		return
	}

	progress.ReadyCount = decisionCounts["ready"]
	progress.WaitCount = decisionCounts["wait"]
	progress.AllReady = progress.AllResponded && progress.EligibleCount > 0 && progress.WaitCount == 0
	progress.LeadingOptions, progress.LeadingOptionCount, progress.LeadingTie = gateWorkflowLeadingOptions(progress.Options, decisionCounts)
	workflow.deriveResolution(progress)
}

func gateProgressIsEmpty(progress GateProgress) bool {
	return strings.TrimSpace(progress.ResponseAuthority) == "" &&
		len(progress.EligibleParticipantIDs) == 0 &&
		len(progress.Options) == 0 &&
		len(progress.Responses) == 0 &&
		len(progress.DecisionCounts) == 0 &&
		progress.RespondedCount == 0 &&
		progress.EligibleCount == 0 &&
		progress.PendingCount == 0 &&
		len(progress.PendingParticipantIDs) == 0 &&
		progress.ReadyCount == 0 &&
		progress.WaitCount == 0 &&
		!progress.AllReady &&
		len(progress.LeadingOptions) == 0 &&
		progress.LeadingOptionCount == 0 &&
		!progress.LeadingTie &&
		strings.TrimSpace(progress.ResolutionState) == "" &&
		strings.TrimSpace(progress.ResolutionReason) == "" &&
		strings.TrimSpace(progress.SuggestedDecision) == "" &&
		!progress.AllResponded
}

func gateWorkflowPendingParticipantIDs(eligible []string, responded map[string]struct{}) []string {
	if len(eligible) == 0 {
		return nil
	}
	pending := make([]string, 0, len(eligible))
	for _, participantID := range eligible {
		if _, ok := responded[strings.TrimSpace(participantID)]; ok {
			continue
		}
		pending = append(pending, participantID)
	}
	if len(pending) == 0 {
		return nil
	}
	return pending
}

func gateWorkflowLeadingOptions(options []string, decisionCounts map[string]int) ([]string, int, bool) {
	if len(decisionCounts) == 0 {
		return nil, 0, false
	}
	candidates := options
	if len(candidates) == 0 {
		candidates = make([]string, 0, len(decisionCounts))
		for option := range decisionCounts {
			candidates = append(candidates, option)
		}
		sort.Strings(candidates)
	}
	leadingCount := 0
	leading := make([]string, 0, len(candidates))
	for _, option := range candidates {
		count := decisionCounts[strings.TrimSpace(option)]
		if count <= 0 {
			continue
		}
		switch {
		case count > leadingCount:
			leadingCount = count
			leading = []string{option}
		case count == leadingCount:
			leading = append(leading, option)
		}
	}
	if len(leading) == 0 {
		return nil, 0, false
	}
	return leading, leadingCount, len(leading) > 1
}

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
