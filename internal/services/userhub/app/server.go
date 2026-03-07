package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
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

// Run starts the userhub gRPC runtime until context cancellation.
//
// Dependencies are dialed leniently — the server starts immediately even if
// game, social, or notifications are unavailable. The domain layer's existing
// stale-cache and DegradedDependencies pattern handles runtime unavailability.
func Run(ctx context.Context, cfg RuntimeConfig) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(cfg.GameAddr) == "" {
		return fmt.Errorf("game address is required")
	}
	if strings.TrimSpace(cfg.SocialAddr) == "" {
		return fmt.Errorf("social address is required")
	}
	if strings.TrimSpace(cfg.NotificationsAddr) == "" {
		return fmt.Errorf("notifications address is required")
	}
	if cfg.Port <= 0 {
		cfg.Port = defaultPort
	}

	// Lenient dials — nil conn is tolerated, gateways handle it gracefully.
	logf := func(format string, args ...any) {
		log.Printf(format, args...)
	}
	gameConn := platformgrpc.DialLenient(ctx, cfg.GameAddr, logf)
	defer closeConn(gameConn, "game")

	socialConn := platformgrpc.DialLenient(ctx, cfg.SocialAddr, logf)
	defer closeConn(socialConn, "social")

	notificationsConn := platformgrpc.DialLenient(ctx, cfg.NotificationsAddr, logf)
	defer closeConn(notificationsConn, "notifications")

	// Build gateways — handle nil connections by passing nil clients.
	var gameGateway domain.GameGateway
	if gameConn != nil {
		gameGateway = newGRPCGameGateway(
			gamev1.NewCampaignServiceClient(gameConn),
			gamev1.NewInviteServiceClient(gameConn),
		)
	}

	var socialGateway domain.SocialGateway
	if socialConn != nil {
		socialGateway = newGRPCSocialGateway(socialv1.NewSocialServiceClient(socialConn))
	}

	var notificationsGateway domain.NotificationsGateway
	if notificationsConn != nil {
		notificationsGateway = newGRPCNotificationsGateway(notificationsv1.NewNotificationServiceClient(notificationsConn))
	}

	domainService := domain.NewService(gameGateway, socialGateway, notificationsGateway, domain.Config{
		CacheFreshTTL: cfg.CacheFreshTTL,
		CacheStaleTTL: cfg.CacheStaleTTL,
	})
	apiService := userhubservice.NewService(domainService)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Port))
	if err != nil {
		return fmt.Errorf("listen on userhub port %d: %w", cfg.Port, err)
	}
	defer func() {
		if closeErr := listener.Close(); closeErr != nil {
			log.Printf("close userhub listener: %v", closeErr)
		}
	}()

	grpcServer := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))
	healthServer := health.NewServer()
	userhubv1.RegisterUserHubServiceServer(grpcServer, apiService)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("userhub.v1.UserHubService", grpc_health_v1.HealthCheckResponse_SERVING)

	// Status reporter.
	statusAddr := cfg.StatusAddr
	if statusAddr == "" {
		statusAddr = serviceaddr.DefaultGRPCAddr(serviceaddr.ServiceStatus)
	}
	statusConn := platformgrpc.DialLenient(ctx, statusAddr, logf)
	defer closeConn(statusConn, "status")

	var statusClient statusv1.StatusServiceClient
	if statusConn != nil {
		statusClient = statusv1.NewStatusServiceClient(statusConn)
	}
	reporter := platformstatus.NewReporter("userhub", statusClient)
	reporter.Register("userhub.dashboard", capStatus(gameConn != nil))
	reporter.Register("userhub.game.integration", capStatus(gameConn != nil))
	reporter.Register("userhub.social.integration", capStatus(socialConn != nil))
	reporter.Register("userhub.notifications.integration", capStatus(notificationsConn != nil))
	stopReporter := reporter.Start(ctx)
	defer stopReporter()

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- grpcServer.Serve(listener)
	}()

	log.Printf("userhub server listening at %v", listener.Addr())

	select {
	case <-ctx.Done():
		healthServer.Shutdown()
		grpcServer.GracefulStop()
		err := <-serveErr
		if err == nil || errors.Is(err, grpc.ErrServerStopped) {
			return nil
		}
		return fmt.Errorf("serve gRPC: %w", err)
	case err := <-serveErr:
		healthServer.Shutdown()
		if err == nil || errors.Is(err, grpc.ErrServerStopped) {
			return nil
		}
		return fmt.Errorf("serve gRPC: %w", err)
	}
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
