package web

import (
	"context"
	"fmt"

	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	"github.com/louisbranch/fracturing.space/internal/services/web"
)

// runtimeDependencies captures the fully assembled startup dependency graph
// passed into the web server, plus the managed connections that must be closed
// when the process stops.
type runtimeDependencies struct {
	bundle     web.DependencyBundle
	depsConns  managedConns
	statusConn closableManagedConn
}

// close releases all managed connections owned by the runtime dependency
// assembly.
func (r runtimeDependencies) close() {
	closeManagedConns(r.depsConns)
	closeManagedConn(r.statusConn, "status")
}

// bootstrapRuntimeDependencies assembles the runtime dependency graph used by
// the web server in one place so startup code does not mutate dependency
// bundles after bootstrap.
func bootstrapRuntimeDependencies(
	ctx context.Context,
	cfg Config,
	reporter *platformstatus.Reporter,
) (runtimeDependencies, error) {
	requirements := dependencyRequirements(cfg)
	bundle, conns, err := bootstrapDependencies(ctx, requirements, cfg.AssetBaseURL, reporter)
	if err != nil {
		return runtimeDependencies{}, fmt.Errorf("init web dependency graph: %w", err)
	}

	statusConn, statusClient := startStatusService(ctx, cfg.StatusAddr, reporter)
	bundle.Modules.Dashboard.StatusClient = statusClient

	return runtimeDependencies{
		bundle:     bundle,
		depsConns:  conns,
		statusConn: statusConn,
	}, nil
}
