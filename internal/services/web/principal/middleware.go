package principal

import (
	"context"
	"net/http"

	sharedhttpx "github.com/louisbranch/fracturing.space/internal/services/shared/httpx"
)

// Middleware seeds the request-scoped principal state used by the resolver.
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
