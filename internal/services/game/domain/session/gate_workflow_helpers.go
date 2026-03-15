package session

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func decodeGateWorkflow(gateType string, data []byte) (gateWorkflow, error) {
	metadata, err := gateWorkflowMetadataFromJSON(data)
	if err != nil {
		return nil, err
	}
	return newGateWorkflow(gateType, metadata)
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
