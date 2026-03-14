package session

import (
	"encoding/json"
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

// StoredGateResolution captures the structured resolution state persisted by
// the session gate projection.
type StoredGateResolution struct {
	Decision string
	Extra    map[string]any
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

// BuildStoredGateResolution normalizes transport/domain resolution payloads
// into the structured projection-owned gate resolution envelope.
func BuildStoredGateResolution(decision string, resolution map[string]any) (StoredGateResolution, error) {
	values, err := BuildGateResolutionMap(decision, resolution)
	if err != nil {
		return StoredGateResolution{}, err
	}
	if len(values) == 0 {
		return StoredGateResolution{}, nil
	}

	stored := StoredGateResolution{
		Extra: gateWorkflowCloneMap(values),
	}
	if decisionValue, ok := stored.Extra["decision"].(string); ok {
		stored.Decision = strings.TrimSpace(decisionValue)
		delete(stored.Extra, "decision")
	}
	if len(stored.Extra) == 0 {
		stored.Extra = nil
	}
	return stored, nil
}

// BuildGateResolutionMapFromStored rebuilds structured resolution storage as
// the transport-facing JSON-object map used by session read APIs.
func BuildGateResolutionMapFromStored(decision string, extra map[string]any) (map[string]any, error) {
	return BuildGateResolutionMap(decision, extra)
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
