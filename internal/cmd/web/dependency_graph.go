package web

import (
	"context"
	"fmt"
	"log"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	discoveryv1 "github.com/louisbranch/fracturing.space/api/gen/go/discovery/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	"github.com/louisbranch/fracturing.space/internal/services/web"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules"
	grpc "google.golang.org/grpc"
)

const (
	dependencyNameAuth          = "auth"
	dependencyNameSocial        = "social"
	dependencyNameGame          = "game"
	dependencyNameAI            = "ai"
	dependencyNameDiscovery     = "discovery"
	dependencyNameUserHub       = "userhub"
	dependencyNameNotifications = "notifications"
)

// newManagedConn wraps platformgrpc.NewManagedConn for testability.
var newManagedConn = platformgrpc.NewManagedConn

// dependencyInputSetter maps one connected dependency into principal/module bundles.
type dependencyInputSetter func(*web.PrincipalDependencies, *modules.Dependencies, *grpc.ClientConn)

// dependencyRequirement describes one startup dependency and its wiring step.
type dependencyRequirement struct {
	name       string
	address    string
	mode       platformgrpc.ManagedConnMode
	capability string
	setInput   dependencyInputSetter
}

// dependencyRequirements returns startup requirements in stable dependency order.
func dependencyRequirements(cfg Config) []dependencyRequirement {
	return []dependencyRequirement{
		dependencyRequirementAuth(cfg.AuthAddr),
		dependencyRequirementSocial(cfg.SocialAddr),
		dependencyRequirementGame(cfg.GameAddr),
		dependencyRequirementAI(cfg.AIAddr),
		dependencyRequirementDiscovery(cfg.DiscoveryAddr),
		dependencyRequirementUserHub(cfg.UserHubAddr),
		dependencyRequirementNotifications(cfg.NotificationsAddr),
	}
}

// dependencyRequirementAuth returns the auth dependency wiring contract.
func dependencyRequirementAuth(address string) dependencyRequirement {
	return dependencyRequirement{
		name:       dependencyNameAuth,
		address:    address,
		mode:       platformgrpc.ModeRequired,
		capability: "web.auth.integration",
		setInput:   setDependencyAuth,
	}
}

// dependencyRequirementSocial returns the social dependency wiring contract.
func dependencyRequirementSocial(address string) dependencyRequirement {
	return dependencyRequirement{
		name:       dependencyNameSocial,
		address:    address,
		mode:       platformgrpc.ModeRequired,
		capability: "web.social.integration",
		setInput:   setDependencySocial,
	}
}

// dependencyRequirementGame returns the game dependency wiring contract.
func dependencyRequirementGame(address string) dependencyRequirement {
	return dependencyRequirement{
		name:       dependencyNameGame,
		address:    address,
		mode:       platformgrpc.ModeRequired,
		capability: "web.game.integration",
		setInput:   setDependencyGame,
	}
}

// dependencyRequirementAI returns the AI dependency wiring contract.
func dependencyRequirementAI(address string) dependencyRequirement {
	return dependencyRequirement{
		name:       dependencyNameAI,
		address:    address,
		mode:       platformgrpc.ModeOptional,
		capability: "web.ai.integration",
		setInput:   setDependencyAI,
	}
}

// dependencyRequirementDiscovery returns the discovery dependency wiring contract.
func dependencyRequirementDiscovery(address string) dependencyRequirement {
	return dependencyRequirement{
		name:       dependencyNameDiscovery,
		address:    address,
		mode:       platformgrpc.ModeOptional,
		capability: "web.discovery.integration",
		setInput:   setDependencyDiscovery,
	}
}

// dependencyRequirementUserHub returns the userhub dependency wiring contract.
func dependencyRequirementUserHub(address string) dependencyRequirement {
	return dependencyRequirement{
		name:       dependencyNameUserHub,
		address:    address,
		mode:       platformgrpc.ModeOptional,
		capability: "web.userhub.integration",
		setInput:   setDependencyUserHub,
	}
}

// dependencyRequirementNotifications returns the notifications dependency wiring contract.
func dependencyRequirementNotifications(address string) dependencyRequirement {
	return dependencyRequirement{
		name:       dependencyNameNotifications,
		address:    address,
		mode:       platformgrpc.ModeOptional,
		capability: "web.notifications.integration",
		setInput:   setDependencyNotifications,
	}
}

