package session

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	// GateTypeReadyCheck collects participant ready/wait responses before action proceeds.
	GateTypeReadyCheck = "ready_check"
	// GateTypeVote collects participant decisions against one or more explicit options.
	GateTypeVote = "vote"
)

const (
	gateWorkflowEligibleParticipantIDsKey = "eligible_participant_ids"
	gateWorkflowOptionsKey                = "options"
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
	// non-proceed outcome (for example a ready check with one or more waits).
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

// NormalizeGateWorkflowMetadata sanitizes workflow-specific metadata while keeping
// unknown keys intact for future gate extensions.
func NormalizeGateWorkflowMetadata(gateType string, metadata map[string]any) (map[string]any, error) {
	if len(metadata) == 0 {
		switch strings.TrimSpace(gateType) {
		case GateTypeReadyCheck:
			return map[string]any{
				gateWorkflowOptionsKey:           []string{"ready", "wait"},
				gateWorkflowResponseAuthorityKey: GateResponseAuthorityParticipant,
			}, nil
		case GateTypeVote:
			return map[string]any{
				gateWorkflowResponseAuthorityKey: GateResponseAuthorityParticipant,
			}, nil
		default:
			return nil, nil
		}
	}

	normalized := make(map[string]any, len(metadata)+1)
	for key, value := range metadata {
		normalized[key] = value
	}

	eligibleIDs, err := gateWorkflowStringSlice(metadata[gateWorkflowEligibleParticipantIDsKey], gateWorkflowEligibleParticipantIDsKey)
	if err != nil {
		return nil, err
	}
	if len(eligibleIDs) > 0 {
		sort.Strings(eligibleIDs)
		normalized[gateWorkflowEligibleParticipantIDsKey] = eligibleIDs
	}

	options, err := gateWorkflowOptions(gateType, metadata[gateWorkflowOptionsKey])
	if err != nil {
		return nil, err
	}
	if len(options) > 0 {
		normalized[gateWorkflowOptionsKey] = options
	}
	responseAuthority, err := gateWorkflowResponseAuthority(gateType, metadata[gateWorkflowResponseAuthorityKey])
	if err != nil {
		return nil, err
	}
	if responseAuthority != "" {
		normalized[gateWorkflowResponseAuthorityKey] = responseAuthority
	}

	return normalized, nil
}

// ValidateGateResponse enforces gate-type-specific response rules while keeping
// the transport-facing contract generic.
func ValidateGateResponse(gateType string, metadataJSON []byte, participantID string, decision string, response map[string]any) (string, map[string]any, error) {
	participantID = strings.TrimSpace(participantID)
	if participantID == "" {
		return "", nil, fmt.Errorf("participant id is required")
	}

	metadata, err := gateWorkflowMetadataFromJSON(metadataJSON)
	if err != nil {
		return "", nil, err
	}
	eligibleIDs, err := gateWorkflowEligibleIDs(metadata)
	if err != nil {
		return "", nil, err
	}
	responseAuthority, err := gateWorkflowResponseAuthority(gateType, metadata[gateWorkflowResponseAuthorityKey])
	if err != nil {
		return "", nil, err
	}
	switch responseAuthority {
	case "", GateResponseAuthorityParticipant:
	default:
		return "", nil, fmt.Errorf("unsupported gate response authority %q", responseAuthority)
	}
	if len(eligibleIDs) > 0 && !gateWorkflowContains(eligibleIDs, participantID) {
		return "", nil, fmt.Errorf("participant %q is not eligible for this gate", participantID)
	}

	decision = strings.TrimSpace(decision)
	switch strings.TrimSpace(gateType) {
	case GateTypeReadyCheck:
		decision = strings.ToLower(decision)
		switch decision {
		case "ready", "wait":
		default:
			return "", nil, fmt.Errorf("ready_check responses must be \"ready\" or \"wait\"")
		}
	case GateTypeVote:
		if decision == "" {
			return "", nil, fmt.Errorf("vote response decision is required")
		}
		options, err := gateWorkflowOptions(gateType, metadata[gateWorkflowOptionsKey])
		if err != nil {
			return "", nil, err
		}
		if len(options) > 0 && !gateWorkflowContains(options, decision) {
			return "", nil, fmt.Errorf("vote response %q is not one of the allowed options", decision)
		}
	default:
		if decision == "" && len(response) == 0 {
			return "", nil, fmt.Errorf("gate response decision or response payload is required")
		}
	}

	if len(response) == 0 {
		return decision, nil, nil
	}
	return decision, response, nil
}

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
	sort.SliceStable(updated, func(i, j int) bool {
		return updated[i].ParticipantID < updated[j].ParticipantID
	})
	progress.Responses = updated
	recomputeGateProgress(&progress)

	return json.Marshal(progress)
}

