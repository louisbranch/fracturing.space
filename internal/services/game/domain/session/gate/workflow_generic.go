package gate

import (
	"fmt"
	"strings"
)

type GenericGateWorkflow struct {
	GateWorkflowBase
}

func NewGateWorkflow(gateType string, metadata map[string]any) (GateWorkflow, error) {
	return NewGenericGateWorkflow(metadata)
}

func NewGenericGateWorkflow(metadata map[string]any) (GenericGateWorkflow, error) {
	base, err := ParseGateWorkflowBase(metadata, "")
	if err != nil {
		return GenericGateWorkflow{}, err
	}
	return GenericGateWorkflow{GateWorkflowBase: base}, nil
}

func (w GenericGateWorkflow) ValidateResponse(decision string, response map[string]any) (string, map[string]any, error) {
	decision = strings.TrimSpace(decision)
	response = NormalizeOptionalGateResponse(response)
	if decision == "" && len(response) == 0 {
		return "", nil, fmt.Errorf("gate response decision or response payload is required")
	}
	return decision, response, nil
}

func (w GenericGateWorkflow) DeriveResolution(*GateProgress) {}
