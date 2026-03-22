package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	invitev1 "github.com/louisbranch/fracturing.space/api/gen/go/invite/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	userhubservice "github.com/louisbranch/fracturing.space/internal/services/userhub/api/grpc/userhub"
	"github.com/louisbranch/fracturing.space/internal/services/userhub/domain"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

// newManagedConn wraps platformgrpc.NewManagedConn for testability.
var newManagedConn = platformgrpc.NewManagedConn

const defaultPort = 8092

var listenTCP = net.Listen

// RuntimeConfig controls userhub service startup and dependency wiring.
type RuntimeConfig struct {
	Port              int
	AuthAddr          string
	GameAddr          string
	InviteAddr        string
	SocialAddr        string
	NotificationsAddr string
	StatusAddr        string
	CacheFreshTTL     time.Duration
	CacheStaleTTL     time.Duration
}

// Server hosts the userhub runtime and dependency connections.
type Server struct {
	listener   net.Listener
	grpcServer *grpc.Server
	health     *health.Server

	reporter     *platformstatus.Reporter
	stopReporter func()

	authMc          *platformgrpc.ManagedConn
	gameMc          *platformgrpc.ManagedConn
	inviteMc        *platformgrpc.ManagedConn
	socialMc        *platformgrpc.ManagedConn
	notificationsMc *platformgrpc.ManagedConn
	statusMc        *platformgrpc.ManagedConn

	stopCampaignProjectionSync context.CancelFunc
	campaignProjectionDone     chan struct{}

	closeOnce sync.Once
}

// Run starts the userhub gRPC runtime until context cancellation.
//
// Dependencies are dialed leniently — the server starts immediately even if
// game, social, or notifications are unavailable. The domain layer's existing
// stale-cache and DegradedDependencies pattern handles runtime unavailability.
func Run(ctx context.Context, cfg RuntimeConfig) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	server, err := New(ctx, cfg)
	if err != nil {
		return err
	}
	return server.Serve(ctx)
}

