package gate

import "strings"

// StoredGateResolution captures the structured resolution state persisted by
// the session gate projection.
type StoredGateResolution struct {
	Decision string
	Extra    map[string]any
}

// MarshalGateResolutionJSON encodes gate resolution state for projection/read
// storage while preserving the explicit decision alongside arbitrary detail
// fields.
func MarshalGateResolutionJSON(decision string, resolution map[string]any) ([]byte, error) {
	if strings.TrimSpace(decision) == "" && len(resolution) == 0 {
		return nil, nil
	}
	combined := map[string]any{}
	if strings.TrimSpace(decision) != "" {
		combined["decision"] = strings.TrimSpace(decision)
	}
	for key, value := range resolution {
		combined[key] = value
	}
	return MarshalOptionalJSONObject(combined)
}

// BuildGateResolutionMap returns a normalized resolution payload that preserves
// an explicit decision alongside arbitrary detail fields.
func BuildGateResolutionMap(decision string, resolution map[string]any) (map[string]any, error) {
	data, err := MarshalGateResolutionJSON(decision, resolution)
	if err != nil {
		return nil, err
	}
	return DecodeGateResolutionMap(data)
}

// MarshalGateResolutionMapJSON encodes a previously built resolution map for
// projection storage.
func MarshalGateResolutionMapJSON(resolution map[string]any) ([]byte, error) {
	return MarshalOptionalJSONObject(resolution)
}

// DecodeGateResolutionMap returns the stored resolution payload as a JSON-object
// map for transport/read-model consumers.
func DecodeGateResolutionMap(data []byte) (map[string]any, error) {
	return DecodeOptionalJSONObject(data, "decode gate resolution")
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
		Extra: WorkflowCloneMap(values),
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
