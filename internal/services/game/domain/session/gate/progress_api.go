package gate

import (
	"encoding/json"
	"strings"
	"time"
)

// BuildInitialGateProgress returns the initial derived gate progress state for an
// opened gate.
func BuildInitialGateProgress(gateType string, metadataJSON []byte) ([]byte, error) {
	progress, err := buildGateProgress(gateType, metadataJSON, nil)
	if err != nil {
		return nil, err
	}
	if gateProgressIsEmpty(progress) {
		return nil, nil
	}
	return json.Marshal(progress)
}

// RecordGateResponseProgress applies one participant response to existing
// projection state and returns the updated encoded gate progress.
func RecordGateResponseProgress(
	gateType string,
	metadataJSON []byte,
	progressJSON []byte,
	payload GateResponseRecordedPayload,
	recordedAt time.Time,
	actorType string,
	actorID string,
) ([]byte, error) {
	progress, err := buildGateProgress(gateType, metadataJSON, progressJSON)
	if err != nil {
		return nil, err
	}

	recordedAt = recordedAt.UTC()
	nextResponse := GateProgressResponse{
		ParticipantID: strings.TrimSpace(payload.ParticipantID.String()),
		Decision:      strings.TrimSpace(payload.Decision),
		Response:      payload.Response,
		RecordedAt:    recordedAt.Format(time.RFC3339Nano),
		ActorType:     strings.TrimSpace(actorType),
		ActorID:       strings.TrimSpace(actorID),
	}

	updated := make([]GateProgressResponse, 0, len(progress.Responses)+1)
	replaced := false
	for _, existing := range progress.Responses {
		if strings.TrimSpace(existing.ParticipantID) == nextResponse.ParticipantID {
			updated = append(updated, nextResponse)
			replaced = true
			continue
		}
		updated = append(updated, existing)
	}
	if !replaced {
		updated = append(updated, nextResponse)
	}
	progress.Responses = updated
	workflow, err := DecodeGateWorkflow(gateType, metadataJSON)
	if err != nil {
		return nil, err
	}
	recomputeGateProgress(&progress, workflow)

	return json.Marshal(progress)
}
