package principal

import (
	"context"
	"net/http"
	"sync"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
)

// requestSnapshot caches principal lookups for one request so repeated handler
// and template resolution does not duplicate backend calls.
type requestSnapshot struct {
	userIDOnce         sync.Once
	userID             string
	viewerOnce         sync.Once
	viewer             module.Viewer
	languageOnce       sync.Once
	language           string
	accountProfileOnce sync.Once
	accountProfile     *authv1.AccountProfile
}

// snapshotContextKey keeps the request snapshot private to this package.
type snapshotContextKey struct{}

// contextFromRequest keeps nil-request behavior consistent across principal
// resolution.
func contextFromRequest(request *http.Request) context.Context {
	if request == nil {
		return context.Background()
	}
	return request.Context()
}

// snapshotFromRequest returns the per-request principal snapshot when the
// middleware has attached one.
func snapshotFromRequest(request *http.Request) *requestSnapshot {
	if request == nil {
		return nil
	}
	return snapshotFromContext(request.Context())
}

// snapshotFromContext returns the per-request principal snapshot when present.
func snapshotFromContext(ctx context.Context) *requestSnapshot {
	if ctx == nil {
		return nil
	}
	snapshot, _ := ctx.Value(snapshotContextKey{}).(*requestSnapshot)
	return snapshot
}