func buildGateProgress(gateType string, metadataJSON []byte, progressJSON []byte) (GateProgress, error) {
	progress := GateProgress{}
	if len(progressJSON) > 0 {
		if err := json.Unmarshal(progressJSON, &progress); err != nil {
			return GateProgress{}, fmt.Errorf("decode gate progress: %w", err)
		}
	}
	progress.WorkflowType = strings.TrimSpace(gateType)

	metadata, err := gateWorkflowMetadataFromJSON(metadataJSON)
	if err != nil {
		return GateProgress{}, err
	}
	eligibleIDs, err := gateWorkflowEligibleIDs(metadata)
	if err != nil {
		return GateProgress{}, err
	}
	options, err := gateWorkflowOptions(gateType, metadata[gateWorkflowOptionsKey])
	if err != nil {
		return GateProgress{}, err
	}
	responseAuthority, err := gateWorkflowResponseAuthority(gateType, metadata[gateWorkflowResponseAuthorityKey])
	if err != nil {
		return GateProgress{}, err
	}
	progress.EligibleParticipantIDs = eligibleIDs
	progress.Options = options
	progress.ResponseAuthority = responseAuthority
	recomputeGateProgress(&progress)
	return progress, nil
}

func recomputeGateProgress(progress *GateProgress) {
	if progress == nil {
		return
	}
	if len(progress.Responses) == 0 {
		progress.Responses = nil
	}
	decisionCounts := map[string]int{}
	eligibleSet := map[string]struct{}{}
	for _, participantID := range progress.EligibleParticipantIDs {
		eligibleSet[participantID] = struct{}{}
	}
	respondedSet := map[string]struct{}{}

	respondedCount := 0
	for _, response := range progress.Responses {
		participantID := strings.TrimSpace(response.ParticipantID)
		if len(eligibleSet) == 0 {
			respondedCount++
			respondedSet[participantID] = struct{}{}
		} else if _, ok := eligibleSet[participantID]; ok {
			respondedCount++
			respondedSet[participantID] = struct{}{}
		}
		if decision := strings.TrimSpace(response.Decision); decision != "" {
			decisionCounts[decision]++
		}
	}

	progress.EligibleCount = len(progress.EligibleParticipantIDs)
	progress.RespondedCount = respondedCount
	if progress.EligibleCount > 0 {
		progress.PendingCount = progress.EligibleCount - progress.RespondedCount
		if progress.PendingCount < 0 {
			progress.PendingCount = 0
		}
		progress.PendingParticipantIDs = gateWorkflowPendingParticipantIDs(progress.EligibleParticipantIDs, respondedSet)
		progress.AllResponded = progress.PendingCount == 0
	} else {
		progress.PendingCount = 0
		progress.PendingParticipantIDs = nil
		progress.AllResponded = false
	}
	if len(decisionCounts) == 0 {
		progress.DecisionCounts = nil
	} else {
		progress.DecisionCounts = decisionCounts
	}
	progress.ReadyCount = 0
	progress.WaitCount = 0
	progress.AllReady = false
	progress.LeadingOptions = nil
	progress.LeadingOptionCount = 0
	progress.LeadingTie = false
	progress.ResolutionState = ""
	progress.ResolutionReason = ""
	progress.SuggestedDecision = ""

	switch strings.TrimSpace(progress.WorkflowType) {
	case GateTypeReadyCheck:
		progress.ReadyCount = decisionCounts["ready"]
		progress.WaitCount = decisionCounts["wait"]
		progress.AllReady = progress.AllResponded && progress.EligibleCount > 0 && progress.WaitCount == 0
		deriveReadyCheckResolution(progress)
	case GateTypeVote:
		progress.LeadingOptions, progress.LeadingOptionCount, progress.LeadingTie = gateWorkflowLeadingOptions(progress.Options, decisionCounts)
		deriveVoteResolution(progress)
	}
}

