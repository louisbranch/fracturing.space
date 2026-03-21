package gate

import (
	"encoding/json"
	"fmt"
	"time"
)

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

// BuildGateProgressFromResponses rebuilds derived gate progress from structured
// metadata plus the persisted participant response rows.
func BuildGateProgressFromResponses(gateType string, metadata map[string]any, responses []GateProgressResponse) (*GateProgress, error) {
	metadataJSON, err := MarshalGateMetadataJSON(gateType, metadata)
	if err != nil {
		return nil, err
	}

	var progressJSON []byte
	if len(responses) > 0 {
		progressJSON, err = json.Marshal(GateProgress{
			Responses: append([]GateProgressResponse(nil), responses...),
		})
		if err != nil {
			return nil, fmt.Errorf("encode gate responses: %w", err)
		}
	}
	return DecodeGateProgress(gateType, metadataJSON, progressJSON)
}
