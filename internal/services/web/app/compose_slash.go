package app

import (
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// canonicalizeTrailingSlash redirects owned slashful requests to the slashless
// browser URL before downstream module handling.
func canonicalizeTrailingSlash(prefix string, canonicalRoot bool, next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	if prefix == routepath.Root {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil || r.URL == nil {
			next.ServeHTTP(w, r)
			return
		}
		path := strings.TrimSpace(r.URL.Path)
		if path == "" || path == routepath.Root || !strings.HasSuffix(path, "/") {
			next.ServeHTTP(w, r)
			return
		}
		if path == prefix && !canonicalRoot {
			next.ServeHTTP(w, r)
			return
		}

		location := strings.TrimSuffix(path, "/")
		if location == "" {
			location = routepath.Root
		}
		if query := strings.TrimSpace(r.URL.RawQuery); query != "" {
			location += "?" + query
		}
		httpx.WriteCanonicalRedirect(w, r, location)
	})
}