// New builds a configured userhub runtime.
func New(ctx context.Context, cfg RuntimeConfig) (*Server, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	normalized, err := normalizeRuntimeConfig(cfg)
	if err != nil {
		return nil, err
	}

	logf := func(format string, args ...any) {
		log.Printf(format, args...)
	}

	// Status reporter — starts with nil client; bound later when statusMc is ready.
	reporter := platformstatus.NewReporter("userhub", nil)

	// Dial all dependencies via ManagedConn. Connections are non-nil
	// immediately; RPCs fail with Unavailable until the peer is up. The domain
	// layer's DegradedDependencies / stale-cache pattern handles this.
	authMc, err := newManagedConn(ctx, platformgrpc.ManagedConnConfig{
		Name:             "auth",
		Addr:             normalized.AuthAddr,
		Mode:             platformgrpc.ModeOptional,
		Logf:             logf,
		StatusReporter:   reporter,
		StatusCapability: "userhub.auth.integration",
	})
	if err != nil {
		return nil, fmt.Errorf("userhub: managed conn auth: %w", err)
	}

	gameMc, err := newManagedConn(ctx, platformgrpc.ManagedConnConfig{
		Name:             "game",
		Addr:             normalized.GameAddr,
		Mode:             platformgrpc.ModeOptional,
		Logf:             logf,
		StatusReporter:   reporter,
		StatusCapability: "userhub.game.integration",
	})
	if err != nil {
		return nil, fmt.Errorf("userhub: managed conn game: %w", err)
	}

	inviteMc, err := newManagedConn(ctx, platformgrpc.ManagedConnConfig{
		Name:             "invite",
		Addr:             normalized.InviteAddr,
		Mode:             platformgrpc.ModeOptional,
		Logf:             logf,
		StatusReporter:   reporter,
		StatusCapability: "userhub.invite.integration",
	})
	if err != nil {
		authMc.Close()
		gameMc.Close()
		return nil, fmt.Errorf("userhub: managed conn invite: %w", err)
	}

	socialMc, err := newManagedConn(ctx, platformgrpc.ManagedConnConfig{
		Name:             "social",
		Addr:             normalized.SocialAddr,
		Mode:             platformgrpc.ModeOptional,
		Logf:             logf,
		StatusReporter:   reporter,
		StatusCapability: "userhub.social.integration",
	})
	if err != nil {
		authMc.Close()
		gameMc.Close()
		inviteMc.Close()
		return nil, fmt.Errorf("userhub: managed conn social: %w", err)
	}

	notificationsMc, err := newManagedConn(ctx, platformgrpc.ManagedConnConfig{
		Name:             "notifications",
		Addr:             normalized.NotificationsAddr,
		Mode:             platformgrpc.ModeOptional,
		Logf:             logf,
		StatusReporter:   reporter,
		StatusCapability: "userhub.notifications.integration",
	})
	if err != nil {
		authMc.Close()
		gameMc.Close()
		inviteMc.Close()
		socialMc.Close()
		return nil, fmt.Errorf("userhub: managed conn notifications: %w", err)
	}

	// Build gateways — conn is always non-nil, RPCs fail gracefully until peer is up.
	authGateway := newGRPCAuthGateway(authv1.NewAuthServiceClient(authMc.Conn()))
	gameGateway := newGRPCGameGateway(
		gamev1.NewCampaignServiceClient(gameMc.Conn()),
		invitev1.NewInviteServiceClient(inviteMc.Conn()),
		gamev1.NewSessionServiceClient(gameMc.Conn()),
		gamev1.NewAuthorizationServiceClient(gameMc.Conn()),
	)
	eventClient := gamev1.NewEventServiceClient(gameMc.Conn())
	socialGateway := newGRPCSocialGateway(socialv1.NewSocialServiceClient(socialMc.Conn()))
	notificationsGateway := newGRPCNotificationsGateway(notificationsv1.NewNotificationServiceClient(notificationsMc.Conn()))

	var domainService *domain.Service
	retainCampaignDependency, releaseCampaignDependency, stopCampaignProjectionSync, campaignProjectionDone := startCampaignProjectionSubscriptionManager(
		ctx,
		eventClient,
		func(campaignID string) {
			if domainService == nil {
				return
			}
			if _, err := domainService.InvalidateDashboards(context.Background(), domain.InvalidateDashboardsInput{
				CampaignIDs: []string{campaignID},
				Reason:      "game.projection_applied",
			}); err != nil {
				log.Printf("userhub: invalidate dashboards for campaign %s: %v", campaignID, err)
			}
		},
	)
	domainService = domain.NewService(authGateway, gameGateway, socialGateway, notificationsGateway, domain.Config{
		CacheFreshTTL: normalized.CacheFreshTTL,
		CacheStaleTTL: normalized.CacheStaleTTL,
		CampaignDependencyObserver: campaignDependencyObserver{
			retain:  retainCampaignDependency,
			release: releaseCampaignDependency,
		},
	})
	apiService := userhubservice.NewService(domainService)
	controlService := userhubservice.NewControlService(domainService)

	listener, err := listenTCP("tcp", fmt.Sprintf(":%d", normalized.Port))
	if err != nil {
		authMc.Close()
		gameMc.Close()
		inviteMc.Close()
		socialMc.Close()
		notificationsMc.Close()
		return nil, fmt.Errorf("listen on userhub port %d: %w", normalized.Port, err)
	}

	grpcServer := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))
	healthServer := health.NewServer()
	userhubv1.RegisterUserHubServiceServer(grpcServer, apiService)
	userhubv1.RegisterUserHubControlServiceServer(grpcServer, controlService)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("userhub.v1.UserHubService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("userhub.v1.UserHubControlService", grpc_health_v1.HealthCheckResponse_SERVING)

	// Status service connection — optional, late-binds reporter when ready.
	statusAddr := normalized.StatusAddr
	if strings.TrimSpace(statusAddr) == "" {
		statusAddr = serviceaddr.DefaultGRPCAddr(serviceaddr.ServiceStatus)
	}
	statusMc, err := newManagedConn(ctx, platformgrpc.ManagedConnConfig{
		Name: "status",
		Addr: statusAddr,
		Mode: platformgrpc.ModeOptional,
		Logf: logf,
	})
	if err != nil {
		authMc.Close()
		gameMc.Close()
		inviteMc.Close()
		socialMc.Close()
		notificationsMc.Close()
		listener.Close()
		return nil, fmt.Errorf("userhub: managed conn status: %w", err)
	}

	// Dashboard capability tracks game readiness since it depends on game data.
	reporter.Register("userhub.dashboard", platformstatus.Unavailable)

	// Late-bind the reporter's client when the status service becomes reachable.
	go func() {
		if statusMc.WaitReady(ctx) == nil {
			reporter.SetClient(statusv1.NewStatusServiceClient(statusMc.Conn()))
		}
	}()

	return &Server{
		listener:                   listener,
		grpcServer:                 grpcServer,
		health:                     healthServer,
		reporter:                   reporter,
		authMc:                     authMc,
		gameMc:                     gameMc,
		inviteMc:                   inviteMc,
		socialMc:                   socialMc,
		notificationsMc:            notificationsMc,
		statusMc:                   statusMc,
		stopCampaignProjectionSync: stopCampaignProjectionSync,
		campaignProjectionDone:     campaignProjectionDone,
	}, nil
}

