package web

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	discoveryv1 "github.com/louisbranch/fracturing.space/api/gen/go/discovery/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	"github.com/louisbranch/fracturing.space/internal/services/web"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
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

// dependencyInputSetter maps one connected dependency into principal/module bundles.
type dependencyInputSetter func(*principal.Dependencies, *modules.Dependencies, *grpc.ClientConn)

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
		setInput:   setDependencyAuth,
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
		setInput:   setDependencySocial,
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
		setInput:   setDependencyGame,
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
		setInput:   setDependencyAI,
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
		setInput:   setDependencyDiscovery,
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
		setInput:   setDependencyUserHub,
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
		setInput:   setDependencyNotifications,
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
		setInput:   setDependencyStatus,
		onConnect:  bindStatusReporter(reporter),
	}
}

// setDependencyAuth wires auth clients into principal and module bundles.
func setDependencyAuth(p *principal.Dependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
	authClient := authv1.NewAuthServiceClient(conn)
	accountClient := authv1.NewAccountServiceClient(conn)
	p.SessionClient = authClient
	p.AccountClient = accountClient
	m.PublicAuth.AuthClient = authClient
	m.Campaigns.AuthClient = authClient
	m.Profile.AuthClient = authClient
	m.Settings.AccountClient = accountClient
	m.Settings.PasskeyClient = authClient
}

// setDependencySocial wires social clients into principal and module bundles.
func setDependencySocial(p *principal.Dependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
	socialClient := socialv1.NewSocialServiceClient(conn)
	p.SocialClient = socialClient
	m.Campaigns.SocialClient = socialClient
	m.Profile.SocialClient = socialClient
	m.Settings.SocialClient = socialClient
}

// setDependencyGame wires game clients into module bundles.
func setDependencyGame(_ *principal.Dependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
	m.Campaigns.CampaignClient = statev1.NewCampaignServiceClient(conn)
	m.Campaigns.CommunicationClient = statev1.NewCommunicationServiceClient(conn)
	m.Campaigns.ParticipantClient = statev1.NewParticipantServiceClient(conn)
	m.Campaigns.CharacterClient = statev1.NewCharacterServiceClient(conn)
	m.Campaigns.DaggerheartContentClient = daggerheartv1.NewDaggerheartContentServiceClient(conn)
	m.Campaigns.DaggerheartAssetClient = daggerheartv1.NewDaggerheartAssetServiceClient(conn)
	m.Campaigns.SessionClient = statev1.NewSessionServiceClient(conn)
	m.Campaigns.InviteClient = statev1.NewInviteServiceClient(conn)
	m.Campaigns.AuthorizationClient = statev1.NewAuthorizationServiceClient(conn)
	m.DashboardSync.GameEventClient = statev1.NewEventServiceClient(conn)
}

// setDependencyAI wires AI clients into module bundles.
func setDependencyAI(_ *principal.Dependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
	m.Settings.CredentialClient = aiv1.NewCredentialServiceClient(conn)
	m.Settings.AgentClient = aiv1.NewAgentServiceClient(conn)
	m.Campaigns.AgentClient = aiv1.NewAgentServiceClient(conn)
}

// setDependencyDiscovery wires discovery clients into module bundles.
func setDependencyDiscovery(_ *principal.Dependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
	m.Discovery.DiscoveryClient = discoveryv1.NewDiscoveryServiceClient(conn)
}

// setDependencyUserHub wires userhub clients into module bundles.
func setDependencyUserHub(_ *principal.Dependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
	m.Dashboard.UserHubClient = userhubv1.NewUserHubServiceClient(conn)
	m.DashboardSync.UserHubControlClient = userhubv1.NewUserHubControlServiceClient(conn)
}

// setDependencyNotifications wires notifications clients into principal and module bundles.
func setDependencyNotifications(p *principal.Dependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
	notificationClient := notificationsv1.NewNotificationServiceClient(conn)
	p.NotificationClient = notificationClient
	m.Notifications.NotificationClient = notificationClient
}

// setDependencyStatus wires the status client into dashboard dependencies.
func setDependencyStatus(_ *principal.Dependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
	m.Dashboard.StatusClient = statusv1.NewStatusServiceClient(conn)
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
	principalDeps := principal.Dependencies{AssetBaseURL: assetBaseURL}
	modDeps := modules.Dependencies{AssetBaseURL: assetBaseURL}
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
			dep.setInput(&principalDeps, &modDeps, mc.Conn())
		}
		if dep.onConnect != nil {
			dep.onConnect(ctx, mc)
		}
	}

	return web.DependencyBundle{Principal: principalDeps, Modules: modDeps}, conns, nil
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
