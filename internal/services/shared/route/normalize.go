package route

import (
	"net/http"
	"strings"
)

// RedirectTrailingSlash canonicalizes request paths by stripping trailing "/" characters.
//
// It returns true when a redirect was written. Route handlers should stop further
// processing when true.
func RedirectTrailingSlash(w http.ResponseWriter, r *http.Request) bool {
	if w == nil || r == nil || r.URL == nil {
		return false
	}

	originalPath := r.URL.Path
	canonical := strings.TrimRight(originalPath, "/")
	if canonical == "" {
		canonical = "/"
	}
	if canonical == originalPath {
		return false
	}

	http.Redirect(w, r, canonical, http.StatusMovedPermanently)
	return true
}
