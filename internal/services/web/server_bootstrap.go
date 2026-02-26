package web

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	webcache "github.com/louisbranch/fracturing.space/internal/services/web/infra/cache"
	webgrpc "github.com/louisbranch/fracturing.space/internal/services/web/infra/grpc"
)

// newServerWithContext builds a configured web server.
func newServerWithContext(ctx context.Context, config Config) (*Server, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	httpAddr := strings.TrimSpace(config.HTTPAddr)
	if httpAddr == "" {
		return nil, errors.New("http address is required")
	}
	if strings.TrimSpace(config.AuthBaseURL) == "" {
		return nil, errors.New("auth base url is required")
	}
	if strings.TrimSpace(config.OAuthClientID) == "" {
		return nil, errors.New("oauth client id is required")
	}
	if strings.TrimSpace(config.CallbackURL) == "" {
		return nil, errors.New("oauth callback url is required")
	}
	if config.GRPCDialTimeout <= 0 {
		config.GRPCDialTimeout = timeouts.GRPCDial
	}

	cacheStore, err := webcache.OpenStore(config.CacheDBPath)
	if err != nil {
		return nil, err
	}

	var authClients webgrpc.AuthClients
	if strings.TrimSpace(config.AuthAddr) != "" {
		authClients, err = webgrpc.DialAuth(ctx, config.AuthAddr, config.GRPCDialTimeout)
		if err != nil {
			if cacheStore != nil {
				_ = cacheStore.Close()
			}
			return nil, fmt.Errorf("dial auth grpc: %w", err)
		}
	}
	var connectionsClients webgrpc.ConnectionsClients
	if strings.TrimSpace(config.ConnectionsAddr) != "" {
		connectionsClients, err = webgrpc.DialConnections(ctx, config.ConnectionsAddr, config.GRPCDialTimeout)
		if err != nil {
			log.Printf("connections gRPC dial failed, invite contact options disabled: %v", err)
		}
	}

	var gameClients webgrpc.GameClients
	if strings.TrimSpace(config.GameAddr) != "" {
		gameClients, err = webgrpc.DialGame(ctx, config.GameAddr, config.GRPCDialTimeout)
		if err != nil {
			log.Printf("game gRPC dial failed, campaign access checks disabled: %v", err)
		}
	}
	var notificationsClients webgrpc.NotificationsClients
	if strings.TrimSpace(config.NotificationsAddr) != "" {
		notificationsClients, err = webgrpc.DialNotifications(ctx, config.NotificationsAddr, config.GRPCDialTimeout)
		if err != nil {
			log.Printf("notifications gRPC dial failed, notifications routes disabled: %v", err)
		}
	}
	var aiClients webgrpc.AIClients
	if strings.TrimSpace(config.AIAddr) != "" {
		aiClients, err = webgrpc.DialAI(ctx, config.AIAddr, config.GRPCDialTimeout)
		if err != nil {
			log.Printf("ai gRPC dial failed, settings ai keys disabled: %v", err)
		}
	}
	var listingClients webgrpc.ListingClients
	if strings.TrimSpace(config.ListingAddr) != "" {
		listingClients, err = webgrpc.DialListing(ctx, config.ListingAddr, config.GRPCDialTimeout)
		if err != nil {
			log.Printf("listing gRPC dial failed, discovery routes in degraded mode: %v", err)
		}
	}
	campaignAccess := newCampaignAccessChecker(config, gameClients.ParticipantClient)
	serverHandler, err := NewHandlerWithCampaignAccess(config, authClients.AuthClient, handlerDependencies{
		campaignAccess:     campaignAccess,
		cacheStore:         cacheStore,
		accountClient:      authClients.AccountClient,
		connectionsClient:  connectionsClients.ConnectionsClient,
		credentialClient:   aiClients.CredentialClient,
		campaignClient:     gameClients.CampaignClient,
		eventClient:        gameClients.EventClient,
		sessionClient:      gameClients.SessionClient,
		participantClient:  gameClients.ParticipantClient,
		characterClient:    gameClients.CharacterClient,
		inviteClient:       gameClients.InviteClient,
		notificationClient: notificationsClients.NotificationClient,
		listingClient:      listingClients.ListingClient,
	})
	if err != nil {
		if cacheStore != nil {
			_ = cacheStore.Close()
		}
		return nil, fmt.Errorf("build handler: %w", err)
	}
	httpServer := &http.Server{
		Addr:              httpAddr,
		Handler:           serverHandler,
		ReadHeaderTimeout: timeouts.ReadHeader,
	}

	invalidationStop, invalidationDone := startCacheInvalidationWorker(cacheStore, gameClients.EventClient)
	campaignUpdateStop, campaignUpdateDone := startCampaignProjectionSubscriptionWorker(cacheStore, gameClients.EventClient)

	return &Server{
		httpAddr:                       httpAddr,
		httpServer:                     httpServer,
		authConn:                       authClients.Conn,
		connectionsConn:                connectionsClients.Conn,
		gameConn:                       gameClients.Conn,
		notificationsConn:              notificationsClients.Conn,
		aiConn:                         aiClients.Conn,
		listingConn:                    listingClients.Conn,
		cacheStore:                     cacheStore,
		cacheInvalidationDone:          invalidationDone,
		cacheInvalidationStop:          invalidationStop,
		campaignUpdateSubscriptionDone: campaignUpdateDone,
		campaignUpdateSubscriptionStop: campaignUpdateStop,
	}, nil
}

