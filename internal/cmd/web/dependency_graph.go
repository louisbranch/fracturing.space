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

// dependencyRequirement describes one startup dependency dial and field wiring step.
type dependencyRequirement struct {
	name       string
	address    string
	required   bool
	capability string
	setInput   func(*web.PrincipalDependencies, *modules.Dependencies, *grpc.ClientConn)
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
		{
			name:       dependencyNameAuth,
			address:    cfg.AuthAddr,
			required:   true,
			capability: "web.auth.integration",
			setInput: func(p *web.PrincipalDependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
				authClient := authv1.NewAuthServiceClient(conn)
				accountClient := authv1.NewAccountServiceClient(conn)
				p.SessionClient = authClient
				p.AccountClient = accountClient
				m.AuthClient = authClient
				m.AccountClient = accountClient
			},
		},
		{
			name:       dependencyNameSocial,
			address:    cfg.SocialAddr,
			required:   true,
			capability: "web.social.integration",
			setInput: func(p *web.PrincipalDependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
				socialClient := socialv1.NewSocialServiceClient(conn)
				p.SocialClient = socialClient
				m.ProfileSocialClient = socialClient
				m.SettingsSocialClient = socialClient
			},
		},
		{
			name:       dependencyNameGame,
			address:    cfg.GameAddr,
			required:   true,
			capability: "web.game.integration",
			setInput: func(_ *web.PrincipalDependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
				m.CampaignClient = statev1.NewCampaignServiceClient(conn)
				m.ParticipantClient = statev1.NewParticipantServiceClient(conn)
				m.CharacterClient = statev1.NewCharacterServiceClient(conn)
				m.DaggerheartContentClient = daggerheartv1.NewDaggerheartContentServiceClient(conn)
				m.SessionClient = statev1.NewSessionServiceClient(conn)
				m.InviteClient = statev1.NewInviteServiceClient(conn)
				m.AuthorizationClient = statev1.NewAuthorizationServiceClient(conn)
			},
		},
		{
			name:       dependencyNameAI,
			address:    cfg.AIAddr,
			required:   false,
			capability: "web.ai.integration",
			setInput: func(_ *web.PrincipalDependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
				m.CredentialClient = aiv1.NewCredentialServiceClient(conn)
			},
		},
		{
			name:       dependencyNameDiscovery,
			address:    cfg.DiscoveryAddr,
			required:   false,
			capability: "web.discovery.integration",
			setInput: func(_ *web.PrincipalDependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
				m.DiscoveryClient = discoveryv1.NewDiscoveryServiceClient(conn)
			},
		},
		{
			name:       dependencyNameUserHub,
			address:    cfg.UserHubAddr,
			required:   false,
			capability: "web.userhub.integration",
			setInput: func(_ *web.PrincipalDependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
				m.UserHubClient = userhubv1.NewUserHubServiceClient(conn)
			},
		},
		{
			name:       dependencyNameNotifications,
			address:    cfg.NotificationsAddr,
			required:   false,
			capability: "web.notifications.integration",
			setInput: func(p *web.PrincipalDependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
				notificationClient := notificationsv1.NewNotificationServiceClient(conn)
				p.NotificationClient = notificationClient
				m.NotificationClient = notificationClient
			},
		},
	}
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
