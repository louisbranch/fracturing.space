package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	authservice "github.com/louisbranch/fracturing.space/internal/services/auth/api/grpc/auth"
	"github.com/louisbranch/fracturing.space/internal/services/auth/oauth"
	authsqlite "github.com/louisbranch/fracturing.space/internal/services/auth/storage/sqlite"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

// Server hosts the auth service.
type Server struct {
	listener     net.Listener
	grpcServer   *grpc.Server
	health       *health.Server
	store        *authsqlite.Store
	httpListener net.Listener
	httpServer   *http.Server
	oauthStore   *oauth.Store
	oauthServer  *oauth.Server
}

// New creates a configured auth server listening on the provided port.
func New(port int, httpAddr string) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("listen on port %d: %w", port, err)
	}
	store, err := openAuthStore()
	if err != nil {
		_ = listener.Close()
		return nil, err
	}

	oauthStore := oauth.NewStore(store.DB())
	oauthConfig := oauth.LoadConfigFromEnv()
	if oauthConfig.Issuer == "" {
		oauthConfig.Issuer = defaultOAuthIssuer(httpAddr)
	}
	if err := bootstrapOAuthUsers(store, oauthStore, oauthConfig); err != nil {
		_ = listener.Close()
		_ = store.Close()
		return nil, err
	}

	var httpListener net.Listener
	var httpServer *http.Server
	var oauthServer *oauth.Server
	if strings.TrimSpace(httpAddr) != "" {
		httpListener, err = net.Listen("tcp", httpAddr)
		if err != nil {
			_ = listener.Close()
			_ = store.Close()
			return nil, fmt.Errorf("listen on http addr %s: %w", httpAddr, err)
		}
		mux := http.NewServeMux()
		oauthServer = oauth.NewServer(oauthConfig, oauthStore, store)
		oauthServer.RegisterRoutes(mux)
		httpServer = &http.Server{Handler: mux}
	}

	grpcServer := grpc.NewServer()
	authService := authservice.NewAuthService(store, store, oauthStore)
	statisticsService := authservice.NewStatisticsService(store)
	healthServer := health.NewServer()
	authv1.RegisterAuthServiceServer(grpcServer, authService)
	authv1.RegisterStatisticsServiceServer(grpcServer, statisticsService)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("auth.v1.AuthService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("auth.v1.StatisticsService", grpc_health_v1.HealthCheckResponse_SERVING)

	return &Server{
		listener:     listener,
		grpcServer:   grpcServer,
		health:       healthServer,
		store:        store,
		httpListener: httpListener,
		httpServer:   httpServer,
		oauthStore:   oauthStore,
		oauthServer:  oauthServer,
	}, nil
}

// Addr returns the listener address for the auth server.
func (s *Server) Addr() string {
	if s == nil || s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// Run creates and serves an auth server until the context ends.
func Run(ctx context.Context, port int, httpAddr string) error {
	grpcServer, err := New(port, httpAddr)
	if err != nil {
		return err
	}
	return grpcServer.Serve(ctx)
}

// Serve starts the auth server and blocks until it stops or the context ends.
func (s *Server) Serve(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	serverCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer s.closeStore()

	if s.oauthServer != nil {
		s.oauthServer.StartCleanup(serverCtx, 5*time.Minute)
	}

	log.Printf("auth server listening at %v", s.listener.Addr())
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- s.grpcServer.Serve(s.listener)
	}()

	httpErr := make(chan error, 1)
	if s.httpServer != nil && s.httpListener != nil {
		log.Printf("auth HTTP server listening at %v", s.httpListener.Addr())
		go func() {
			httpErr <- s.httpServer.Serve(s.httpListener)
		}()
	}

	handleErr := func(err error) error {
		if err == nil || errors.Is(err, grpc.ErrServerStopped) {
			return nil
		}
		return fmt.Errorf("serve gRPC: %w", err)
	}

	shutdownGRPC := func() {
		if s.health != nil {
			s.health.Shutdown()
		}
		s.grpcServer.GracefulStop()
	}
	shutdownHTTP := func() {
		if s.httpServer != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = s.httpServer.Shutdown(shutdownCtx)
		}
	}

	select {
	case <-ctx.Done():
		shutdownGRPC()
		shutdownHTTP()
		err := <-serveErr
		return handleErr(err)
	case err := <-serveErr:
		shutdownHTTP()
		return handleErr(err)
	case err := <-httpErr:
		if err == http.ErrServerClosed {
			return nil
		}
		shutdownGRPC()
		grpcErr := <-serveErr
		if handled := handleErr(grpcErr); handled != nil {
			return handled
		}
		return fmt.Errorf("serve HTTP: %w", err)
	}
}

func openAuthStore() (*authsqlite.Store, error) {
	path := strings.TrimSpace(os.Getenv("FRACTURING_SPACE_AUTH_DB_PATH"))
	if path == "" {
		path = filepath.Join("data", "auth.db")
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create storage dir: %w", err)
		}
	}

	store, err := authsqlite.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open auth sqlite store: %w", err)
	}
	return store, nil
}

func (s *Server) closeStore() {
	if s == nil {
		return
	}
	if s.store != nil {
		if err := s.store.Close(); err != nil {
			log.Printf("close auth store: %v", err)
		}
	}
}

func defaultOAuthIssuer(httpAddr string) string {
	addr := strings.TrimSpace(httpAddr)
	if addr == "" {
		return ""
	}
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return strings.TrimRight(addr, "/")
	}
	if strings.HasPrefix(addr, ":") {
		return "http://localhost" + addr
	}
	return "http://" + addr
}

func bootstrapOAuthUsers(store *authsqlite.Store, oauthStore *oauth.Store, config oauth.Config) error {
	if store == nil || oauthStore == nil {
		return nil
	}
	for _, bootstrap := range config.BootstrapUsers {
		username := strings.TrimSpace(bootstrap.Username)
		password := strings.TrimSpace(bootstrap.Password)
		displayName := strings.TrimSpace(bootstrap.DisplayName)
		if username == "" || password == "" || displayName == "" {
			continue
		}
		existing, err := oauthStore.GetOAuthUserByUsername(username)
		if err != nil {
			return fmt.Errorf("lookup oauth user: %w", err)
		}
		userID := ""
		if existing != nil {
			userID = existing.UserID
		}
		if userID == "" {
			created, err := user.CreateUser(user.CreateUserInput{DisplayName: displayName}, time.Now, nil)
			if err != nil {
				return fmt.Errorf("create oauth user: %w", err)
			}
			if err := store.PutUser(context.Background(), created); err != nil {
				return fmt.Errorf("store oauth user: %w", err)
			}
			userID = created.ID
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash oauth password: %w", err)
		}
		if err := oauthStore.UpsertOAuthUserCredentials(userID, username, string(hash), time.Now().UTC()); err != nil {
			return fmt.Errorf("store oauth credentials: %w", err)
		}
	}
	return nil
}