// Addr returns the listener address for the server.
func (s *Server) Addr() string {
	if s == nil || s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// Serve starts userhub serving until context cancellation or server failure.
func (s *Server) Serve(ctx context.Context) error {
	if s == nil {
		return errors.New("server is nil")
	}
	if ctx == nil {
		return errors.New("context is required")
	}
	defer s.Close()

	if s.reporter != nil {
		s.stopReporter = s.reporter.Start(ctx)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- s.grpcServer.Serve(s.listener)
	}()

	log.Printf("userhub server listening at %v", s.listener.Addr())

	select {
	case <-ctx.Done():
		if s.health != nil {
			s.health.Shutdown()
		}
		s.grpcServer.GracefulStop()
		err := <-serveErr
		if err == nil || errors.Is(err, grpc.ErrServerStopped) {
			return nil
		}
		return fmt.Errorf("serve gRPC: %w", err)
	case err := <-serveErr:
		if s.health != nil {
			s.health.Shutdown()
		}
		if err == nil || errors.Is(err, grpc.ErrServerStopped) {
			return nil
		}
		return fmt.Errorf("serve gRPC: %w", err)
	}
}

// Close releases all runtime resources.
func (s *Server) Close() {
	if s == nil {
		return
	}

	s.closeOnce.Do(func() {
		if s.stopReporter != nil {
			s.stopReporter()
		}
		if s.health != nil {
			s.health.Shutdown()
		}
		if s.grpcServer != nil {
			s.grpcServer.Stop()
		}
		if s.stopCampaignProjectionSync != nil {
			s.stopCampaignProjectionSync()
		}
		if s.campaignProjectionDone != nil {
			<-s.campaignProjectionDone
		}
		if s.listener != nil {
			if err := s.listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
				log.Printf("close userhub listener: %v", err)
			}
		}
		closeManagedConn(s.statusMc, "status")
		closeManagedConn(s.notificationsMc, "notifications")
		closeManagedConn(s.socialMc, "social")
		closeManagedConn(s.inviteMc, "invite")
		closeManagedConn(s.gameMc, "game")
		closeManagedConn(s.authMc, "auth")
	})
}

func normalizeRuntimeConfig(cfg RuntimeConfig) (RuntimeConfig, error) {
	if strings.TrimSpace(cfg.AuthAddr) == "" {
		return RuntimeConfig{}, fmt.Errorf("auth address is required")
	}
	if strings.TrimSpace(cfg.GameAddr) == "" {
		return RuntimeConfig{}, fmt.Errorf("game address is required")
	}
	if strings.TrimSpace(cfg.InviteAddr) == "" {
		return RuntimeConfig{}, fmt.Errorf("invite address is required")
	}
	if strings.TrimSpace(cfg.SocialAddr) == "" {
		return RuntimeConfig{}, fmt.Errorf("social address is required")
	}
	if strings.TrimSpace(cfg.NotificationsAddr) == "" {
		return RuntimeConfig{}, fmt.Errorf("notifications address is required")
	}
	if cfg.Port <= 0 {
		cfg.Port = defaultPort
	}
	return cfg, nil
}

func closeManagedConn(mc *platformgrpc.ManagedConn, name string) {
	if mc == nil {
		return
	}
	if err := mc.Close(); err != nil {
		log.Printf("close %s managed conn: %v", name, err)
	}
}
