package session

import (
	"fmt"
	"strings"
)

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
	return marshalOptionalJSONObject(normalized)
}

// DecodeGateMetadataMap returns normalized workflow metadata from stored JSON.
func DecodeGateMetadataMap(gateType string, data []byte) (map[string]any, error) {
	workflow, err := decodeGateWorkflow(gateType, data)
	if err != nil {
		return nil, err
	}
	metadata := workflow.normalizedMetadata()
	if len(metadata) == 0 {
		return nil, nil
	}
	return jsonObjectFromValue(metadata)
}

// ValidateGateResponseMetadata enforces gate-type-specific response rules using
// already-decoded workflow metadata.
func ValidateGateResponseMetadata(gateType string, metadata map[string]any, participantID string, decision string, response map[string]any) (string, map[string]any, error) {
	workflow, err := newGateWorkflow(gateType, metadata)
	if err != nil {
		return "", nil, err
	}
	if err := workflow.validateParticipant(participantID); err != nil {
		return "", nil, err
	}
	return workflow.validateResponse(decision, response)
}

// BuildStoredGateMetadata normalizes transport/domain metadata into the
// structured projection-owned gate metadata envelope.
func BuildStoredGateMetadata(gateType string, metadata map[string]any) (StoredGateMetadata, error) {
	workflow, err := newGateWorkflow(gateType, metadata)
	if err != nil {
		return StoredGateMetadata{}, err
	}
	switch typed := workflow.(type) {
	case readyCheckGateWorkflow:
		return storedGateMetadataFromBase(typed.gateWorkflowBase, []string{"ready", "wait"}), nil
	case voteGateWorkflow:
		return storedGateMetadataFromBase(typed.gateWorkflowBase, typed.options), nil
	case genericGateWorkflow:
		return storedGateMetadataFromBase(typed.gateWorkflowBase, nil), nil
	default:
		return StoredGateMetadata{}, fmt.Errorf("unsupported gate workflow type %q", strings.TrimSpace(gateType))
	}
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

func storedGateMetadataFromBase(base gateWorkflowBase, options []string) StoredGateMetadata {
	return StoredGateMetadata{
		ResponseAuthority:      strings.TrimSpace(base.responseAuthority),
		EligibleParticipantIDs: append([]string(nil), base.eligibleParticipantIDs...),
		Options:                append([]string(nil), options...),
		Extra:                  gateWorkflowCloneMap(base.extraMetadata),
	}
}

func storedGateMetadataValue(gateType string, stored StoredGateMetadata) map[string]any {
	values := gateWorkflowCloneMap(stored.Extra)
	if len(stored.EligibleParticipantIDs) > 0 {
		values[gateWorkflowEligibleParticipantIDsKey] = append([]string(nil), stored.EligibleParticipantIDs...)
	}
	if strings.TrimSpace(stored.ResponseAuthority) != "" {
		values[gateWorkflowResponseAuthorityKey] = strings.TrimSpace(stored.ResponseAuthority)
	}

	switch strings.TrimSpace(gateType) {
	case GateTypeReadyCheck:
		values[gateWorkflowOptionsKey] = []string{"ready", "wait"}
	case GateTypeVote:
		if len(stored.Options) > 0 {
			values[gateWorkflowOptionsKey] = append([]string(nil), stored.Options...)
		}
	}
	if len(values) == 0 {
		return nil
	}
	return values
}
