package http

import (
	"net/http"
	"strings"
)

// WithStaticMime attaches explicit content-type hints for known static assets.
func WithStaticMime(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch path := strings.ToLower(r.URL.Path); {
		case strings.HasSuffix(path, ".css"):
			w.Header().Set("Content-Type", "text/css")
		case strings.HasSuffix(path, ".js"):
			w.Header().Set("Content-Type", "application/javascript")
		case strings.HasSuffix(path, ".svg"):
			w.Header().Set("Content-Type", "image/svg+xml")
		}
		next.ServeHTTP(w, r)
	})
}
