package web

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/state/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// defaultGRPCDialTimeout caps the dial wait time for gRPC connections.
const defaultGRPCDialTimeout = 2 * time.Second

// defaultGRPCRetryDelay sets the initial wait time between gRPC dial attempts.
const defaultGRPCRetryDelay = 500 * time.Millisecond

// maxGRPCRetryDelay caps the backoff between gRPC dial attempts.
const maxGRPCRetryDelay = 10 * time.Second

// Config defines the inputs for the web server.
type Config struct {
	HTTPAddr        string
	GRPCAddr        string
	GRPCDialTimeout time.Duration
}

// Server hosts the web client HTTP server and optional gRPC connection.
type Server struct {
	httpAddr    string
	grpcAddr    string
	grpcClients *grpcClients
	httpServer  *http.Server
}

// grpcClients stores gRPC connections and clients for the web server.
type grpcClients struct {
	mu                sync.RWMutex
	conn              *grpc.ClientConn
	authClient        authv1.AuthServiceClient
	daggerheartClient daggerheartv1.DaggerheartServiceClient
	campaignClient    statev1.CampaignServiceClient
	sessionClient     statev1.SessionServiceClient
	characterClient   statev1.CharacterServiceClient
	participantClient statev1.ParticipantServiceClient
	snapshotClient    statev1.SnapshotServiceClient
	eventClient       statev1.EventServiceClient
}

