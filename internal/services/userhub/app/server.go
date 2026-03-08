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

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
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

const defaultPort = 8092

var (
	dialLenient = platformgrpc.DialLenient
	listenTCP   = net.Listen
)

// RuntimeConfig controls userhub service startup and dependency wiring.
type RuntimeConfig struct {
	Port              int
	GameAddr          string
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

	gameConn          *grpc.ClientConn
	socialConn        *grpc.ClientConn
	notificationsConn *grpc.ClientConn
	statusConn        *grpc.ClientConn

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

	gameConn := dialLenient(ctx, normalized.GameAddr, logf)
	socialConn := dialLenient(ctx, normalized.SocialAddr, logf)
	notificationsConn := dialLenient(ctx, normalized.NotificationsAddr, logf)

	// Build gateways — handle nil connections by passing nil clients.
	var gameGateway domain.GameGateway
	var eventClient gamev1.EventServiceClient
	if gameConn != nil {
		gameGateway = newGRPCGameGateway(
			gamev1.NewCampaignServiceClient(gameConn),
			gamev1.NewInviteServiceClient(gameConn),
		)
		eventClient = gamev1.NewEventServiceClient(gameConn)
	}

	var socialGateway domain.SocialGateway
	if socialConn != nil {
		socialGateway = newGRPCSocialGateway(socialv1.NewSocialServiceClient(socialConn))
	}

	var notificationsGateway domain.NotificationsGateway
	if notificationsConn != nil {
		notificationsGateway = newGRPCNotificationsGateway(notificationsv1.NewNotificationServiceClient(notificationsConn))
	}

	var retainCampaignDependency func(string)
	var releaseCampaignDependency func(string)
	var stopCampaignProjectionSync context.CancelFunc
	var campaignProjectionDone chan struct{}
	var domainService *domain.Service
	if eventClient != nil {
		retainCampaignDependency, releaseCampaignDependency, stopCampaignProjectionSync, campaignProjectionDone = startCampaignProjectionSubscriptionManager(
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
	}
	domainService = domain.NewService(gameGateway, socialGateway, notificationsGateway, domain.Config{
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
		closeConn(gameConn, "game")
		closeConn(socialConn, "social")
		closeConn(notificationsConn, "notifications")
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

	statusAddr := normalized.StatusAddr
	if strings.TrimSpace(statusAddr) == "" {
		statusAddr = serviceaddr.DefaultGRPCAddr(serviceaddr.ServiceStatus)
	}
	statusConn := dialLenient(ctx, statusAddr, logf)
	var statusClient statusv1.StatusServiceClient
	if statusConn != nil {
		statusClient = statusv1.NewStatusServiceClient(statusConn)
	}

	reporter := platformstatus.NewReporter("userhub", statusClient)
	reporter.Register("userhub.dashboard", capStatus(gameConn != nil))
	reporter.Register("userhub.game.integration", capStatus(gameConn != nil))
	reporter.Register("userhub.social.integration", capStatus(socialConn != nil))
	reporter.Register("userhub.notifications.integration", capStatus(notificationsConn != nil))

	return &Server{
		listener:                   listener,
		grpcServer:                 grpcServer,
		health:                     healthServer,
		reporter:                   reporter,
		gameConn:                   gameConn,
		socialConn:                 socialConn,
		notificationsConn:          notificationsConn,
		statusConn:                 statusConn,
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
		closeConn(s.statusConn, "status")
		closeConn(s.notificationsConn, "notifications")
		closeConn(s.socialConn, "social")
		closeConn(s.gameConn, "game")
	})
}

func normalizeRuntimeConfig(cfg RuntimeConfig) (RuntimeConfig, error) {
	if strings.TrimSpace(cfg.GameAddr) == "" {
		return RuntimeConfig{}, fmt.Errorf("game address is required")
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

func closeConn(conn *grpc.ClientConn, name string) {
	if conn == nil {
		return
	}
	if err := conn.Close(); err != nil {
		log.Printf("close %s connection: %v", name, err)
	}
}

func capStatus(available bool) platformstatus.CapabilityStatus {
	if available {
		return platformstatus.Operational
	}
	return platformstatus.Unavailable
}
