// Package web parses command config and boots the web service.
package web

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	listingv1 "github.com/louisbranch/fracturing.space/api/gen/go/listing/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	entrypoint "github.com/louisbranch/fracturing.space/internal/platform/cmd"
	"github.com/louisbranch/fracturing.space/internal/platform/discovery"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	"github.com/louisbranch/fracturing.space/internal/services/web"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	grpc "google.golang.org/grpc"
)

// Config holds command inputs for web startup.
type Config struct {
	HTTPAddr            string        `env:"FRACTURING_SPACE_WEB_HTTP_ADDR" envDefault:"localhost:8080"`
	ChatHTTPAddr        string        `env:"FRACTURING_SPACE_CHAT_HTTP_ADDR" envDefault:"localhost:8086"`
	TrustForwardedProto bool          `env:"FRACTURING_SPACE_WEB_TRUST_FORWARDED_PROTO" envDefault:"false"`
	AuthAddr            string        `env:"FRACTURING_SPACE_AUTH_ADDR"`
	SocialAddr          string        `env:"FRACTURING_SPACE_SOCIAL_ADDR"`
	GameAddr            string        `env:"FRACTURING_SPACE_GAME_ADDR"`
	AIAddr              string        `env:"FRACTURING_SPACE_AI_ADDR"`
	ListingAddr         string        `env:"FRACTURING_SPACE_LISTING_ADDR"`
	NotificationsAddr   string        `env:"FRACTURING_SPACE_NOTIFICATIONS_ADDR"`
	UserHubAddr         string        `env:"FRACTURING_SPACE_USERHUB_ADDR"`
	StatusAddr          string        `env:"FRACTURING_SPACE_STATUS_ADDR"`
	AssetBaseURL        string        `env:"FRACTURING_SPACE_ASSET_BASE_URL"`
	GRPCDialTimeout     time.Duration `env:"FRACTURING_SPACE_WEB_DIAL_TIMEOUT" envDefault:"2s"`
}

// ParseConfig parses environment and flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := entrypoint.ParseConfig(&cfg); err != nil {
		return Config{}, err
	}
	if cfg.GRPCDialTimeout <= 0 {
		cfg.GRPCDialTimeout = timeouts.GRPCDial
	}
	cfg.AuthAddr = discovery.OrDefaultGRPCAddr(cfg.AuthAddr, discovery.ServiceAuth)
	cfg.SocialAddr = discovery.OrDefaultGRPCAddr(cfg.SocialAddr, discovery.ServiceSocial)
	cfg.GameAddr = discovery.OrDefaultGRPCAddr(cfg.GameAddr, discovery.ServiceGame)
	cfg.AIAddr = discovery.OrDefaultGRPCAddr(cfg.AIAddr, discovery.ServiceAI)
	cfg.ListingAddr = discovery.OrDefaultGRPCAddr(cfg.ListingAddr, discovery.ServiceListing)
	cfg.NotificationsAddr = discovery.OrDefaultGRPCAddr(cfg.NotificationsAddr, discovery.ServiceNotifications)
	cfg.UserHubAddr = discovery.OrDefaultGRPCAddr(cfg.UserHubAddr, discovery.ServiceUserHub)
	cfg.StatusAddr = discovery.OrDefaultGRPCAddr(cfg.StatusAddr, discovery.ServiceStatus)

	fs.StringVar(&cfg.HTTPAddr, "http-addr", cfg.HTTPAddr, "HTTP listen address")
	fs.StringVar(&cfg.ChatHTTPAddr, "chat-http-addr", cfg.ChatHTTPAddr, "Chat HTTP listen address")
	fs.StringVar(&cfg.AuthAddr, "auth-addr", cfg.AuthAddr, "Auth service gRPC address")
	fs.StringVar(&cfg.SocialAddr, "social-addr", cfg.SocialAddr, "Social service gRPC address")
	fs.StringVar(&cfg.GameAddr, "game-addr", cfg.GameAddr, "Game service gRPC address")
	fs.StringVar(&cfg.AIAddr, "ai-addr", cfg.AIAddr, "AI service gRPC address")
	fs.StringVar(&cfg.ListingAddr, "listing-addr", cfg.ListingAddr, "Listing service gRPC address")
	fs.StringVar(&cfg.NotificationsAddr, "notifications-addr", cfg.NotificationsAddr, "Notifications service gRPC address")
	fs.StringVar(&cfg.UserHubAddr, "userhub-addr", cfg.UserHubAddr, "Userhub service gRPC address")
	fs.StringVar(&cfg.AssetBaseURL, "asset-base-url", cfg.AssetBaseURL, "Asset base URL for image delivery")
	fs.BoolVar(&cfg.TrustForwardedProto, "trust-forwarded-proto", cfg.TrustForwardedProto, "Trust X-Forwarded-Proto when resolving request scheme")
	if err := entrypoint.ParseArgs(fs, args); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// grpcDialer abstracts dependency dialing for startup tests.
type grpcDialer func(context.Context, string, time.Duration) (*grpc.ClientConn, error)

