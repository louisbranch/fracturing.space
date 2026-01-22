// Package testutil provides shared helpers for server tests.
package testutil

import "testing"

// StructInt extracts a numeric value from a map payload for tests.
func StructInt(t *testing.T, data map[string]any, key string) int {
	t.Helper()
	value, ok := data[key]
	if !ok {
		t.Fatalf("step data missing %q", key)
	}
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		t.Fatalf("step data %q has type %T", key, value)
	}
	return 0
}
