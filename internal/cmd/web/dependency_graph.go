package web

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	"github.com/louisbranch/fracturing.space/internal/services/web"
)

// closableManagedConn is the shutdown contract used by web startup wiring.
type closableManagedConn interface {
	Close() error
}

// managedConns captures the connection slice contract used during dependency
// bootstrap so runtime assembly can own shutdown in one place.
type managedConns []*platformgrpc.ManagedConn

// managedConnMode maps the web startup policy to the underlying managed-conn
// behavior used during bootstrap.
func managedConnMode(policy web.StartupDependencyPolicy) platformgrpc.ManagedConnMode {
	if policy == web.StartupDependencyRequired {
		return platformgrpc.ModeRequired
	}
	return platformgrpc.ModeOptional
}

// managedConnFactory builds one managed backend connection during startup.
type managedConnFactory func(context.Context, platformgrpc.ManagedConnConfig) (*platformgrpc.ManagedConn, error)

// dependencyRequirement describes one startup dependency and its wiring step.
type dependencyRequirement struct {
	name       string
	address    string
	policy     web.StartupDependencyPolicy
	capability string
	surfaces   []string
	setInput   web.DependencyBinder
	onConnect  dependencyConnHook
}

// dependencyConnHook performs optional post-connect setup for one dependency.
type dependencyConnHook func(context.Context, *platformgrpc.ManagedConn)

// dependencyAddressResolver maps one command config to the backend address for
// a service-owned startup dependency descriptor.
type dependencyAddressResolver func(Config) string

var dependencyAddressResolvers = map[string]dependencyAddressResolver{
	web.DependencyNameAuth:          func(cfg Config) string { return cfg.AuthAddr },
	web.DependencyNameSocial:        func(cfg Config) string { return cfg.SocialAddr },
	web.DependencyNameGame:          func(cfg Config) string { return cfg.GameAddr },
	web.DependencyNameAI:            func(cfg Config) string { return cfg.AIAddr },
	web.DependencyNameDiscovery:     func(cfg Config) string { return cfg.DiscoveryAddr },
	web.DependencyNameUserHub:       func(cfg Config) string { return cfg.UserHubAddr },
	web.DependencyNameNotifications: func(cfg Config) string { return cfg.NotificationsAddr },
	web.DependencyNameStatus:        func(cfg Config) string { return cfg.StatusAddr },
}

// dependencyRequirements returns startup requirements in stable dependency
// order and fails fast when command-layer address wiring drifts from the
// service-owned descriptor table.
func dependencyRequirements(cfg Config, reporter *platformstatus.Reporter) ([]dependencyRequirement, error) {
	descriptors := web.StartupDependencyDescriptors()
	requirements := make([]dependencyRequirement, 0, len(descriptors))
	for _, descriptor := range descriptors {
		address, err := dependencyAddress(cfg, descriptor.Name)
		if err != nil {
			return nil, err
		}
		requirements = append(requirements, dependencyRequirement{
			name:       descriptor.Name,
			address:    address,
			policy:     descriptor.Policy,
			capability: descriptor.Capability,
			surfaces:   append([]string(nil), descriptor.Surfaces...),
			setInput:   descriptor.Bind,
			onConnect:  dependencyOnConnect(descriptor.Name, reporter),
		})
	}
	return requirements, nil
}

// dependencyAddress resolves the configured backend address for one
// service-owned startup dependency descriptor.
func dependencyAddress(cfg Config, name string) (string, error) {
	resolve, ok := dependencyAddressResolvers[name]
	if !ok {
		return "", fmt.Errorf("web startup dependency %q is missing a command-layer address resolver", strings.TrimSpace(name))
	}
	return resolve(cfg), nil
}

// dependencyOnConnect returns any late-binding hook that should run after one
// dependency connects, keeping those side effects out of the descriptor table.
func dependencyOnConnect(name string, reporter *platformstatus.Reporter) dependencyConnHook {
	if name == web.DependencyNameStatus {
		return bindStatusReporter(reporter)
	}
	return nil
}

// bindStatusReporter late-binds the status reporter client once the connection
// becomes healthy.
func bindStatusReporter(reporter *platformstatus.Reporter) dependencyConnHook {
	if reporter == nil {
		return nil
	}
	return func(ctx context.Context, mc *platformgrpc.ManagedConn) {
		if mc == nil {
			return
		}
		client := statusv1.NewStatusServiceClient(mc.Conn())
		go func() {
			if mc.WaitReady(ctx) == nil {
				reporter.SetClient(client)
			}
		}()
	}
}

// bootstrapDependencies creates ManagedConns for each requirement and wires
// gRPC clients into the dependency bundle. Required deps block until healthy;
// optional deps return immediately.
func bootstrapDependencies(
	ctx context.Context,
	requirements []dependencyRequirement,
	assetBaseURL string,
	reporter *platformstatus.Reporter,
	logger *slog.Logger,
	newConn managedConnFactory,
) (web.DependencyBundle, managedConns, error) {
	bundle := web.NewDependencyBundle(assetBaseURL)
	var conns managedConns
	logger = defaultLogger(logger)
	if newConn == nil {
		newConn = platformgrpc.NewManagedConn
	}

	logf := func(format string, args ...any) {
		logger.Info(fmt.Sprintf(format, args...))
	}

	for _, dep := range requirements {
		if strings.TrimSpace(dep.address) == "" {
			continue
		}
		mc, err := newConn(ctx, platformgrpc.ManagedConnConfig{
			Name:             dep.name,
			Addr:             dep.address,
			Mode:             managedConnMode(dep.policy),
			Logf:             logf,
			StatusReporter:   reporter,
			StatusCapability: dep.capability,
		})
		if err != nil {
			closeManagedConns(conns, logger)
			return web.DependencyBundle{}, nil, fmt.Errorf("dependency %s: %w", dep.name, err)
		}
		conns = append(conns, mc)
		if dep.setInput != nil {
			dep.setInput(&bundle, mc.Conn())
		}
		if dep.onConnect != nil {
			dep.onConnect(ctx, mc)
		}
	}

	return bundle, conns, nil
}

// defaultLogger normalizes nil logger inputs to the process default logger.
func defaultLogger(logger *slog.Logger) *slog.Logger {
	if logger == nil {
		return slog.Default()
	}
	return logger
}

// closeManagedConns closes all ManagedConn instances.
func closeManagedConns(conns managedConns, logger *slog.Logger) {
	for _, mc := range conns {
		closeManagedConn(mc, "dependency", logger)
	}
}

// closeManagedConn nil-safely closes a ManagedConn with error logging.
func closeManagedConn(mc closableManagedConn, name string, logger *slog.Logger) {
	if mc == nil {
		return
	}
	if err := mc.Close(); err != nil {
		defaultLogger(logger).Error("close web managed conn", "name", name, "error", err)
	}
}