// dependencyRequirement describes one startup dependency dial and field wiring step.
type dependencyRequirement struct {
	name     string
	address  string
	setInput func(*web.PrincipalDependencies, *modules.Dependencies, *grpc.ClientConn)
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

// bootstrapDependencies dials service dependencies and maps connected clients
// into principal and module dependency bundles.
func bootstrapDependencies(
	ctx context.Context,
	cfg Config,
	dialer grpcDialer,
) (web.DependencyBundle, []*grpc.ClientConn, map[string]dependencyStatus, error) {
	var principal web.PrincipalDependencies
	principal.AssetBaseURL = cfg.AssetBaseURL
	var modDeps modules.Dependencies
	modDeps.AssetBaseURL = cfg.AssetBaseURL
	conns := []*grpc.ClientConn{}
	statuses := map[string]dependencyStatus{}

	deps := []dependencyRequirement{
		{
			name:    "auth",
			address: cfg.AuthAddr,
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
			name:    "social",
			address: cfg.SocialAddr,
			setInput: func(p *web.PrincipalDependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
				socialClient := socialv1.NewSocialServiceClient(conn)
				p.SocialClient = socialClient
				m.ProfileSocialClient = socialClient
				m.SettingsSocialClient = socialClient
			},
		},
		{
			name:    "game",
			address: cfg.GameAddr,
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
			name:    "ai",
			address: cfg.AIAddr,
			setInput: func(_ *web.PrincipalDependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
				m.CredentialClient = aiv1.NewCredentialServiceClient(conn)
			},
		},
		{
			name:    "listing",
			address: cfg.ListingAddr,
			setInput: func(_ *web.PrincipalDependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
				m.ListingClient = listingv1.NewCampaignListingServiceClient(conn)
			},
		},
		{
			name:    "userhub",
			address: cfg.UserHubAddr,
			setInput: func(_ *web.PrincipalDependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
				m.UserHubClient = userhubv1.NewUserHubServiceClient(conn)
			},
		},
		{
			name:    "notifications",
			address: cfg.NotificationsAddr,
			setInput: func(p *web.PrincipalDependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
				notificationClient := notificationsv1.NewNotificationServiceClient(conn)
				p.NotificationClient = notificationClient
				m.NotificationClient = notificationClient
			},
		},
	}

	for _, dep := range deps {
		status := dependencyStatus{
			Name:    dep.name,
			Address: dep.address,
			State:   dependencyDialStateConnected,
		}
		conn, err := dialer(ctx, dep.address, cfg.GRPCDialTimeout)
		if err != nil {
			status.State = dependencyDialStateDialFailed
			status.Detail = err.Error()
			statuses[dep.name] = status
			continue
		}
		if conn == nil {
			status.State = dependencyDialStateUnavailable
			statuses[dep.name] = status
			continue
		}
		dep.setInput(&principal, &modDeps, conn)
		conns = append(conns, conn)
		statuses[dep.name] = status
	}

	bundle := web.DependencyBundle{Principal: principal, Modules: modDeps}
	return bundle, conns, statuses, nil
}

// dependencyStatusWarnings maps status diagnostics into legacy warning strings.
func dependencyStatusWarnings(statuses map[string]dependencyStatus) []string {
	if len(statuses) == 0 {
		return nil
	}
	order := []string{"auth", "social", "game", "ai", "listing", "userhub", "notifications"}
	warnings := make([]string, 0, len(statuses))
	for _, name := range order {
		status, ok := statuses[name]
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
	_ time.Duration,
) (*grpc.ClientConn, error) {
	conn := platformgrpc.DialLenient(ctx, address, log.Printf)
	return conn, nil
}

// Run starts the web service process.
func Run(ctx context.Context, cfg Config) error {
	if err := catalog.ValidateEmbeddedCatalogManifests(); err != nil {
		return err
	}
	return entrypoint.RunWithTelemetry(ctx, entrypoint.ServiceWeb, func(context.Context) error {
		dependencies, dependencyConns, statuses, err := bootstrapDependencies(ctx, cfg, dialDependency)
		if err != nil {
			return fmt.Errorf("init web dependency graph: %w", err)
		}
		defer closeDependencyConnections(dependencyConns)
		for _, statusLine := range formatDependencyStatusLines(statuses) {
			log.Printf("web startup: %s", statusLine)
		}
		for _, warning := range dependencyStatusWarnings(statuses) {
			log.Printf("web startup: %s", warning)
		}

		// Status reporter.
		statusConn := platformgrpc.DialLenient(ctx, cfg.StatusAddr, log.Printf)
		if statusConn != nil {
			defer func() {
				if err := statusConn.Close(); err != nil {
					log.Printf("close status connection: %v", err)
				}
			}()
		}
		var statusClient statusv1.StatusServiceClient
		if statusConn != nil {
			statusClient = statusv1.NewStatusServiceClient(statusConn)
		}
		reporter := platformstatus.NewReporter("web", statusClient)
		for _, dep := range []string{"auth", "social", "game", "ai", "listing", "userhub", "notifications"} {
			capName := "web." + dep + ".integration"
			if s, ok := statuses[dep]; ok && s.State == dependencyDialStateConnected {
				reporter.Register(capName, platformstatus.Operational)
			} else {
				reporter.Register(capName, platformstatus.Unavailable)
			}
		}
		stopReporter := reporter.Start(ctx)
		defer stopReporter()

		// Share the status client with modules for dashboard health queries.
		dependencies.Modules.StatusClient = statusClient

		server, err := web.NewServer(ctx, web.Config{
			HTTPAddr:            cfg.HTTPAddr,
			ChatHTTPAddr:        cfg.ChatHTTPAddr,
			RequestSchemePolicy: requestmeta.SchemePolicy{TrustForwardedProto: cfg.TrustForwardedProto},
			Dependencies:        &dependencies,
		})
		if err != nil {
			return fmt.Errorf("init web server: %w", err)
		}
		defer server.Close()
		if err := server.ListenAndServe(ctx); err != nil {
			return fmt.Errorf("serve web: %w", err)
		}
		return nil
	})
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
