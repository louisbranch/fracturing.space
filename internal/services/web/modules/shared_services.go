package modules

import (
	"log/slog"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/dashboardsync"
)

// sharedServices bundles cross-module runtime helpers that the registry
// constructs once and passes into owned module composition entrypoints.
type sharedServices struct {
	dashboardSync dashboardsync.Service
}

// newSharedServices centralizes shared helper construction so public and
// protected module sets do not rebuild the same runtime policy independently.
func newSharedServices(deps Dependencies, logger *slog.Logger) sharedServices {
	if deps.DashboardSync.UserHubControlClient == nil && deps.DashboardSync.GameEventClient == nil {
		return sharedServices{dashboardSync: dashboardsync.Noop{}}
	}
	return sharedServices{
		dashboardSync: dashboardsync.New(deps.DashboardSync.UserHubControlClient, deps.DashboardSync.GameEventClient, logger),
	}
}
