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

// startupDependencyPolicy defines whether missing connectivity blocks web
// startup or only degrades specific mounted surfaces.
type startupDependencyPolicy string

const (
	startupDependencyRequired startupDependencyPolicy = "required"
	startupDependencyOptional startupDependencyPolicy = "optional"
)

// managedConnMode maps the web startup policy to the underlying managed-conn
// behavior used during bootstrap.
func (p startupDependencyPolicy) managedConnMode() platformgrpc.ManagedConnMode {
	if p == startupDependencyRequired {
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
	policy     startupDependencyPolicy
	capability string
	surfaces   []string
	setInput   web.DependencyBinder
	onConnect  dependencyConnHook
}

// dependencyConnHook performs optional post-connect setup for one dependency.
type dependencyConnHook func(context.Context, *platformgrpc.ManagedConn)

// dependencyPolicySpec keeps command-layer availability policy separate from
// the service-owned dependency binder table.
type dependencyPolicySpec struct {
	name       string
	address    string
	policy     startupDependencyPolicy
	capability string
	surfaces   []string
	onConnect  dependencyConnHook
}

// dependencyRequirements returns startup requirements in stable dependency order.
func dependencyRequirements(cfg Config, reporter *platformstatus.Reporter) []dependencyRequirement {
	specs := []dependencyPolicySpec{
		{
			name:       web.DependencyNameAuth,
			address:    cfg.AuthAddr,
			policy:     startupDependencyRequired,
			capability: "web.auth.integration",
			surfaces:   []string{"principal", "publicauth", "profile", "settings"},
		},
		{
			name:       web.DependencyNameSocial,
			address:    cfg.SocialAddr,
			policy:     startupDependencyRequired,
			capability: "web.social.integration",
			surfaces:   []string{"principal", "profile", "settings", "campaigns"},
		},
		{
			name:       web.DependencyNameGame,
			address:    cfg.GameAddr,
			policy:     startupDependencyRequired,
			capability: "web.game.integration",
			surfaces:   []string{"campaigns", "dashboard-sync"},
		},
		{
			name:       web.DependencyNameAI,
			address:    cfg.AIAddr,
			policy:     startupDependencyOptional,
			capability: "web.ai.integration",
			surfaces:   []string{"settings.ai", "campaigns.ai"},
		},
		{
			name:       web.DependencyNameDiscovery,
			address:    cfg.DiscoveryAddr,
			policy:     startupDependencyOptional,
			capability: "web.discovery.integration",
			surfaces:   []string{"discovery"},
		},
		{
			name:       web.DependencyNameUserHub,
			address:    cfg.UserHubAddr,
			policy:     startupDependencyOptional,
			capability: "web.userhub.integration",
			surfaces:   []string{"dashboard", "dashboard-sync"},
		},
		{
			name:       web.DependencyNameNotifications,
			address:    cfg.NotificationsAddr,
			policy:     startupDependencyOptional,
			capability: "web.notifications.integration",
			surfaces:   []string{"principal", "notifications"},
		},
		{
			name:       web.DependencyNameStatus,
			address:    cfg.StatusAddr,
			policy:     startupDependencyOptional,
			capability: "web.status.integration",
			surfaces:   []string{"dashboard.health"},
			onConnect:  bindStatusReporter(reporter),
		},
	}

	requirements := make([]dependencyRequirement, 0, len(specs))
	for _, spec := range specs {
		descriptor, ok := web.LookupStartupDependencyDescriptor(spec.name)
		if !ok {
			panic(fmt.Sprintf("missing web startup dependency descriptor for %q", spec.name))
		}
		requirements = append(requirements, dependencyRequirement{
			name:       spec.name,
			address:    spec.address,
			policy:     spec.policy,
			capability: spec.capability,
			surfaces:   spec.surfaces,
			setInput:   descriptor.Bind,
			onConnect:  spec.onConnect,
		})
	}
	return requirements
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
			Mode:             dep.policy.managedConnMode(),
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
