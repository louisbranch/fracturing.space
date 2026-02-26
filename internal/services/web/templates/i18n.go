package templates

import (
	"fmt"

	"golang.org/x/text/message"
)

// Localizer provides translated strings for web templ components.
type Localizer interface {
	Sprintf(key message.Reference, args ...any) string
}

// T returns a translated string or a key-derived fallback.
func T(loc Localizer, key message.Reference, args ...any) string {
	if loc != nil {
		return loc.Sprintf(key, args...)
	}
	if keyString, ok := key.(string); ok {
		if len(args) > 0 {
			return fmt.Sprintf(keyString, args...)
		}
		return keyString
	}
	return ""
}
