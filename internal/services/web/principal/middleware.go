package principal

import (
	"context"
	"net/http"

	sharedhttpx "github.com/louisbranch/fracturing.space/internal/services/shared/httpx"
)

// Middleware seeds the request-scoped principal state used by the resolver.
//
// The middleware itself does NOT perform authentication or resolve session
// state. It seeds an empty requestPrincipal into the request context. All
// actual resolution (session validation, viewer fetch, language detection,
// account profile lookup) happens lazily via sync.Once when downstream code
// first calls the corresponding resolver method. This design avoids paying
// resolution cost for routes that never inspect principal state (e.g.
// static assets, health checks).
func (r Resolver) Middleware() sharedhttpx.Middleware {
	return func(next http.Handler) http.Handler {
		if next == nil {
			next = http.NotFoundHandler()
		}
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			if request == nil {
				next.ServeHTTP(w, request)
				return
			}
			snapshot := &requestPrincipal{}
			ctx := context.WithValue(request.Context(), snapshotContextKey{}, snapshot)
			next.ServeHTTP(w, request.WithContext(ctx))
		})
	}
}

// AuthRequired reports whether the request carries a validated authenticated
// session and therefore may access protected app routes.
func (r Resolver) AuthRequired(request *http.Request) bool {
	return r.ResolveSignedIn(request)
}
