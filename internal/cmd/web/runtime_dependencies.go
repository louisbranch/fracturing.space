package web

import (
	"context"
	"fmt"
	"log/slog"

	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	"github.com/louisbranch/fracturing.space/internal/services/web"
)

// runtimeDependencies captures the fully assembled startup dependency graph
// passed into the web server, plus the managed connections that must be closed
// when the process stops.
type runtimeDependencies struct {
	bundle    web.DependencyBundle
	depsConns managedConns
}

// close releases all managed connections owned by the runtime dependency
// assembly.
func (r runtimeDependencies) close() {
	closeManagedConns(r.depsConns, slog.Default())
}

// bootstrapRuntimeDependencies assembles the runtime dependency graph used by
// the web server in one place so startup code does not mutate dependency
// bundles after bootstrap.
func bootstrapRuntimeDependencies(
	ctx context.Context,
	cfg Config,
	reporter *platformstatus.Reporter,
) (runtimeDependencies, error) {
	return bootstrapRuntimeDependenciesWithConnFactory(ctx, cfg, reporter, platformgrpc.NewManagedConn)
}

// bootstrapRuntimeDependenciesWithConnFactory assembles the runtime
// dependency graph while keeping the managed-connection factory injectable for
// tests.
func bootstrapRuntimeDependenciesWithConnFactory(
	ctx context.Context,
	cfg Config,
	reporter *platformstatus.Reporter,
	newConn managedConnFactory,
) (runtimeDependencies, error) {
	requirements, err := dependencyRequirements(cfg, reporter)
	if err != nil {
		return runtimeDependencies{}, fmt.Errorf("resolve web dependency requirements: %w", err)
	}
	bundle, conns, err := bootstrapDependencies(ctx, requirements, cfg.AssetBaseURL, reporter, slog.Default(), newConn)
	if err != nil {
		return runtimeDependencies{}, fmt.Errorf("init web dependency graph: %w", err)
	}

	return runtimeDependencies{
		bundle:    bundle,
		depsConns: conns,
	}, nil
}
