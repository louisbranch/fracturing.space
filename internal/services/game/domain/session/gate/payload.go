package gate

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"

// GateOpenedPayload captures the payload for session.gate_opened events.
type GateOpenedPayload struct {
	GateID   ids.GateID     `json:"gate_id"`
	GateType string         `json:"gate_type"`
	Reason   string         `json:"reason,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// GateResolvedPayload captures the payload for session.gate_resolved events.
type GateResolvedPayload struct {
	GateID     ids.GateID     `json:"gate_id"`
	Decision   string         `json:"decision,omitempty"`
	Resolution map[string]any `json:"resolution,omitempty"`
}

// GateResponseRecordedPayload captures the payload for session.gate_response_recorded events.
type GateResponseRecordedPayload struct {
	GateID        ids.GateID        `json:"gate_id"`
	ParticipantID ids.ParticipantID `json:"participant_id"`
	Decision      string            `json:"decision,omitempty"`
	Response      map[string]any    `json:"response,omitempty"`
}

// GateAbandonedPayload captures the payload for session.gate_abandoned events.
type GateAbandonedPayload struct {
	GateID ids.GateID `json:"gate_id"`
	Reason string     `json:"reason,omitempty"`
}
