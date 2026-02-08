package templates

import "golang.org/x/text/message"

// Localizer provides translated strings for templ components.
type Localizer interface {
	Sprintf(key message.Reference, args ...any) string
}

// T returns a translated string or the key if no localizer is available.
func T(loc Localizer, key message.Reference, args ...any) string {
	if loc == nil {
		if keyString, ok := key.(string); ok {
			return keyString
		}
		return ""
	}
	return loc.Sprintf(key, args...)
}
