package sharedpath

import "strings"

// SplitPathParts normalizes a slash-delimited route suffix into non-empty path segments.
func SplitPathParts(path string) []string {
	rawParts := strings.Split(path, "/")
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
