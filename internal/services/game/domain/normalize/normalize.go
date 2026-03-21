package normalize

import "strings"

// String trims leading and trailing whitespace from a string value.
func String(s string) string {
	return strings.TrimSpace(s)
}

// ID trims leading and trailing whitespace from any string-based identifier
// type (ids.CharacterID, ids.AdversaryID, etc.).
func ID[T ~string](id T) T {
	return T(strings.TrimSpace(string(id)))
}

// RequireID trims and returns the identifier. If the result is empty, ok is
// false. Callers use this to gate early-return paths that skip processing for
// blank IDs.
func RequireID[T ~string](id T) (T, bool) {
	trimmed := ID(id)
	return trimmed, trimmed != ""
}
