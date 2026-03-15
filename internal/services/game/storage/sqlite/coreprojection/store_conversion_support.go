package coreprojection

import (
	"encoding/json"
	"fmt"
	"strings"
)

// enumToStorage converts a domain enum to its uppercase storage representation.
func enumToStorage[T ~string](val T) string {
	if val == "" {
		return "UNSPECIFIED"
	}
	return strings.ToUpper(string(val))
}

// enumFromStorage converts an uppercase storage string to a domain enum
// using the domain's existing Normalize function.
func enumFromStorage[T ~string](s string, normalize func(string) (T, bool)) T {
	val, _ := normalize(s)
	return val
}

func boolToInt(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

func intToBool(value int64) bool {
	return value != 0
}

// unmarshalOptionalJSON decodes a JSON string into dest if non-empty.
// Skips silently when raw is blank; returns a labeled error on decode failure.
func unmarshalOptionalJSON[T any](raw string, dest *T, label string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(raw), dest); err != nil {
		return fmt.Errorf("decode %s: %w", label, err)
	}
	return nil
}
