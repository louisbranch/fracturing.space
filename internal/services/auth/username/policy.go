// Package username canonicalizes and validates auth usernames.
package username

import (
	"fmt"
	"regexp"
	"strings"
)

var canonicalPattern = regexp.MustCompile(`^[a-z][a-z0-9._-]{2,31}$`)

// Canonicalize normalizes a username to lowercase ASCII and validates policy.
func Canonicalize(input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("Username is required.")
	}

	var builder strings.Builder
	builder.Grow(len(input))
	for i := 0; i < len(input); i++ {
		ch := input[i]
		if ch > 0x7f {
			return "", fmt.Errorf("Username must be ASCII.")
		}
		if ch >= 'A' && ch <= 'Z' {
			ch = ch - 'A' + 'a'
		}
		builder.WriteByte(ch)
	}

	canonical := builder.String()
	if !canonicalPattern.MatchString(canonical) {
		return "", fmt.Errorf("Username does not match the required format.")
	}
	return canonical, nil
}
