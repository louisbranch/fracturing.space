package provideroauth

import "strings"

// NormalizeScopes deduplicates and trims an OAuth scope list while preserving
// insertion order.
func NormalizeScopes(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, value := range in {
		scope := strings.TrimSpace(value)
		if scope == "" {
			continue
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		out = append(out, scope)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