// listenAndServe runs the HTTP server until the context ends.
//
// On cancellation, it performs a bounded shutdown so in-flight requests
// are drained before hard close.
func (s *Server) listenAndServe(ctx context.Context) error {
	if s == nil {
		return errors.New("web server is nil")
	}
	if ctx == nil {
		return errors.New("context is required")
	}

	serveErr := make(chan error, 1)
	log.Printf("web login listening on %s", s.httpAddr)
	go func() {
		serveErr <- s.httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), timeouts.Shutdown)
		err := s.httpServer.Shutdown(shutdownCtx)
		cancel()
		if err != nil {
			return fmt.Errorf("shutdown http server: %w", err)
		}
		return nil
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve http: %w", err)
	}
}

// close releases any gRPC resources held by the server.
func (s *Server) close() {
	if s == nil {
		return
	}
	if s.cacheInvalidationStop != nil {
		s.cacheInvalidationStop()
	}
	if s.campaignUpdateSubscriptionStop != nil {
		s.campaignUpdateSubscriptionStop()
	}
	if s.cacheInvalidationDone != nil {
		<-s.cacheInvalidationDone
	}
	if s.campaignUpdateSubscriptionDone != nil {
		<-s.campaignUpdateSubscriptionDone
	}
	if s.authConn != nil {
		if err := s.authConn.Close(); err != nil {
			log.Printf("close auth gRPC connection: %v", err)
		}
	}
	if s.connectionsConn != nil {
		if err := s.connectionsConn.Close(); err != nil {
			log.Printf("close connections gRPC connection: %v", err)
		}
	}
	if s.gameConn != nil {
		if err := s.gameConn.Close(); err != nil {
			log.Printf("close game gRPC connection: %v", err)
		}
	}
	if s.notificationsConn != nil {
		if err := s.notificationsConn.Close(); err != nil {
			log.Printf("close notifications gRPC connection: %v", err)
		}
	}
	if s.aiConn != nil {
		if err := s.aiConn.Close(); err != nil {
			log.Printf("close ai gRPC connection: %v", err)
		}
	}
	if s.listingConn != nil {
		if err := s.listingConn.Close(); err != nil {
			log.Printf("close listing gRPC connection: %v", err)
		}
	}
	if s.cacheStore != nil {
		if err := s.cacheStore.Close(); err != nil {
			log.Printf("close web cache store: %v", err)
		}
	}
}
