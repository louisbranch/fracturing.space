package session

import (
	"fmt"
	"strings"
)

type genericGateWorkflow struct {
	gateWorkflowBase
}

func newGateWorkflow(gateType string, metadata map[string]any) (gateWorkflow, error) {
	switch strings.TrimSpace(gateType) {
	case GateTypeReadyCheck:
		return newReadyCheckGateWorkflow(metadata)
	case GateTypeVote:
		return newVoteGateWorkflow(metadata)
	default:
		return newGenericGateWorkflow(metadata)
	}
}

func newGenericGateWorkflow(metadata map[string]any) (genericGateWorkflow, error) {
	base, err := parseGateWorkflowBase(metadata, "")
	if err != nil {
		return genericGateWorkflow{}, err
	}
	return genericGateWorkflow{gateWorkflowBase: base}, nil
}

func (w genericGateWorkflow) validateResponse(decision string, response map[string]any) (string, map[string]any, error) {
	decision = strings.TrimSpace(decision)
	response = normalizeOptionalGateResponse(response)
	if decision == "" && len(response) == 0 {
		return "", nil, fmt.Errorf("gate response decision or response payload is required")
	}
	return decision, response, nil
}

func (w genericGateWorkflow) deriveResolution(*GateProgress) {}
