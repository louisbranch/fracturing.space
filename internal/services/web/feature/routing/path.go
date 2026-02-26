package routing

import "strings"

// TrimSubpath returns the normalized subpath after prefix removal.
func TrimSubpath(path string, prefix string) string {
	return strings.TrimSpace(strings.TrimPrefix(path, prefix))
}

// SplitParts trims and splits a path into non-empty slash-delimited parts.
func SplitParts(path string) []string {
	rawParts := strings.Split(strings.TrimSpace(path), "/")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		parts = append(parts, part)
	}
	return parts
}

// SingleSegment extracts a single safe path segment from a prefixed path.
func SingleSegment(path string, prefix string) (string, bool) {
	segment := TrimSubpath(path, prefix)
	if segment == "" || strings.Contains(segment, "/") {
		return "", false
	}
	return segment, true
}