// CampaignClient returns the current campaign client.
func (g *grpcClients) CampaignClient() statev1.CampaignServiceClient {
	if g == nil {
		return nil
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.campaignClient
}

// AuthClient returns the current auth client.
func (g *grpcClients) AuthClient() authv1.AuthServiceClient {
	if g == nil {
		return nil
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.authClient
}

// SessionClient returns the current session client.
func (g *grpcClients) SessionClient() statev1.SessionServiceClient {
	if g == nil {
		return nil
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.sessionClient
}

// CharacterClient returns the current character client.
func (g *grpcClients) CharacterClient() statev1.CharacterServiceClient {
	if g == nil {
		return nil
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.characterClient
}

// ParticipantClient returns the current participant client.
func (g *grpcClients) ParticipantClient() statev1.ParticipantServiceClient {
	if g == nil {
		return nil
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.participantClient
}

// SnapshotClient returns the current snapshot client.
func (g *grpcClients) SnapshotClient() statev1.SnapshotServiceClient {
	if g == nil {
		return nil
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.snapshotClient
}

// EventClient returns the current event client.
func (g *grpcClients) EventClient() statev1.EventServiceClient {
	if g == nil {
		return nil
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.eventClient
}

// HasConnection reports whether a gRPC connection is already set.
func (g *grpcClients) HasConnection() bool {
	if g == nil {
		return false
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.conn != nil
}

// Set stores the gRPC connection and clients.
func (g *grpcClients) Set(conn *grpc.ClientConn, authClient authv1.AuthServiceClient, daggerheartClient daggerheartv1.DaggerheartServiceClient, campaignClient statev1.CampaignServiceClient, sessionClient statev1.SessionServiceClient, characterClient statev1.CharacterServiceClient, participantClient statev1.ParticipantServiceClient, snapshotClient statev1.SnapshotServiceClient, eventClient statev1.EventServiceClient) {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.conn != nil {
		return
	}
	g.conn = conn
	g.authClient = authClient
	g.daggerheartClient = daggerheartClient
	g.campaignClient = campaignClient
	g.sessionClient = sessionClient
	g.characterClient = characterClient
	g.participantClient = participantClient
	g.snapshotClient = snapshotClient
	g.eventClient = eventClient
}

// Close releases any gRPC resources held by the clients.
func (g *grpcClients) Close() {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.conn == nil {
		return
	}
	if err := g.conn.Close(); err != nil {
		log.Printf("close web gRPC connection: %v", err)
	}
	g.conn = nil
}

// NewServer builds a configured web server.
func NewServer(ctx context.Context, config Config) (*Server, error) {
	httpAddr := strings.TrimSpace(config.HTTPAddr)
	if httpAddr == "" {
		return nil, errors.New("http address is required")
	}
	if config.GRPCDialTimeout <= 0 {
		config.GRPCDialTimeout = defaultGRPCDialTimeout
	}

	clients := &grpcClients{}
	if strings.TrimSpace(config.GRPCAddr) != "" {
		conn, authClient, daggerheartClient, campaignClient, sessionClient, characterClient, participantClient, snapshotClient, eventClient, err := dialGRPC(ctx, config)
		if err != nil {
			log.Printf("web gRPC dial failed: %v", err)
			go connectGRPCWithRetry(ctx, config, clients)
		} else {
			clients.Set(conn, authClient, daggerheartClient, campaignClient, sessionClient, characterClient, participantClient, snapshotClient, eventClient)
		}
	}

	handler := NewHandler(clients)
	httpServer := &http.Server{
		Addr:              httpAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &Server{
		httpAddr:    httpAddr,
		grpcAddr:    config.GRPCAddr,
		grpcClients: clients,
		httpServer:  httpServer,
	}, nil
}

// ListenAndServe runs the HTTP server until the context ends.
func (s *Server) ListenAndServe(ctx context.Context) error {
	if s == nil {
		return errors.New("web server is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	serveErr := make(chan error, 1)
	log.Printf("web listening on %s", s.httpAddr)
	go func() {
		serveErr <- s.httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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

// Close releases any gRPC resources held by the server.
func (s *Server) Close() {
	if s == nil || s.grpcClients == nil {
		return
	}
	s.grpcClients.Close()
}

// dialGRPC connects to the gRPC server and returns a client.
func dialGRPC(ctx context.Context, config Config) (*grpc.ClientConn, authv1.AuthServiceClient, daggerheartv1.DaggerheartServiceClient, statev1.CampaignServiceClient, statev1.SessionServiceClient, statev1.CharacterServiceClient, statev1.ParticipantServiceClient, statev1.SnapshotServiceClient, statev1.EventServiceClient, error) {
	grpcAddr := strings.TrimSpace(config.GRPCAddr)
	if grpcAddr == "" {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	dialCtx, cancel := context.WithTimeout(ctx, config.GRPCDialTimeout)
	defer cancel()

	conn, err := grpc.DialContext(
		dialCtx,
		grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}

	authClient := authv1.NewAuthServiceClient(conn)
	daggerheartClient := daggerheartv1.NewDaggerheartServiceClient(conn)
	campaignClient := statev1.NewCampaignServiceClient(conn)
	sessionClient := statev1.NewSessionServiceClient(conn)
	characterClient := statev1.NewCharacterServiceClient(conn)
	participantClient := statev1.NewParticipantServiceClient(conn)
	snapshotClient := statev1.NewSnapshotServiceClient(conn)
	eventClient := statev1.NewEventServiceClient(conn)
	return conn, authClient, daggerheartClient, campaignClient, sessionClient, characterClient, participantClient, snapshotClient, eventClient, nil
}

// connectGRPCWithRetry keeps dialing until a connection is established or context ends.
func connectGRPCWithRetry(ctx context.Context, config Config, clients *grpcClients) {
	if clients == nil {
		return
	}
	if strings.TrimSpace(config.GRPCAddr) == "" {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	retryDelay := defaultGRPCRetryDelay
	for {
		if ctx.Err() != nil {
			return
		}
		if clients.HasConnection() {
			return
		}
		conn, authClient, daggerheartClient, campaignClient, sessionClient, characterClient, participantClient, snapshotClient, eventClient, err := dialGRPC(ctx, config)
		if err == nil {
			clients.Set(conn, authClient, daggerheartClient, campaignClient, sessionClient, characterClient, participantClient, snapshotClient, eventClient)
			log.Printf("web gRPC connected to %s", config.GRPCAddr)
			return
		}
		log.Printf("web gRPC dial failed: %v", err)
		timer := time.NewTimer(retryDelay)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return
		}
		if retryDelay < maxGRPCRetryDelay {
			retryDelay *= 2
			if retryDelay > maxGRPCRetryDelay {
				retryDelay = maxGRPCRetryDelay
			}
		}
	}
}
