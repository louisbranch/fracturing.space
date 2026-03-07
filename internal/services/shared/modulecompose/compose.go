// Package modulecompose provides shared validation for HTTP module composition.
package modulecompose

import (
	"fmt"
	"strings"
)

// ValidatePrefix rejects non-canonical module mount prefixes before route
// registration to avoid duplicate ownership and accidental mismatches.
func ValidatePrefix(prefix string) error {
	if prefix == "" {
		return fmt.Errorf("prefix is required")
	}
	if strings.TrimSpace(prefix) != prefix {
		return fmt.Errorf("prefix must not include surrounding whitespace")
	}
	if !strings.HasPrefix(prefix, "/") {
		return fmt.Errorf("prefix must begin with /")
	}
	if !strings.HasSuffix(prefix, "/") {
		return fmt.Errorf("prefix must end with /")
	}
	return nil
}
