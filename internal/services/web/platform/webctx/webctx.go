package webctx

import (
	"context"
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/userid"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
)

// WithResolvedUserID returns request context enriched with resolved user metadata.
func WithResolvedUserID(r *http.Request, resolve principal.UserIDFunc) context.Context {
	if r == nil {
		return context.Background()
	}
	ctx := r.Context()
	if resolve == nil {
		return ctx
	}
	userID := userid.Normalize(resolve(r))
	if userID == "" {
		return ctx
	}
	return grpcauthctx.WithUserID(ctx, userID)
}
