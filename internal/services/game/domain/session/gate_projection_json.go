package session

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

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

// BuildInitialGateProgressState returns the initial typed gate progress for an
// opened gate.
func BuildInitialGateProgressState(gateType string, metadata map[string]any) (*GateProgress, error) {
	metadataJSON, err := MarshalGateMetadataJSON(gateType, metadata)
	if err != nil {
		return nil, err
	}
	return DecodeGateProgress(gateType, metadataJSON, nil)
}

// DecodeGateProgressMap rebuilds stored progress as a stable JSON-object map for
// transport/read-model consumers.
func DecodeGateProgressMap(gateType string, metadataJSON, progressJSON []byte) (map[string]any, error) {
	progress, err := DecodeGateProgress(gateType, metadataJSON, progressJSON)
	if err != nil {
		return nil, err
	}
	if progress == nil {
		return nil, nil
	}
	return jsonObjectFromValue(progress)
}

// DecodeGateProgress rebuilds stored progress as typed gate progress state.
func DecodeGateProgress(gateType string, metadataJSON, progressJSON []byte) (*GateProgress, error) {
	progress, err := buildGateProgress(gateType, metadataJSON, progressJSON)
	if err != nil {
		return nil, err
	}
	if gateProgressIsEmpty(progress) {
		return nil, nil
	}
	return &progress, nil
}

// MarshalGateProgressJSON encodes typed gate progress for projection storage.
func MarshalGateProgressJSON(progress *GateProgress) ([]byte, error) {
	if progress == nil || gateProgressIsEmpty(*progress) {
		return nil, nil
	}
	return json.Marshal(progress)
}

// RecordGateResponseProgressState applies one participant response to typed gate
// progress and returns the updated typed state.
func RecordGateResponseProgressState(
	gateType string,
	metadata map[string]any,
	progress *GateProgress,
	payload GateResponseRecordedPayload,
	recordedAt time.Time,
	actorType string,
	actorID string,
) (*GateProgress, error) {
	metadataJSON, err := MarshalGateMetadataJSON(gateType, metadata)
	if err != nil {
		return nil, err
	}
	progressJSON, err := MarshalGateProgressJSON(progress)
	if err != nil {
		return nil, err
	}
	nextProgressJSON, err := RecordGateResponseProgress(
		gateType,
		metadataJSON,
		progressJSON,
		payload,
		recordedAt,
		actorType,
		actorID,
	)
	if err != nil {
		return nil, err
	}
	return DecodeGateProgress(gateType, metadataJSON, nextProgressJSON)
}

// MarshalGateResolutionJSON encodes gate resolution state for projection/read
// storage while preserving the explicit decision alongside arbitrary detail
// fields.
func MarshalGateResolutionJSON(decision string, resolution map[string]any) ([]byte, error) {
	if strings.TrimSpace(decision) == "" && len(resolution) == 0 {
		return nil, nil
	}
	combined := map[string]any{}
	if strings.TrimSpace(decision) != "" {
		combined["decision"] = strings.TrimSpace(decision)
	}
	for key, value := range resolution {
		combined[key] = value
	}
	return marshalOptionalJSONObject(combined)
}

// BuildGateResolutionMap returns a normalized resolution payload that preserves
// an explicit decision alongside arbitrary detail fields.
func BuildGateResolutionMap(decision string, resolution map[string]any) (map[string]any, error) {
	data, err := MarshalGateResolutionJSON(decision, resolution)
	if err != nil {
		return nil, err
	}
	return DecodeGateResolutionMap(data)
}

// MarshalGateResolutionMapJSON encodes a previously built resolution map for
// projection storage.
func MarshalGateResolutionMapJSON(resolution map[string]any) ([]byte, error) {
	return marshalOptionalJSONObject(resolution)
}

// DecodeGateResolutionMap returns the stored resolution payload as a JSON-object
// map for transport/read-model consumers.
func DecodeGateResolutionMap(data []byte) (map[string]any, error) {
	return decodeOptionalJSONObject(data, "decode gate resolution")
}

func marshalOptionalJSONObject(values map[string]any) ([]byte, error) {
	if len(values) == 0 {
		return nil, nil
	}
	return json.Marshal(values)
}

func decodeOptionalJSONObject(data []byte, decodeMessage string) (map[string]any, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var values map[string]any
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("%s: %w", decodeMessage, err)
	}
	return values, nil
}

func jsonObjectFromValue(value any) (map[string]any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode gate projection value: %w", err)
	}
	var values map[string]any
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("decode gate projection value: %w", err)
	}
	return values, nil
}

// JSONMapFromValue converts typed projection state into a generic JSON object
// map for transport/storage adapters.
func JSONMapFromValue(value any) (map[string]any, error) {
	return jsonObjectFromValue(value)
}
