package web

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	discoveryv1 "github.com/louisbranch/fracturing.space/api/gen/go/discovery/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
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

// grpcDialer abstracts dependency dialing for startup tests.
type grpcDialer func(context.Context, string, time.Duration) (*grpc.ClientConn, error)

// dependencyInputSetter maps one connected dependency into principal/module bundles.
type dependencyInputSetter func(*web.PrincipalDependencies, *modules.Dependencies, *grpc.ClientConn)

// dependencyRequirement describes one startup dependency dial and field wiring step.
type dependencyRequirement struct {
	name       string
	address    string
	required   bool
	capability string
	setInput   dependencyInputSetter
}

// dependencyDialState classifies dependency dial outcomes at startup.
type dependencyDialState string

const (
	dependencyDialStateConnected   dependencyDialState = "connected"
	dependencyDialStateDialFailed  dependencyDialState = "dial_failed"
	dependencyDialStateUnavailable dependencyDialState = "unavailable"
)

// dependencyStatus captures one dependency dial result for startup diagnostics.
type dependencyStatus struct {
	Name    string
	Address string
	State   dependencyDialState
	Detail  string
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
		required:   true,
		capability: "web.auth.integration",
		setInput:   setDependencyAuth,
	}
}

// dependencyRequirementSocial returns the social dependency wiring contract.
func dependencyRequirementSocial(address string) dependencyRequirement {
	return dependencyRequirement{
		name:       dependencyNameSocial,
		address:    address,
		required:   true,
		capability: "web.social.integration",
		setInput:   setDependencySocial,
	}
}

// dependencyRequirementGame returns the game dependency wiring contract.
func dependencyRequirementGame(address string) dependencyRequirement {
	return dependencyRequirement{
		name:       dependencyNameGame,
		address:    address,
		required:   true,
		capability: "web.game.integration",
		setInput:   setDependencyGame,
	}
}

// dependencyRequirementAI returns the AI dependency wiring contract.
func dependencyRequirementAI(address string) dependencyRequirement {
	return dependencyRequirement{
		name:       dependencyNameAI,
		address:    address,
		required:   false,
		capability: "web.ai.integration",
		setInput:   setDependencyAI,
	}
}

// dependencyRequirementDiscovery returns the discovery dependency wiring contract.
func dependencyRequirementDiscovery(address string) dependencyRequirement {
	return dependencyRequirement{
		name:       dependencyNameDiscovery,
		address:    address,
		required:   false,
		capability: "web.discovery.integration",
		setInput:   setDependencyDiscovery,
	}
}

// dependencyRequirementUserHub returns the userhub dependency wiring contract.
func dependencyRequirementUserHub(address string) dependencyRequirement {
	return dependencyRequirement{
		name:       dependencyNameUserHub,
		address:    address,
		required:   false,
		capability: "web.userhub.integration",
		setInput:   setDependencyUserHub,
	}
}

// dependencyRequirementNotifications returns the notifications dependency wiring contract.
func dependencyRequirementNotifications(address string) dependencyRequirement {
	return dependencyRequirement{
		name:       dependencyNameNotifications,
		address:    address,
		required:   false,
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

// bootstrapDependencies dials service dependencies and maps connected clients
// into principal and module dependency bundles.
func bootstrapDependencies(
	ctx context.Context,
	requirements []dependencyRequirement,
	assetBaseURL string,
	dialTimeout time.Duration,
	dialer grpcDialer,
) (web.DependencyBundle, []*grpc.ClientConn, map[string]dependencyStatus, error) {
	if dialer == nil {
		dialer = dialDependency
	}

	principal := web.PrincipalDependencies{AssetBaseURL: assetBaseURL}
	modDeps := modules.Dependencies{AssetBaseURL: assetBaseURL}
	conns := []*grpc.ClientConn{}
	statuses := map[string]dependencyStatus{}
	requiredMissing := make([]string, 0)

	for _, dep := range requirements {
		status := dependencyStatus{
			Name:    dep.name,
			Address: dep.address,
			State:   dependencyDialStateConnected,
		}
		conn, err := dialer(ctx, dep.address, dialTimeout)
		if err != nil {
			status.State = dependencyDialStateDialFailed
			status.Detail = err.Error()
			statuses[dep.name] = status
			if dep.required {
				requiredMissing = append(requiredMissing, dep.name)
			}
			continue
		}
		if conn == nil {
			status.State = dependencyDialStateUnavailable
			statuses[dep.name] = status
			if dep.required {
				requiredMissing = append(requiredMissing, dep.name)
			}
			continue
		}
		dep.setInput(&principal, &modDeps, conn)
		conns = append(conns, conn)
		statuses[dep.name] = status
	}

	bundle := web.DependencyBundle{Principal: principal, Modules: modDeps}
	if len(requiredMissing) > 0 {
		return bundle, conns, statuses, fmt.Errorf("required dependencies unavailable: %s", strings.Join(requiredMissing, ", "))
	}
	return bundle, conns, statuses, nil
}

// dependencyStatusWarnings maps startup diagnostics into stable warning strings.
func dependencyStatusWarnings(requirements []dependencyRequirement, statuses map[string]dependencyStatus) []string {
	if len(statuses) == 0 {
		return nil
	}
	warnings := make([]string, 0, len(statuses))
	for _, dep := range requirements {
		status, ok := statuses[dep.name]
		if !ok {
			continue
		}
		switch status.State {
		case dependencyDialStateDialFailed:
			if strings.TrimSpace(status.Detail) != "" {
				warnings = append(warnings, fmt.Sprintf("%s dependency at %s unavailable: %s", status.Name, status.Address, status.Detail))
				continue
			}
			warnings = append(warnings, fmt.Sprintf("%s dependency at %s unavailable", status.Name, status.Address))
		case dependencyDialStateUnavailable:
			warnings = append(warnings, fmt.Sprintf("%s dependency at %s unavailable", status.Name, status.Address))
		}
	}
	return warnings
}

// formatDependencyStatusLines renders stable startup diagnostics for each dependency.
func formatDependencyStatusLines(statuses map[string]dependencyStatus) []string {
	if len(statuses) == 0 {
		return nil
	}
	names := make([]string, 0, len(statuses))
	for name := range statuses {
		names = append(names, name)
	}
	sort.Strings(names)

	lines := make([]string, 0, len(names))
	for _, name := range names {
		status := statuses[name]
		line := fmt.Sprintf("dependency=%s state=%s address=%s", status.Name, status.State, status.Address)
		if strings.TrimSpace(status.Detail) != "" {
			line += fmt.Sprintf(" detail=%s", strings.TrimSpace(status.Detail))
		}
		lines = append(lines, line)
	}
	return lines
}

// closeDependencyConnections closes all successfully dialed dependency connections.
func closeDependencyConnections(conns []*grpc.ClientConn) {
	for _, conn := range conns {
		if conn == nil {
			continue
		}
		_ = conn.Close()
	}
}

// dialDependency dials one dependency endpoint leniently.
// Returns nil connection on failure instead of an error.
func dialDependency(
	ctx context.Context,
	address string,
	dialTimeout time.Duration,
) (*grpc.ClientConn, error) {
	conn := platformgrpc.DialLenientWithTimeout(ctx, address, dialTimeout, log.Printf)
	return conn, nil
}
