package action

// RollResolvePayload captures the payload for action.roll.resolve commands and action.roll_resolved events.
type RollResolvePayload struct {
	RequestID  string         `json:"request_id"`
	RollSeq    uint64         `json:"roll_seq"`
	Results    map[string]any `json:"results"`
	Outcome    string         `json:"outcome,omitempty"`
	SystemData map[string]any `json:"system_data,omitempty"`
}

// OutcomeAppliedChange captures a single applied change.
type OutcomeAppliedChange struct {
	CharacterID string `json:"character_id,omitempty"`
	Field       string `json:"field"`
	Before      int    `json:"before"`
	After       int    `json:"after"`
}

// OutcomeApplyPayload captures the payload for action.outcome.apply commands and action.outcome_applied events.
type OutcomeApplyPayload struct {
	RequestID            string                 `json:"request_id"`
	RollSeq              uint64                 `json:"roll_seq"`
	Targets              []string               `json:"targets"`
	RequiresComplication bool                   `json:"requires_complication"`
	AppliedChanges       []OutcomeAppliedChange `json:"applied_changes,omitempty"`
}

// OutcomeRejectPayload captures the payload for action.outcome.reject commands and action.outcome_rejected events.
type OutcomeRejectPayload struct {
	RequestID  string `json:"request_id"`
	RollSeq    uint64 `json:"roll_seq"`
	ReasonCode string `json:"reason_code"`
	Message    string `json:"message,omitempty"`
}

// NoteAddPayload captures the payload for action.note.add commands and action.note_added events.
type NoteAddPayload struct {
	Content     string `json:"content"`
	CharacterID string `json:"character_id,omitempty"`
}