func gateProgressIsEmpty(progress GateProgress) bool {
	return strings.TrimSpace(progress.ResponseAuthority) == "" &&
		len(progress.EligibleParticipantIDs) == 0 &&
		len(progress.Options) == 0 &&
		len(progress.Responses) == 0 &&
		len(progress.DecisionCounts) == 0 &&
		progress.RespondedCount == 0 &&
		progress.EligibleCount == 0 &&
		progress.PendingCount == 0 &&
		len(progress.PendingParticipantIDs) == 0 &&
		progress.ReadyCount == 0 &&
		progress.WaitCount == 0 &&
		!progress.AllReady &&
		len(progress.LeadingOptions) == 0 &&
		progress.LeadingOptionCount == 0 &&
		!progress.LeadingTie &&
		strings.TrimSpace(progress.ResolutionState) == "" &&
		strings.TrimSpace(progress.ResolutionReason) == "" &&
		strings.TrimSpace(progress.SuggestedDecision) == "" &&
		!progress.AllResponded
}

func gateWorkflowMetadataFromJSON(data []byte) (map[string]any, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var metadata map[string]any
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("decode gate metadata: %w", err)
	}
	return metadata, nil
}

func gateWorkflowEligibleIDs(metadata map[string]any) ([]string, error) {
	return gateWorkflowStringSlice(metadata[gateWorkflowEligibleParticipantIDsKey], gateWorkflowEligibleParticipantIDsKey)
}

func gateWorkflowOptions(gateType string, value any) ([]string, error) {
	if strings.TrimSpace(gateType) == GateTypeReadyCheck && value == nil {
		return []string{"ready", "wait"}, nil
	}
	options, err := gateWorkflowStringSlice(value, gateWorkflowOptionsKey)
	if err != nil {
		return nil, err
	}
	switch strings.TrimSpace(gateType) {
	case GateTypeReadyCheck:
		if len(options) == 0 {
			return []string{"ready", "wait"}, nil
		}
		if len(options) != 2 || !gateWorkflowContains(options, "ready") || !gateWorkflowContains(options, "wait") {
			return nil, fmt.Errorf("ready_check options must be exactly [\"ready\", \"wait\"]")
		}
	case GateTypeVote:
		if len(options) == 1 {
			return nil, fmt.Errorf("vote options must contain at least two choices when provided")
		}
	}
	return options, nil
}

func gateWorkflowResponseAuthority(gateType string, value any) (string, error) {
	if value == nil {
		switch strings.TrimSpace(gateType) {
		case GateTypeReadyCheck, GateTypeVote:
			return GateResponseAuthorityParticipant, nil
		default:
			return "", nil
		}
	}
	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", gateWorkflowResponseAuthorityKey)
	}
	text = strings.ToLower(strings.TrimSpace(text))
	switch text {
	case "":
		return "", nil
	case GateResponseAuthorityParticipant:
		return text, nil
	default:
		return "", fmt.Errorf("%s %q is not supported", gateWorkflowResponseAuthorityKey, text)
	}
}

func gateWorkflowStringSlice(value any, fieldName string) ([]string, error) {
	if value == nil {
		return nil, nil
	}
	rawValues, ok := value.([]any)
	if ok {
		normalized := make([]string, 0, len(rawValues))
		for _, entry := range rawValues {
			text, ok := entry.(string)
			if !ok {
				return nil, fmt.Errorf("%s entries must be strings", fieldName)
			}
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}
			normalized = append(normalized, text)
		}
		return gateWorkflowUniqueStrings(normalized), nil
	}
	if rawStrings, ok := value.([]string); ok {
		return gateWorkflowUniqueStrings(rawStrings), nil
	}
	return nil, fmt.Errorf("%s must be an array of strings", fieldName)
}

func gateWorkflowUniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	unique := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	sort.Strings(unique)
	if len(unique) == 0 {
		return nil
	}
	return unique
}

func gateWorkflowContains(values []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}

func gateWorkflowPendingParticipantIDs(eligible []string, responded map[string]struct{}) []string {
	if len(eligible) == 0 {
		return nil
	}
	pending := make([]string, 0, len(eligible))
	for _, participantID := range eligible {
		if _, ok := responded[strings.TrimSpace(participantID)]; ok {
			continue
		}
		pending = append(pending, participantID)
	}
	if len(pending) == 0 {
		return nil
	}
	return pending
}

func gateWorkflowLeadingOptions(options []string, decisionCounts map[string]int) ([]string, int, bool) {
	if len(decisionCounts) == 0 {
		return nil, 0, false
	}
	candidates := options
	if len(candidates) == 0 {
		candidates = make([]string, 0, len(decisionCounts))
		for option := range decisionCounts {
			candidates = append(candidates, option)
		}
		sort.Strings(candidates)
	}
	leadingCount := 0
	leading := make([]string, 0, len(candidates))
	for _, option := range candidates {
		count := decisionCounts[strings.TrimSpace(option)]
		if count <= 0 {
			continue
		}
		switch {
		case count > leadingCount:
			leadingCount = count
			leading = []string{option}
		case count == leadingCount:
			leading = append(leading, option)
		}
	}
	if len(leading) == 0 {
		return nil, 0, false
	}
	return leading, leadingCount, len(leading) > 1
}

func deriveReadyCheckResolution(progress *GateProgress) {
	if progress == nil {
		return
	}
	switch {
	case progress.WaitCount > 0:
		progress.ResolutionState = GateResolutionStateBlocked
		progress.ResolutionReason = "wait_response_present"
		progress.SuggestedDecision = "wait"
	case progress.AllReady:
		progress.ResolutionState = GateResolutionStateReadyToResolve
		progress.ResolutionReason = "all_ready"
		progress.SuggestedDecision = "ready"
	default:
		progress.ResolutionState = GateResolutionStatePendingResponses
		progress.ResolutionReason = "waiting_on_participants"
	}
}

func deriveVoteResolution(progress *GateProgress) {
	if progress == nil {
		return
	}
	if progress.EligibleCount == 0 {
		switch {
		case progress.RespondedCount == 0:
			progress.ResolutionState = GateResolutionStatePendingResponses
			progress.ResolutionReason = "waiting_on_participants"
		case progress.LeadingOptionCount == 0:
			progress.ResolutionState = GateResolutionStateManualReview
			progress.ResolutionReason = "no_votes_recorded"
		case progress.LeadingTie:
			progress.ResolutionState = GateResolutionStateManualReview
			progress.ResolutionReason = "vote_tied"
		default:
			progress.ResolutionState = GateResolutionStateManualReview
			progress.ResolutionReason = "open_ended_vote"
			if len(progress.LeadingOptions) == 1 {
				progress.SuggestedDecision = progress.LeadingOptions[0]
			}
		}
		return
	}
	switch {
	case !progress.AllResponded:
		progress.ResolutionState = GateResolutionStatePendingResponses
		progress.ResolutionReason = "waiting_on_participants"
	case progress.LeadingOptionCount == 0:
		progress.ResolutionState = GateResolutionStateManualReview
		progress.ResolutionReason = "no_votes_recorded"
	case progress.LeadingTie:
		progress.ResolutionState = GateResolutionStateManualReview
		progress.ResolutionReason = "vote_tied"
	case len(progress.LeadingOptions) == 1:
		progress.ResolutionState = GateResolutionStateReadyToResolve
		progress.ResolutionReason = "leader_selected"
		progress.SuggestedDecision = progress.LeadingOptions[0]
	default:
		progress.ResolutionState = GateResolutionStateManualReview
		progress.ResolutionReason = "manual_resolution_required"
	}
}