// setDependencyAuth wires auth clients into principal and module bundles.
func setDependencyAuth(p *web.PrincipalDependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
	authClient := authv1.NewAuthServiceClient(conn)
	accountClient := authv1.NewAccountServiceClient(conn)
	p.SessionClient = authClient
	p.AccountClient = accountClient
	m.PublicAuth.AuthClient = authClient
	m.Settings.AccountClient = accountClient
}

// setDependencySocial wires social clients into principal and module bundles.
func setDependencySocial(p *web.PrincipalDependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
	socialClient := socialv1.NewSocialServiceClient(conn)
	p.SocialClient = socialClient
	m.Profile.SocialClient = socialClient
	m.Settings.SocialClient = socialClient
}

// setDependencyGame wires game clients into module bundles.
func setDependencyGame(_ *web.PrincipalDependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
	m.Campaigns.CampaignClient = statev1.NewCampaignServiceClient(conn)
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
func setDependencyAI(_ *web.PrincipalDependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
	m.Settings.CredentialClient = aiv1.NewCredentialServiceClient(conn)
	m.Settings.AgentClient = aiv1.NewAgentServiceClient(conn)
	m.Campaigns.AgentClient = aiv1.NewAgentServiceClient(conn)
}

// setDependencyDiscovery wires discovery clients into module bundles.
func setDependencyDiscovery(_ *web.PrincipalDependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
	m.Discovery.DiscoveryClient = discoveryv1.NewDiscoveryServiceClient(conn)
}

// setDependencyUserHub wires userhub clients into module bundles.
func setDependencyUserHub(_ *web.PrincipalDependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
	m.Dashboard.UserHubClient = userhubv1.NewUserHubServiceClient(conn)
	m.DashboardSync.UserHubControlClient = userhubv1.NewUserHubControlServiceClient(conn)
}

// setDependencyNotifications wires notifications clients into principal and module bundles.
func setDependencyNotifications(p *web.PrincipalDependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
	notificationClient := notificationsv1.NewNotificationServiceClient(conn)
	p.NotificationClient = notificationClient
	m.Notifications.NotificationClient = notificationClient
}

// bootstrapDependencies creates ManagedConns for each requirement and wires
// gRPC clients into the dependency bundle. Required deps block until healthy;
// optional deps return immediately.
func bootstrapDependencies(
	ctx context.Context,
	requirements []dependencyRequirement,
	assetBaseURL string,
	reporter *platformstatus.Reporter,
) (web.DependencyBundle, []*platformgrpc.ManagedConn, error) {
	principal := web.PrincipalDependencies{AssetBaseURL: assetBaseURL}
	modDeps := modules.Dependencies{AssetBaseURL: assetBaseURL}
	var conns []*platformgrpc.ManagedConn

	logf := func(format string, args ...any) {
		log.Printf(format, args...)
	}

	for _, dep := range requirements {
		if strings.TrimSpace(dep.address) == "" {
			continue
		}
		mc, err := newManagedConn(ctx, platformgrpc.ManagedConnConfig{
			Name:             dep.name,
			Addr:             dep.address,
			Mode:             dep.mode,
			Logf:             logf,
			StatusReporter:   reporter,
			StatusCapability: dep.capability,
		})
		if err != nil {
			closeManagedConns(conns)
			return web.DependencyBundle{}, nil, fmt.Errorf("dependency %s: %w", dep.name, err)
		}
		conns = append(conns, mc)
		dep.setInput(&principal, &modDeps, mc.Conn())
	}

	return web.DependencyBundle{Principal: principal, Modules: modDeps}, conns, nil
}

// closeManagedConns closes all ManagedConn instances.
func closeManagedConns(conns []*platformgrpc.ManagedConn) {
	for _, mc := range conns {
		closeManagedConn(mc, "dependency")
	}
}

// closeManagedConn nil-safely closes a ManagedConn with error logging.
func closeManagedConn(mc *platformgrpc.ManagedConn, name string) {
	if mc == nil {
		return
	}
	if err := mc.Close(); err != nil {
		log.Printf("close web %s managed conn: %v", name, err)
	}
}
