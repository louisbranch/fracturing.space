package session

// StartPayload captures the payload for session.start commands and session.started events.
type StartPayload struct {
	SessionID   string `json:"session_id"`
	SessionName string `json:"session_name,omitempty"`
}

// EndPayload captures the payload for session.end commands and session.ended events.
type EndPayload struct {
	SessionID string `json:"session_id"`
}

// GateOpenedPayload captures the payload for session.gate_opened events.
type GateOpenedPayload struct {
	GateID   string         `json:"gate_id"`
	GateType string         `json:"gate_type"`
	Reason   string         `json:"reason,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// GateResolvedPayload captures the payload for session.gate_resolved events.
type GateResolvedPayload struct {
	GateID     string         `json:"gate_id"`
	Decision   string         `json:"decision,omitempty"`
	Resolution map[string]any `json:"resolution,omitempty"`
}

// GateAbandonedPayload captures the payload for session.gate_abandoned events.
type GateAbandonedPayload struct {
	GateID string `json:"gate_id"`
	Reason string `json:"reason,omitempty"`
}

// SpotlightSetPayload captures the payload for session.spotlight_set events.
type SpotlightSetPayload struct {
	SpotlightType string `json:"spotlight_type"`
	CharacterID   string `json:"character_id,omitempty"`
}

// SpotlightClearedPayload captures the payload for session.spotlight_cleared events.
type SpotlightClearedPayload struct {
	Reason string `json:"reason,omitempty"`
}
