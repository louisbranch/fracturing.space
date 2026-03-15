package session

const (
	gateWorkflowEligibleParticipantIDsKey = "eligible_participant_ids"
	gateWorkflowResponseAuthorityKey      = "response_authority"
)

const (
	// GateResponseAuthorityParticipant means one participant seat owns exactly one
	// counted response in the workflow, regardless of which persona they are
	// currently speaking as in transcript surfaces.
	GateResponseAuthorityParticipant = "participant"
)

const (
	// GateResolutionStatePendingResponses means the workflow still needs more
	// responses before a stable outcome can be derived.
	GateResolutionStatePendingResponses = "pending_responses"
	// GateResolutionStateReadyToResolve means the workflow has a stable derived
	// outcome and can be resolved without additional participant input.
	GateResolutionStateReadyToResolve = "ready_to_resolve"
	// GateResolutionStateBlocked means current responses produce a stable
	// non-proceed outcome.
	GateResolutionStateBlocked = "blocked"
	// GateResolutionStateManualReview means participant responses are complete but
	// no single derived outcome exists, so a human/GM must decide.
	GateResolutionStateManualReview = "manual_review"
)

// GateProgress captures the derived in-progress state for one open gate.
//
// It is projection-owned read state rebuilt from gate metadata plus response
// events; transports should treat it as authoritative UI data and never mutate
// it directly.
type GateProgress struct {
	WorkflowType           string                 `json:"workflow_type,omitempty"`
	ResponseAuthority      string                 `json:"response_authority,omitempty"`
	EligibleParticipantIDs []string               `json:"eligible_participant_ids,omitempty"`
	Options                []string               `json:"options,omitempty"`
	Responses              []GateProgressResponse `json:"responses,omitempty"`
	RespondedCount         int                    `json:"responded_count"`
	EligibleCount          int                    `json:"eligible_count"`
	PendingCount           int                    `json:"pending_count"`
	PendingParticipantIDs  []string               `json:"pending_participant_ids,omitempty"`
	AllResponded           bool                   `json:"all_responded"`
	DecisionCounts         map[string]int         `json:"decision_counts,omitempty"`
	ReadyCount             int                    `json:"ready_count,omitempty"`
	WaitCount              int                    `json:"wait_count,omitempty"`
	AllReady               bool                   `json:"all_ready,omitempty"`
	LeadingOptions         []string               `json:"leading_options,omitempty"`
	LeadingOptionCount     int                    `json:"leading_option_count,omitempty"`
	LeadingTie             bool                   `json:"leading_tie,omitempty"`
	ResolutionState        string                 `json:"resolution_state,omitempty"`
	ResolutionReason       string                 `json:"resolution_reason,omitempty"`
	SuggestedDecision      string                 `json:"suggested_decision,omitempty"`
}

// GateProgressResponse stores one participant's latest gate response.
type GateProgressResponse struct {
	ParticipantID string         `json:"participant_id"`
	Decision      string         `json:"decision,omitempty"`
	Response      map[string]any `json:"response,omitempty"`
	RecordedAt    string         `json:"recorded_at,omitempty"`
	ActorType     string         `json:"actor_type,omitempty"`
	ActorID       string         `json:"actor_id,omitempty"`
}
