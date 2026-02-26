package character

import (
	"encoding/json"
	"fmt"
	"strings"
)

func normalizeAliases(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func normalizeAliasesField(value string) ([]string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	var raw []string
	if err := json.Unmarshal([]byte(value), &raw); err != nil {
		return nil, fmt.Errorf("aliases must be a JSON array of strings")
	}
	return normalizeAliases(raw), nil
}
