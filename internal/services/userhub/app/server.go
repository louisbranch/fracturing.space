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
	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
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
	CacheFreshTTL     time.Duration
	CacheStaleTTL     time.Duration
	GRPCDialTimeout   time.Duration
}

// Run starts the userhub gRPC runtime until context cancellation.
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
	if cfg.GRPCDialTimeout <= 0 {
		cfg.GRPCDialTimeout = timeouts.GRPCDial
	}

	gameConn, err := platformgrpc.DialWithHealth(
		ctx,
		nil,
		cfg.GameAddr,
		cfg.GRPCDialTimeout,
		log.Printf,
		platformgrpc.DefaultClientDialOptions()...,
	)
	if err != nil {
		return fmt.Errorf("dial game service: %w", err)
	}
	defer func() {
		if closeErr := gameConn.Close(); closeErr != nil {
			log.Printf("close game connection: %v", closeErr)
		}
	}()

	socialConn, err := platformgrpc.DialWithHealth(
		ctx,
		nil,
		cfg.SocialAddr,
		cfg.GRPCDialTimeout,
		log.Printf,
		platformgrpc.DefaultClientDialOptions()...,
	)
	if err != nil {
		return fmt.Errorf("dial social service: %w", err)
	}
	defer func() {
		if closeErr := socialConn.Close(); closeErr != nil {
			log.Printf("close social connection: %v", closeErr)
		}
	}()

	notificationsConn, err := platformgrpc.DialWithHealth(
		ctx,
		nil,
		cfg.NotificationsAddr,
		cfg.GRPCDialTimeout,
		log.Printf,
		platformgrpc.DefaultClientDialOptions()...,
	)
	if err != nil {
		return fmt.Errorf("dial notifications service: %w", err)
	}
	defer func() {
		if closeErr := notificationsConn.Close(); closeErr != nil {
			log.Printf("close notifications connection: %v", closeErr)
		}
	}()

	gameGateway := newGRPCGameGateway(
		gamev1.NewCampaignServiceClient(gameConn),
		gamev1.NewInviteServiceClient(gameConn),
	)
	socialGateway := newGRPCSocialGateway(socialv1.NewSocialServiceClient(socialConn))
	notificationsGateway := newGRPCNotificationsGateway(notificationsv1.NewNotificationServiceClient(notificationsConn))
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
