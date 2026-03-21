package gate

// NormalizeGateWorkflowMetadata sanitizes workflow-specific metadata while keeping
// unknown keys intact for future gate extensions.
func NormalizeGateWorkflowMetadata(gateType string, metadata map[string]any) (map[string]any, error) {
	workflow, err := NewGateWorkflow(gateType, metadata)
	if err != nil {
		return nil, err
	}
	return workflow.NormalizedMetadata(), nil
}

// ValidateGateResponse enforces gate-type-specific response rules while keeping
// the transport-facing contract generic.
func ValidateGateResponse(gateType string, metadataJSON []byte, participantID string, decision string, response map[string]any) (string, map[string]any, error) {
	workflow, err := DecodeGateWorkflow(gateType, metadataJSON)
	if err != nil {
		return "", nil, err
	}
	if err := workflow.ValidateParticipant(participantID); err != nil {
		return "", nil, err
	}
	return workflow.ValidateResponse(decision, response)
}
