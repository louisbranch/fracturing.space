package server

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/config"
	aiservice "github.com/louisbranch/fracturing.space/internal/services/ai/api/grpc/ai"
	"github.com/louisbranch/fracturing.space/internal/services/ai/secret"
	aisqlite "github.com/louisbranch/fracturing.space/internal/services/ai/storage/sqlite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

// serverEnv holds env-parsed configuration for the AI server.
type serverEnv struct {
	DBPath        string `env:"FRACTURING_SPACE_AI_DB_PATH"`
	EncryptionKey string `env:"FRACTURING_SPACE_AI_ENCRYPTION_KEY"`

	OpenAIOAuthAuthURL      string `env:"FRACTURING_SPACE_AI_OPENAI_OAUTH_AUTH_URL"`
	OpenAIOAuthTokenURL     string `env:"FRACTURING_SPACE_AI_OPENAI_OAUTH_TOKEN_URL"`
	OpenAIOAuthClientID     string `env:"FRACTURING_SPACE_AI_OPENAI_OAUTH_CLIENT_ID"`
	OpenAIOAuthClientSecret string `env:"FRACTURING_SPACE_AI_OPENAI_OAUTH_CLIENT_SECRET"`
	OpenAIOAuthRedirectURI  string `env:"FRACTURING_SPACE_AI_OPENAI_OAUTH_REDIRECT_URI"`
	OpenAIResponsesURL      string `env:"FRACTURING_SPACE_AI_OPENAI_RESPONSES_URL"`
}

func loadServerEnv() serverEnv {
	var cfg serverEnv
	_ = config.ParseEnv(&cfg)
	if cfg.DBPath == "" {
		cfg.DBPath = filepath.Join("data", "ai.db")
	}
	return cfg
}

func openAIOAuthConfigFromEnv() (*aiservice.OpenAIOAuthConfig, error) {
	env := loadServerEnv()
	authURL := strings.TrimSpace(env.OpenAIOAuthAuthURL)
	tokenURL := strings.TrimSpace(env.OpenAIOAuthTokenURL)
	clientID := strings.TrimSpace(env.OpenAIOAuthClientID)
	clientSecret := strings.TrimSpace(env.OpenAIOAuthClientSecret)
	redirectURI := strings.TrimSpace(env.OpenAIOAuthRedirectURI)

	required := map[string]string{
		"FRACTURING_SPACE_AI_OPENAI_OAUTH_AUTH_URL":      authURL,
		"FRACTURING_SPACE_AI_OPENAI_OAUTH_TOKEN_URL":     tokenURL,
		"FRACTURING_SPACE_AI_OPENAI_OAUTH_CLIENT_ID":     clientID,
		"FRACTURING_SPACE_AI_OPENAI_OAUTH_CLIENT_SECRET": clientSecret,
		"FRACTURING_SPACE_AI_OPENAI_OAUTH_REDIRECT_URI":  redirectURI,
	}

	setCount := 0
	missing := make([]string, 0, len(required))
	for key, value := range required {
		if value == "" {
			missing = append(missing, key)
			continue
		}
		setCount++
	}
	if setCount == 0 {
		return nil, nil
	}
	if setCount != len(required) {
		return nil, fmt.Errorf("partial OpenAI OAuth env config; missing: %s", strings.Join(missing, ", "))
	}

	// Keep provider secrets in-memory only; callers must never log this struct.
	return &aiservice.OpenAIOAuthConfig{
		AuthorizationURL: authURL,
		TokenURL:         tokenURL,
		ClientID:         clientID,
		ClientSecret:     clientSecret,
		RedirectURI:      redirectURI,
	}, nil
}

// Server hosts the AI service.
type Server struct {
	listener   net.Listener
	grpcServer *grpc.Server
	health     *health.Server
	store      *aisqlite.Store
	closeOnce  sync.Once
}

// New creates a configured AI server listening on the provided port.
func New(port int) (*Server, error) {
	return NewWithAddr(fmt.Sprintf(":%d", port))
}

