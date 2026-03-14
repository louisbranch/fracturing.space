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
	grpc "google.golang.org/grpc"
)

// closableManagedConn is the shutdown contract used by web startup wiring.
type closableManagedConn interface {
	Close() error
}

// managedConns captures the connection slice contract used during dependency
// bootstrap so runtime assembly can own shutdown in one place.
type managedConns []*platformgrpc.ManagedConn

const (
	dependencyNameAuth          = "auth"
	dependencyNameSocial        = "social"
	dependencyNameGame          = "game"
	dependencyNameAI            = "ai"
	dependencyNameDiscovery     = "discovery"
	dependencyNameUserHub       = "userhub"
	dependencyNameNotifications = "notifications"
	dependencyNameStatus        = "status"
)

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

// newManagedConn wraps platformgrpc.NewManagedConn for testability.
var newManagedConn = platformgrpc.NewManagedConn

// dependencyInputSetter maps one connected dependency into the web runtime bundle.
type dependencyInputSetter func(*web.DependencyBundle, *grpc.ClientConn)

// dependencyRequirement describes one startup dependency and its wiring step.
type dependencyRequirement struct {
	name       string
	address    string
	policy     startupDependencyPolicy
	capability string
	surfaces   []string
	setInput   dependencyInputSetter
	onConnect  dependencyConnHook
}

// dependencyConnHook performs optional post-connect setup for one dependency.
type dependencyConnHook func(context.Context, *platformgrpc.ManagedConn)

// dependencyRequirements returns startup requirements in stable dependency order.
func dependencyRequirements(cfg Config, reporter *platformstatus.Reporter) []dependencyRequirement {
	return []dependencyRequirement{
		dependencyRequirementAuth(cfg.AuthAddr),
		dependencyRequirementSocial(cfg.SocialAddr),
		dependencyRequirementGame(cfg.GameAddr),
		dependencyRequirementAI(cfg.AIAddr),
		dependencyRequirementDiscovery(cfg.DiscoveryAddr),
		dependencyRequirementUserHub(cfg.UserHubAddr),
		dependencyRequirementNotifications(cfg.NotificationsAddr),
		dependencyRequirementStatus(cfg.StatusAddr, reporter),
	}
}

// dependencyRequirementAuth returns the auth dependency wiring contract.
func dependencyRequirementAuth(address string) dependencyRequirement {
	return dependencyRequirement{
		name:       dependencyNameAuth,
		address:    address,
		policy:     startupDependencyRequired,
		capability: "web.auth.integration",
		surfaces:   []string{"principal", "publicauth", "profile", "settings"},
		setInput:   web.BindAuthDependency,
	}
}

// dependencyRequirementSocial returns the social dependency wiring contract.
func dependencyRequirementSocial(address string) dependencyRequirement {
	return dependencyRequirement{
		name:       dependencyNameSocial,
		address:    address,
		policy:     startupDependencyRequired,
		capability: "web.social.integration",
		surfaces:   []string{"principal", "profile", "settings", "campaigns"},
		setInput:   web.BindSocialDependency,
	}
}

// dependencyRequirementGame returns the game dependency wiring contract.
func dependencyRequirementGame(address string) dependencyRequirement {
	return dependencyRequirement{
		name:       dependencyNameGame,
		address:    address,
		policy:     startupDependencyRequired,
		capability: "web.game.integration",
		surfaces:   []string{"campaigns", "dashboard-sync"},
		setInput:   web.BindGameDependency,
	}
}

// dependencyRequirementAI returns the AI dependency wiring contract.
func dependencyRequirementAI(address string) dependencyRequirement {
	return dependencyRequirement{
		name:       dependencyNameAI,
		address:    address,
		policy:     startupDependencyOptional,
		capability: "web.ai.integration",
		surfaces:   []string{"settings.ai", "campaigns.ai"},
		setInput:   web.BindAIDependency,
	}
}

// dependencyRequirementDiscovery returns the discovery dependency wiring contract.
func dependencyRequirementDiscovery(address string) dependencyRequirement {
	return dependencyRequirement{
		name:       dependencyNameDiscovery,
		address:    address,
		policy:     startupDependencyOptional,
		capability: "web.discovery.integration",
		surfaces:   []string{"discovery"},
		setInput:   web.BindDiscoveryDependency,
	}
}

// dependencyRequirementUserHub returns the userhub dependency wiring contract.
func dependencyRequirementUserHub(address string) dependencyRequirement {
	return dependencyRequirement{
		name:       dependencyNameUserHub,
		address:    address,
		policy:     startupDependencyOptional,
		capability: "web.userhub.integration",
		surfaces:   []string{"dashboard", "dashboard-sync"},
		setInput:   web.BindUserHubDependency,
	}
}

// dependencyRequirementNotifications returns the notifications dependency wiring contract.
func dependencyRequirementNotifications(address string) dependencyRequirement {
	return dependencyRequirement{
		name:       dependencyNameNotifications,
		address:    address,
		policy:     startupDependencyOptional,
		capability: "web.notifications.integration",
		surfaces:   []string{"principal", "notifications"},
		setInput:   web.BindNotificationsDependency,
	}
}

// dependencyRequirementStatus returns the status dependency wiring contract.
func dependencyRequirementStatus(address string, reporter *platformstatus.Reporter) dependencyRequirement {
	return dependencyRequirement{
		name:       dependencyNameStatus,
		address:    address,
		policy:     startupDependencyOptional,
		capability: "web.status.integration",
		surfaces:   []string{"dashboard.health"},
		setInput:   web.BindStatusDependency,
		onConnect:  bindStatusReporter(reporter),
	}
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
) (web.DependencyBundle, managedConns, error) {
	bundle := web.NewDependencyBundle(assetBaseURL)
	var conns managedConns
	logger = defaultLogger(logger)

	logf := func(format string, args ...any) {
		logger.Info(fmt.Sprintf(format, args...))
	}

	for _, dep := range requirements {
		if strings.TrimSpace(dep.address) == "" {
			continue
		}
		mc, err := newManagedConn(ctx, platformgrpc.ManagedConnConfig{
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
