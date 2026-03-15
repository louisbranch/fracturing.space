package routeparam

import (
	"net/http"
	"strings"
)

// Read returns a trimmed route parameter and whether it is present.
func Read(r *http.Request, name string) (string, bool) {
	if r == nil {
		return "", false
	}
	value := strings.TrimSpace(r.PathValue(strings.TrimSpace(name)))
	if value == "" {
		return "", false
	}
	return value, true
}

// WithRequired extracts one required route parameter and delegates to fn.
// When the parameter is missing, onMissing handles the response instead.
func WithRequired(
	name string,
	onMissing func(http.ResponseWriter, *http.Request),
	fn func(http.ResponseWriter, *http.Request, string),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		value, ok := Read(r, name)
		if !ok {
			if onMissing != nil {
				onMissing(w, r)
			}
			return
		}
		if fn != nil {
			fn(w, r, value)
		}
	}
}
