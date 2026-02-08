// Package encoding provides content addressing utilities for event sourcing.
package encoding

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

// CanonicalJSON produces deterministic JSON output inspired by RFC 8785 (JCS) principles:
// - Object keys sorted lexicographically
// - No unnecessary whitespace
// - Unicode normalization (via Go's json package)
// - Numbers without trailing zeros
func CanonicalJSON(v any) ([]byte, error) {
	// First marshal to get JSON representation
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	// Decode into interface{} to get the raw structure
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	// Recursively canonicalize the structure
	canonical := canonicalize(raw)

	// Marshal the canonicalized structure
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(canonical); err != nil {
		return nil, fmt.Errorf("encode canonical: %w", err)
	}

	// Remove trailing newline from Encode
	result := buf.Bytes()
	if len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}

	return result, nil
}

// canonicalize recursively processes a value to ensure canonical JSON ordering.
func canonicalize(v any) any {
	switch val := v.(type) {
	case map[string]any:
		// Sort keys and create a new ordered map representation
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		// Use a slice of key-value pairs to preserve order in output
		result := make(map[string]any, len(val))
		for _, k := range keys {
			result[k] = canonicalize(val[k])
		}
		return orderedMap{keys: keys, values: result}

	case []any:
		result := make([]any, len(val))
		for i, item := range val {
			result[i] = canonicalize(item)
		}
		return result

	default:
		return v
	}
}

// orderedMap is a helper type that marshals map keys in sorted order.
type orderedMap struct {
	keys   []string
	values map[string]any
}

// MarshalJSON implements json.Marshaler with sorted keys.
func (o orderedMap) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')

	for i, k := range o.keys {
		if i > 0 {
			buf.WriteByte(',')
		}

		// Write key using encoder with HTML escaping disabled
		keyJSON, err := marshalWithoutHTMLEscape(k)
		if err != nil {
			return nil, err
		}
		buf.Write(keyJSON)
		buf.WriteByte(':')

		// Write value using encoder with HTML escaping disabled
		valJSON, err := marshalWithoutHTMLEscape(o.values[k])
		if err != nil {
			return nil, err
		}
		buf.Write(valJSON)
	}

	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// marshalWithoutHTMLEscape marshals a value without HTML escaping to match
// the top-level encoder behavior.
func marshalWithoutHTMLEscape(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	// Remove trailing newline from Encode
	result := buf.Bytes()
	if len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}
	return result, nil
}

// ContentHash computes a SHA-256 hash of the canonical JSON representation,
// truncated to 128 bits (32 hex characters) for a compact content-addressed identity.
func ContentHash(v any) (string, error) {
	canonical, err := CanonicalJSON(v)
	if err != nil {
		return "", fmt.Errorf("canonical json: %w", err)
	}

	hash := sha256.Sum256(canonical)

	// Truncate to 128 bits (16 bytes = 32 hex chars)
	return hex.EncodeToString(hash[:16]), nil
}
