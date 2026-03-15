package session

import (
	"fmt"
	"strings"
)

type readyCheckGateWorkflow struct {
	gateWorkflowBase
}

func newReadyCheckGateWorkflow(metadata map[string]any) (readyCheckGateWorkflow, error) {
	base, err := parseGateWorkflowBase(metadata, GateResponseAuthorityParticipant)
	if err != nil {
		return readyCheckGateWorkflow{}, err
	}
	if _, err := gateWorkflowOptionsForReadyCheck(metadataValue(metadata, gateWorkflowOptionsKey)); err != nil {
		return readyCheckGateWorkflow{}, err
	}
	return readyCheckGateWorkflow{gateWorkflowBase: base}, nil
}

func (w readyCheckGateWorkflow) normalizedMetadata() map[string]any {
	values := w.gateWorkflowBase.normalizedMetadata()
	if values == nil {
		values = map[string]any{}
	}
	values[gateWorkflowOptionsKey] = []string{"ready", "wait"}
	return values
}

func (w readyCheckGateWorkflow) applyProgressMetadata(progress *GateProgress) {
	w.gateWorkflowBase.applyProgressMetadata(progress)
	if progress != nil {
		progress.Options = []string{"ready", "wait"}
	}
}

func (w readyCheckGateWorkflow) validateResponse(decision string, response map[string]any) (string, map[string]any, error) {
	decision = strings.ToLower(strings.TrimSpace(decision))
	switch decision {
	case "ready", "wait":
		return decision, normalizeOptionalGateResponse(response), nil
	default:
		return "", nil, fmt.Errorf("ready_check responses must be \"ready\" or \"wait\"")
	}
}

func (w readyCheckGateWorkflow) deriveResolution(progress *GateProgress) {
	deriveReadyCheckResolution(progress)
}
