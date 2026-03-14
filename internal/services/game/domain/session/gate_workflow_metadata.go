package session

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// gateWorkflow captures typed behavior for one workflow family while keeping
// the transport-facing contract generic at the package boundary.
type gateWorkflow interface {
	normalizedMetadata() map[string]any
	applyProgressMetadata(*GateProgress)
	validateParticipant(participantID string) error
	validateResponse(decision string, response map[string]any) (string, map[string]any, error)
	deriveResolution(*GateProgress)
}

// gateWorkflowBase holds workflow metadata shared by all gate types so
// ready-check, vote, and generic flows do not repeatedly parse raw maps.
type gateWorkflowBase struct {
	extraMetadata          map[string]any
	eligibleParticipantIDs []string
	responseAuthority      string
}

type readyCheckGateWorkflow struct {
	gateWorkflowBase
}

type voteGateWorkflow struct {
	gateWorkflowBase
	options []string
}

type genericGateWorkflow struct {
	gateWorkflowBase
}

func newGateWorkflow(gateType string, metadata map[string]any) (gateWorkflow, error) {
	switch strings.TrimSpace(gateType) {
	case GateTypeReadyCheck:
		return newReadyCheckGateWorkflow(metadata)
	case GateTypeVote:
		return newVoteGateWorkflow(metadata)
	default:
		return newGenericGateWorkflow(metadata)
	}
}

func decodeGateWorkflow(gateType string, data []byte) (gateWorkflow, error) {
	metadata, err := gateWorkflowMetadataFromJSON(data)
	if err != nil {
		return nil, err
	}
	return newGateWorkflow(gateType, metadata)
}

func newReadyCheckGateWorkflow(metadata map[string]any) (readyCheckGateWorkflow, error) {
	base, err := parseGateWorkflowBase(metadata, GateResponseAuthorityParticipant)
	if err != nil {
		return readyCheckGateWorkflow{}, err
	}
	if _, err := gateWorkflowOptionsForReadyCheck(metadataValue(metadata, gateWorkflowOptionsKey)); err != nil {
		return readyCheckGateWorkflow{}, err
	}
	return readyCheckGateWorkflow{gateWorkflowBase: base}, nil
}

func newVoteGateWorkflow(metadata map[string]any) (voteGateWorkflow, error) {
	base, err := parseGateWorkflowBase(metadata, GateResponseAuthorityParticipant)
	if err != nil {
		return voteGateWorkflow{}, err
	}
	options, err := gateWorkflowOptionsForVote(metadataValue(metadata, gateWorkflowOptionsKey))
	if err != nil {
		return voteGateWorkflow{}, err
	}
	return voteGateWorkflow{
		gateWorkflowBase: base,
		options:          options,
	}, nil
}

func newGenericGateWorkflow(metadata map[string]any) (genericGateWorkflow, error) {
	base, err := parseGateWorkflowBase(metadata, "")
	if err != nil {
		return genericGateWorkflow{}, err
	}
	return genericGateWorkflow{gateWorkflowBase: base}, nil
}

func parseGateWorkflowBase(metadata map[string]any, defaultResponseAuthority string) (gateWorkflowBase, error) {
	eligibleIDs, err := gateWorkflowStringSlice(metadataValue(metadata, gateWorkflowEligibleParticipantIDsKey), gateWorkflowEligibleParticipantIDsKey)
	if err != nil {
		return gateWorkflowBase{}, err
	}
	responseAuthority, err := gateWorkflowResponseAuthority(metadataValue(metadata, gateWorkflowResponseAuthorityKey), defaultResponseAuthority)
	if err != nil {
		return gateWorkflowBase{}, err
	}
	return gateWorkflowBase{
		extraMetadata:          gateWorkflowExtraMetadata(metadata),
		eligibleParticipantIDs: eligibleIDs,
		responseAuthority:      responseAuthority,
	}, nil
}

func metadataValue(metadata map[string]any, key string) any {
	if metadata == nil {
		return nil
	}
	return metadata[key]
}