// NewWithAddr creates a configured AI server listening on the provided address.
func NewWithAddr(addr string) (*Server, error) {
	srvEnv := loadServerEnv()

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", addr, err)
	}

	store, err := openAIStore(srvEnv.DBPath)
	if err != nil {
		_ = listener.Close()
		return nil, err
	}

	encryptionKey := strings.TrimSpace(srvEnv.EncryptionKey)
	if encryptionKey == "" {
		_ = listener.Close()
		_ = store.Close()
		// Refuse startup when key material is missing so secrets are never stored
		// without encryption.
		return nil, errors.New("FRACTURING_SPACE_AI_ENCRYPTION_KEY is required")
	}
	keyBytes, err := decodeBase64Key(encryptionKey)
	if err != nil {
		_ = listener.Close()
		_ = store.Close()
		return nil, fmt.Errorf("decode encryption key: %w", err)
	}

	sealer, err := secret.NewAESGCMSealer(keyBytes)
	if err != nil {
		_ = listener.Close()
		_ = store.Close()
		return nil, fmt.Errorf("build secret sealer: %w", err)
	}

	grpcServer := grpc.NewServer()
	service := aiservice.NewService(store, store, sealer)
	openAIOAuthConfig, err := openAIOAuthConfigFromEnv()
	if err != nil {
		_ = listener.Close()
		_ = store.Close()
		return nil, fmt.Errorf("load OpenAI OAuth config: %w", err)
	}
	if openAIOAuthConfig != nil {
		service.SetOpenAIOAuthAdapter(aiservice.NewOpenAIOAuthAdapter(*openAIOAuthConfig))
	}
	if strings.TrimSpace(srvEnv.OpenAIResponsesURL) != "" {
		service.SetOpenAIInvocationAdapter(aiservice.NewOpenAIInvokeAdapter(aiservice.OpenAIInvokeConfig{
			ResponsesURL: strings.TrimSpace(srvEnv.OpenAIResponsesURL),
		}))
	}
	healthServer := health.NewServer()
	aiv1.RegisterCredentialServiceServer(grpcServer, service)
	aiv1.RegisterAgentServiceServer(grpcServer, service)
	aiv1.RegisterInvocationServiceServer(grpcServer, service)
	aiv1.RegisterProviderGrantServiceServer(grpcServer, service)
	aiv1.RegisterAccessRequestServiceServer(grpcServer, service)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("ai.v1.CredentialService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("ai.v1.AgentService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("ai.v1.InvocationService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("ai.v1.ProviderGrantService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("ai.v1.AccessRequestService", grpc_health_v1.HealthCheckResponse_SERVING)

	return &Server{
		listener:   listener,
		grpcServer: grpcServer,
		health:     healthServer,
		store:      store,
	}, nil
}

// Addr returns the listener address for the AI server.
func (s *Server) Addr() string {
	if s == nil || s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// Run creates and serves an AI server until the context ends.
func Run(ctx context.Context, port int) error {
	server, err := New(port)
	if err != nil {
		return err
	}
	return server.Serve(ctx)
}

// RunWithAddr creates and serves an AI server until the context ends.
func RunWithAddr(ctx context.Context, addr string) error {
	server, err := NewWithAddr(addr)
	if err != nil {
		return err
	}
	return server.Serve(ctx)
}

// Serve starts the AI server and blocks until it stops or context ends.
func (s *Server) Serve(ctx context.Context) error {
	if s == nil {
		return errors.New("server is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	defer s.Close()

	log.Printf("ai server listening at %v", s.listener.Addr())
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- s.grpcServer.Serve(s.listener)
	}()

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
		if err == nil || errors.Is(err, grpc.ErrServerStopped) {
			return nil
		}
		return fmt.Errorf("serve gRPC: %w", err)
	}
}

// Close releases server resources.
func (s *Server) Close() {
	if s == nil {
		return
	}

	s.closeOnce.Do(func() {
		if s.health != nil {
			s.health.Shutdown()
		}
		if s.grpcServer != nil {
			s.grpcServer.Stop()
		}
		if s.listener != nil {
			if err := s.listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
				log.Printf("close ai listener: %v", err)
			}
		}
		if s.store != nil {
			if err := s.store.Close(); err != nil {
				log.Printf("close ai store: %v", err)
			}
		}
	})
}

func openAIStore(path string) (*aisqlite.Store, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create storage dir: %w", err)
		}
	}
	store, err := aisqlite.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open ai sqlite store: %w", err)
	}
	return store, nil
}

// decodeBase64Key accepts both raw and padded base64 encodings to reduce
// operational friction across secret managers while preserving exact key bytes.
func decodeBase64Key(value string) ([]byte, error) {
	key, rawErr := base64.RawStdEncoding.DecodeString(value)
	if rawErr == nil {
		return key, nil
	}
	key, stdErr := base64.StdEncoding.DecodeString(value)
	if stdErr == nil {
		return key, nil
	}
	return nil, rawErr
}
