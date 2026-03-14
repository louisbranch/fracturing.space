package routepath

import (
	"net/url"
	"strings"
)

// escapeSegment centralizes this web behavior in one helper seam.
func escapeSegment(raw string) string {
	return url.PathEscape(strings.TrimSpace(raw))
}