func (w gateWorkflowBase) normalizedMetadata() map[string]any {
	values := gateWorkflowCloneMap(w.extraMetadata)
	if len(w.eligibleParticipantIDs) > 0 {
		values[gateWorkflowEligibleParticipantIDsKey] = append([]string(nil), w.eligibleParticipantIDs...)
	}
	if strings.TrimSpace(w.responseAuthority) != "" {
		values[gateWorkflowResponseAuthorityKey] = w.responseAuthority
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

func (w gateWorkflowBase) applyProgressMetadata(progress *GateProgress) {
	if progress == nil {
		return
	}
	progress.EligibleParticipantIDs = append([]string(nil), w.eligibleParticipantIDs...)
	progress.ResponseAuthority = w.responseAuthority
}

func (w gateWorkflowBase) validateParticipant(participantID string) error {
	participantID = strings.TrimSpace(participantID)
	if participantID == "" {
		return fmt.Errorf("participant id is required")
	}
	switch w.responseAuthority {
	case "", GateResponseAuthorityParticipant:
	default:
		return fmt.Errorf("unsupported gate response authority %q", w.responseAuthority)
	}
	if len(w.eligibleParticipantIDs) > 0 && !gateWorkflowContains(w.eligibleParticipantIDs, participantID) {
		return fmt.Errorf("participant %q is not eligible for this gate", participantID)
	}
	return nil
}

func (w readyCheckGateWorkflow) normalizedMetadata() map[string]any {
	values := w.gateWorkflowBase.normalizedMetadata()
	if values == nil {
		values = map[string]any{}
	}
	values[gateWorkflowOptionsKey] = []string{"ready", "wait"}
	return values
}

func (w readyCheckGateWorkflow) applyProgressMetadata(progress *GateProgress) {
	w.gateWorkflowBase.applyProgressMetadata(progress)
	if progress != nil {
		progress.Options = []string{"ready", "wait"}
	}
}

func (w readyCheckGateWorkflow) validateResponse(decision string, response map[string]any) (string, map[string]any, error) {
	decision = strings.ToLower(strings.TrimSpace(decision))
	switch decision {
	case "ready", "wait":
		return decision, normalizeOptionalGateResponse(response), nil
	default:
		return "", nil, fmt.Errorf("ready_check responses must be \"ready\" or \"wait\"")
	}
}

func (w readyCheckGateWorkflow) deriveResolution(progress *GateProgress) {
	deriveReadyCheckResolution(progress)
}

func (w voteGateWorkflow) normalizedMetadata() map[string]any {
	values := w.gateWorkflowBase.normalizedMetadata()
	if len(w.options) == 0 {
		return values
	}
	if values == nil {
		values = map[string]any{}
	}
	values[gateWorkflowOptionsKey] = append([]string(nil), w.options...)
	return values
}

func (w voteGateWorkflow) applyProgressMetadata(progress *GateProgress) {
	w.gateWorkflowBase.applyProgressMetadata(progress)
	if progress != nil {
		progress.Options = append([]string(nil), w.options...)
	}
}

func (w voteGateWorkflow) validateResponse(decision string, response map[string]any) (string, map[string]any, error) {
	decision = strings.TrimSpace(decision)
	if decision == "" {
		return "", nil, fmt.Errorf("vote response decision is required")
	}
	if len(w.options) > 0 && !gateWorkflowContains(w.options, decision) {
		return "", nil, fmt.Errorf("vote response %q is not one of the allowed options", decision)
	}
	return decision, normalizeOptionalGateResponse(response), nil
}

func (w voteGateWorkflow) deriveResolution(progress *GateProgress) {
	deriveVoteResolution(progress)
}

func (w genericGateWorkflow) validateResponse(decision string, response map[string]any) (string, map[string]any, error) {
	decision = strings.TrimSpace(decision)
	response = normalizeOptionalGateResponse(response)
	if decision == "" && len(response) == 0 {
		return "", nil, fmt.Errorf("gate response decision or response payload is required")
	}
	return decision, response, nil
}

func (w genericGateWorkflow) deriveResolution(*GateProgress) {}

func normalizeOptionalGateResponse(response map[string]any) map[string]any {
	if len(response) == 0 {
		return nil
	}
	return response
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

func gateWorkflowExtraMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return nil
	}
	extra := make(map[string]any, len(metadata))
	for key, value := range metadata {
		switch key {
		case gateWorkflowEligibleParticipantIDsKey, gateWorkflowOptionsKey, gateWorkflowResponseAuthorityKey:
			continue
		default:
			extra[key] = value
		}
	}
	if len(extra) == 0 {
		return nil
	}
	return extra
}

func gateWorkflowCloneMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func gateWorkflowOptionsForReadyCheck(value any) ([]string, error) {
	if value == nil {
		return []string{"ready", "wait"}, nil
	}
	options, err := gateWorkflowStringSlice(value, gateWorkflowOptionsKey)
	if err != nil {
		return nil, err
	}
	if len(options) == 0 {
		return []string{"ready", "wait"}, nil
	}
	if len(options) != 2 || !gateWorkflowContains(options, "ready") || !gateWorkflowContains(options, "wait") {
		return nil, fmt.Errorf("ready_check options must be exactly [\"ready\", \"wait\"]")
	}
	return []string{"ready", "wait"}, nil
}

func gateWorkflowOptionsForVote(value any) ([]string, error) {
	options, err := gateWorkflowStringSlice(value, gateWorkflowOptionsKey)
	if err != nil {
		return nil, err
	}
	if len(options) == 1 {
		return nil, fmt.Errorf("vote options must contain at least two choices when provided")
	}
	return options, nil
}

func gateWorkflowResponseAuthority(value any, defaultValue string) (string, error) {
	if value == nil {
		return defaultValue, nil
	}
	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", gateWorkflowResponseAuthorityKey)
	}
	text = strings.ToLower(strings.TrimSpace(text))
	switch text {
	case "":
		return defaultValue, nil
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
