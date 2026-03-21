package gate

import "strings"

// StoredGateMetadata captures the structured workflow metadata persisted by the
// session gate projection.
type StoredGateMetadata struct {
	ResponseAuthority      string
	EligibleParticipantIDs []string
	Options                []string
	Extra                  map[string]any
}

// MarshalGateMetadataJSON encodes normalized gate workflow metadata for
// projection/read-model storage.
func MarshalGateMetadataJSON(gateType string, metadata map[string]any) ([]byte, error) {
	normalized, err := NormalizeGateWorkflowMetadata(gateType, metadata)
	if err != nil {
		return nil, err
	}
	return MarshalOptionalJSONObject(normalized)
}

// DecodeGateMetadataMap returns normalized workflow metadata from stored JSON.
func DecodeGateMetadataMap(gateType string, data []byte) (map[string]any, error) {
	workflow, err := DecodeGateWorkflow(gateType, data)
	if err != nil {
		return nil, err
	}
	metadata := workflow.NormalizedMetadata()
	if len(metadata) == 0 {
		return nil, nil
	}
	return jsonObjectFromValue(metadata)
}

// ValidateGateResponseMetadata enforces gate-type-specific response rules using
// already-decoded workflow metadata.
func ValidateGateResponseMetadata(gateType string, metadata map[string]any, participantID string, decision string, response map[string]any) (string, map[string]any, error) {
	workflow, err := NewGateWorkflow(gateType, metadata)
	if err != nil {
		return "", nil, err
	}
	if err := workflow.ValidateParticipant(participantID); err != nil {
		return "", nil, err
	}
	return workflow.ValidateResponse(decision, response)
}

// BuildStoredGateMetadata normalizes transport/domain metadata into the
// structured projection-owned gate metadata envelope.
func BuildStoredGateMetadata(gateType string, metadata map[string]any) (StoredGateMetadata, error) {
	workflow, err := NewGateWorkflow(gateType, metadata)
	if err != nil {
		return StoredGateMetadata{}, err
	}
	typed, _ := workflow.(GenericGateWorkflow)
	return storedGateMetadataFromBase(typed.GateWorkflowBase, nil), nil
}

// BuildGateMetadataMapFromStored rebuilds structured projection metadata as the
// transport-facing JSON-object map used by session read APIs.
func BuildGateMetadataMapFromStored(gateType string, stored StoredGateMetadata) (map[string]any, error) {
	values := storedGateMetadataValue(gateType, stored)
	if len(values) == 0 {
		return nil, nil
	}
	return jsonObjectFromValue(values)
}

func storedGateMetadataFromBase(base GateWorkflowBase, options []string) StoredGateMetadata {
	return StoredGateMetadata{
		ResponseAuthority:      strings.TrimSpace(base.ResponseAuthority),
		EligibleParticipantIDs: append([]string(nil), base.EligibleParticipantIDs...),
		Options:                append([]string(nil), options...),
		Extra:                  WorkflowCloneMap(base.ExtraMetadata),
	}
}

func storedGateMetadataValue(gateType string, stored StoredGateMetadata) map[string]any {
	values := WorkflowCloneMap(stored.Extra)
	if len(stored.EligibleParticipantIDs) > 0 {
		values[WorkflowEligibleParticipantIDsKey] = append([]string(nil), stored.EligibleParticipantIDs...)
	}
	if strings.TrimSpace(stored.ResponseAuthority) != "" {
		values[WorkflowResponseAuthorityKey] = strings.TrimSpace(stored.ResponseAuthority)
	}

	if len(values) == 0 {
		return nil
	}
	return values
}
