package session

import (
	"fmt"
	"strings"
)

type voteGateWorkflow struct {
	gateWorkflowBase
	options []string
}

func newVoteGateWorkflow(metadata map[string]any) (voteGateWorkflow, error) {
	base, err := parseGateWorkflowBase(metadata, GateResponseAuthorityParticipant)
	if err != nil {
		return voteGateWorkflow{}, err
	}
	options, err := gateWorkflowOptionsForVote(metadataValue(metadata, gateWorkflowOptionsKey))
	if err != nil {
		return voteGateWorkflow{}, err
	}
	return voteGateWorkflow{
		gateWorkflowBase: base,
		options:          options,
	}, nil
}

func (w voteGateWorkflow) normalizedMetadata() map[string]any {
	values := w.gateWorkflowBase.normalizedMetadata()
	if len(w.options) == 0 {
		return values
	}
	if values == nil {
		values = map[string]any{}
	}
	values[gateWorkflowOptionsKey] = append([]string(nil), w.options...)
	return values
}

func (w voteGateWorkflow) applyProgressMetadata(progress *GateProgress) {
	w.gateWorkflowBase.applyProgressMetadata(progress)
	if progress != nil {
		progress.Options = append([]string(nil), w.options...)
	}
}

func (w voteGateWorkflow) validateResponse(decision string, response map[string]any) (string, map[string]any, error) {
	decision = strings.TrimSpace(decision)
	if decision == "" {
		return "", nil, fmt.Errorf("vote response decision is required")
	}
	if len(w.options) > 0 && !gateWorkflowContains(w.options, decision) {
		return "", nil, fmt.Errorf("vote response %q is not one of the allowed options", decision)
	}
	return decision, normalizeOptionalGateResponse(response), nil
}

func (w voteGateWorkflow) deriveResolution(progress *GateProgress) {
	deriveVoteResolution(progress)
}
