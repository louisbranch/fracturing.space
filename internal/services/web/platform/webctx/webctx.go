// Package webctx provides shared web request context helpers.
package webctx

import (
	"context"
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
)

// WithResolvedUserID returns request context enriched with resolved user metadata.
func WithResolvedUserID(r *http.Request, resolve module.ResolveUserID) context.Context {
	if r == nil {
		return context.Background()
	}
	ctx := r.Context()
	if resolve == nil {
		return ctx
	}
	userID := strings.TrimSpace(resolve(r))
	if userID == "" {
		return ctx
	}
	return grpcauthctx.WithUserID(ctx, userID)
}
