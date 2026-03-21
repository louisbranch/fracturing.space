package dashboard

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"time"

	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	dashboardapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard/app"
	dashboardgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
)

// CompositionConfig owns the startup wiring required to construct the
// production dashboard module without leaking gateway internals into the
// registry package.
type CompositionConfig struct {
	Base          modulehandler.Base
	Logger        *slog.Logger
	UserHubClient dashboardgateway.UserHubClient
	StatusClient  statusv1.StatusServiceClient
}

// ProtectedSurfaceOptions carries the cross-cutting inputs the protected registry is
// allowed to pass into dashboard composition.
type ProtectedSurfaceOptions struct {
	Base   modulehandler.Base
	Logger *slog.Logger
}

// Compose builds the production dashboard module from area-owned startup
// dependencies.
func Compose(config CompositionConfig) module.Module {
	gateway := dashboardgateway.NewGRPCGateway(config.UserHubClient)
	return New(Config{
		Service: dashboardapp.NewService(
			gateway,
			config.Logger,
			StatusHealthProvider(config.StatusClient, config.Logger),
		),
		Base: config.Base,
	})
}

// ComposeProtected composes the protected dashboard surface from module-owned
// startup dependencies and shared cross-cutting inputs.
func ComposeProtected(options ProtectedSurfaceOptions, deps Dependencies) module.Module {
	return Compose(newCompositionConfig(options, deps))
}

// newCompositionConfig projects startup dependencies into dashboard composition input.
func newCompositionConfig(options ProtectedSurfaceOptions, deps Dependencies) CompositionConfig {
	return CompositionConfig{
		Base:          options.Base,
		Logger:        options.Logger,
		UserHubClient: deps.UserHubClient,
		StatusClient:  deps.StatusClient,
	}
}

// statusHealthTimeout caps a per-request status service query.
const statusHealthTimeout = 3 * time.Second

// StatusHealthProvider returns a HealthProvider that queries the status service
// on each dashboard load. Returns nil when no status client is available.
func StatusHealthProvider(client statusv1.StatusServiceClient, logger *slog.Logger) dashboardapp.HealthProvider {
	if client == nil {
		return nil
	}
	return func(ctx context.Context) []dashboardapp.ServiceHealthEntry {
		ctx, cancel := context.WithTimeout(ctx, statusHealthTimeout)
		defer cancel()
		resp, err := client.GetSystemStatus(ctx, &statusv1.GetSystemStatusRequest{})
		if err != nil {
			if logger != nil {
				logger.Warn("dashboard status health query failed", "error", err)
			}
			return nil
		}
		services := resp.GetServices()
		if len(services) == 0 {
			return nil
		}
		entries := make([]dashboardapp.ServiceHealthEntry, 0, len(services))
		for _, svc := range services {
			if svc == nil {
				continue
			}
			entries = append(entries, dashboardapp.ServiceHealthEntry{
				Label:     capitalizeService(strings.TrimSpace(svc.GetService())),
				Available: svc.GetAggregateStatus() == statusv1.CapabilityStatus_CAPABILITY_STATUS_OPERATIONAL,
			})
		}
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Label < entries[j].Label
		})
		return entries
	}
}

// capitalizeService keeps dashboard-owned status labels readable without
// leaking formatting helpers back into the module registry.
func capitalizeService(id string) string {
	if id == "" {
		return id
	}
	return strings.ToUpper(id[:1]) + id[1:]
}
