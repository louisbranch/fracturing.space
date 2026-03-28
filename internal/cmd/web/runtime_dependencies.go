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

// bootstrapOptions holds injectable overrides for runtime dependency assembly.
// Production callers leave all fields nil to use defaults; tests override
// NewConn and/or Descriptors for deterministic wiring.
type bootstrapOptions struct {
	NewConn     managedConnFactory
	Descriptors []web.StartupDependencyDescriptor
}

// bootstrapRuntimeDependencies assembles the runtime dependency graph used by
// the web server. Pass nil opts to use production defaults; tests supply
// overrides via bootstrapOptions.
func bootstrapRuntimeDependencies(
	ctx context.Context,
	cfg Config,
	reporter *platformstatus.Reporter,
	opts *bootstrapOptions,
) (runtimeDependencies, error) {
	newConn := platformgrpc.NewManagedConn
	descriptors := web.StartupDependencyDescriptors()
	if opts != nil {
		if opts.NewConn != nil {
			newConn = opts.NewConn
		}
		if opts.Descriptors != nil {
			descriptors = append([]web.StartupDependencyDescriptor(nil), opts.Descriptors...)
		}
	}

	requirements, err := dependencyRequirementsWithDescriptors(cfg, reporter, descriptors)
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
