package gate

import (
	"encoding/json"
	"fmt"
)

func MarshalOptionalJSONObject(values map[string]any) ([]byte, error) {
	if len(values) == 0 {
		return nil, nil
	}
	return json.Marshal(values)
}

func DecodeOptionalJSONObject(data []byte, decodeMessage string) (map[string]any, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var values map[string]any
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("%s: %w", decodeMessage, err)
	}
	return values, nil
}

func jsonObjectFromValue(value any) (map[string]any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode gate projection value: %w", err)
	}
	var values map[string]any
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("decode gate projection value: %w", err)
	}
	return values, nil
}

// JSONMapFromValue converts typed projection state into a generic JSON object
// map for transport/storage adapters.
func JSONMapFromValue(value any) (map[string]any, error) {
	return jsonObjectFromValue(value)
}
