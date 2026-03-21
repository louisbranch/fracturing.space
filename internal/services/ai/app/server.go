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
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	aiservice "github.com/louisbranch/fracturing.space/internal/services/ai/api/grpc/ai"
	"github.com/louisbranch/fracturing.space/internal/services/ai/campaigncontext"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration/gametools"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	openaiprovider "github.com/louisbranch/fracturing.space/internal/services/ai/provider/openai"
	"github.com/louisbranch/fracturing.space/internal/services/ai/secret"
	aisqlite "github.com/louisbranch/fracturing.space/internal/services/ai/storage/sqlite"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

var (
	newManagedConn = platformgrpc.NewManagedConn
	listenTCP      = net.Listen
)

// Server hosts the AI service and coordinates gRPC + health serving.
//
// It treats AI credential material as externalized secrets and never exposes
// decrypted values from the API layer.
type Server struct {
	listener   net.Listener
	grpcServer *grpc.Server
	health     *health.Server
	store      *aisqlite.Store
	gameMc     *platformgrpc.ManagedConn
	closeOnce  sync.Once
}

// New creates a configured AI server listening on the provided port.
func New(port int) (*Server, error) {
	return NewWithAddr(fmt.Sprintf(":%d", port))
}

// NewWithAddr creates a configured AI server listening on the provided address.
//
// This constructor keeps backward compatibility for call sites that do not pass
// startup context while NewWithAddrContext enables context-bound dependency dial.
func NewWithAddr(addr string) (*Server, error) {
	return NewWithAddrContext(context.Background(), addr)
}

// NewWithAddrContext creates a configured AI server using one startup context
// for dependency dialing and one parsed runtime config snapshot.
func NewWithAddrContext(ctx context.Context, addr string) (*Server, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	cfg, err := loadRuntimeConfigFromEnv()
	if err != nil {
		return nil, err
	}
	return newServerWithRuntimeConfig(ctx, addr, cfg)
}

func newServerWithRuntimeConfig(ctx context.Context, addr string, cfg runtimeConfig) (*Server, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}

	listener, err := listenTCP("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", addr, err)
	}

	store, err := openAIStore(cfg.DBPath)
	if err != nil {
		_ = listener.Close()
		return nil, err
	}

	encryptionKey := strings.TrimSpace(cfg.EncryptionKey)
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

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(serviceIdentityValidationUnaryInterceptor(cfg.InternalServiceAllowlist)),
		grpc.ChainStreamInterceptor(serviceIdentityValidationStreamInterceptor(cfg.InternalServiceAllowlist)),
	)
	providerOAuthAdapters := map[provider.Provider]aiservice.ProviderOAuthAdapter{
		provider.OpenAI: openaiprovider.NewDefaultOAuthAdapter(),
	}
	if cfg.OpenAIOAuthConfig != nil {
		providerOAuthAdapters[provider.OpenAI] = openaiprovider.NewOAuthAdapter(*cfg.OpenAIOAuthConfig)
	}
	openAIAdapter := openaiprovider.NewInvokeAdapter(openaiprovider.InvokeConfig{
		ResponsesURL: cfg.OpenAIResponsesURL,
	})
	providerInvocationAdapters := map[provider.Provider]aiservice.ProviderInvocationAdapter{
		provider.OpenAI: openAIAdapter,
	}
	providerToolAdapters := make(map[provider.Provider]orchestration.Provider, 1)
	if toolAdapter, ok := openAIAdapter.(orchestration.Provider); ok {
		providerToolAdapters[provider.OpenAI] = toolAdapter
	}
	providerModelAdapters := make(map[provider.Provider]aiservice.ProviderModelAdapter, 1)
	if modelAdapter, ok := openAIAdapter.(aiservice.ProviderModelAdapter); ok {
		providerModelAdapters[provider.OpenAI] = modelAdapter
	}
	instructionLoader := campaigncontext.NewInstructionLoader(cfg.InstructionsRoot)
	campaignArtifactManager := campaigncontext.NewManager(store, nil)
	campaignArtifactManager.SetInstructionLoader(instructionLoader)
	service := aiservice.NewService(aiservice.ServiceConfig{
		CredentialStore:            store,
		AgentStore:                 store,
		ProviderGrantStore:         store,
		ConnectSessionStore:        store,
		AccessRequestStore:         store,
		AuditEventStore:            store,
		CampaignArtifactStore:      store,
		CampaignArtifactManager:    campaignArtifactManager,
		ProviderOAuthAdapters:      providerOAuthAdapters,
		ProviderInvocationAdapters: providerInvocationAdapters,
		ProviderToolAdapters:       providerToolAdapters,
		ProviderModelAdapters:      providerModelAdapters,
		Sealer:                     sealer,
	})
	systemReferenceHandlers := aiservice.NewSystemReferenceHandlers(nil)
	if strings.TrimSpace(cfg.DaggerheartReferenceRoot) != "" {
		systemReferenceHandlers = aiservice.NewSystemReferenceHandlers(campaigncontext.NewReferenceCorpus(cfg.DaggerheartReferenceRoot))
	}

	var gameMc *platformgrpc.ManagedConn
	var gameCampaignAIClient gamev1.CampaignAIServiceClient
	var gameAuthorizationClient gamev1.AuthorizationServiceClient
	if gameAddr := strings.TrimSpace(cfg.GameAddr); gameAddr != "" {
		mc, err := newManagedConn(ctx, platformgrpc.ManagedConnConfig{
			Name: "game",
			Addr: gameAddr,
			Mode: platformgrpc.ModeOptional,
			Logf: log.Printf,
			DialOpts: append(
				platformgrpc.LenientDialOptions(),
				grpc.WithChainUnaryInterceptor(grpcauthctx.ServiceIDUnaryClientInterceptor(serviceaddr.ServiceAI)),
				grpc.WithChainStreamInterceptor(grpcauthctx.ServiceIDStreamClientInterceptor(serviceaddr.ServiceAI)),
			),
		})
		if err != nil {
			log.Printf("ai: game managed conn unavailable; agent usage guard disabled: %v", err)
		} else {
			gameMc = mc
			gameCampaignAIClient = gamev1.NewCampaignAIServiceClient(mc.Conn())
			service.SetGameCampaignAIClient(gameCampaignAIClient)
			gameAuthorizationClient = gamev1.NewAuthorizationServiceClient(mc.Conn())
		}
	}
	credentialHandlers := aiservice.NewCredentialHandlers(aiservice.CredentialHandlersConfig{
		CredentialStore:      store,
		AgentStore:           store,
		GameCampaignAIClient: gameCampaignAIClient,
		Sealer:               sealer,
	})
	campaignArtifactHandlers := aiservice.NewCampaignArtifactHandlers(aiservice.CampaignArtifactHandlersConfig{
		Manager:                  campaignArtifactManager,
		AuthorizationClient:      gameAuthorizationClient,
		InternalServiceAllowlist: cfg.InternalServiceAllowlist,
	})
	providerGrantHandlers := aiservice.NewProviderGrantHandlers(aiservice.ProviderGrantHandlersConfig{
		ProviderGrantStore:    store,
		ConnectSessionStore:   store,
		AgentStore:            store,
		GameCampaignAIClient:  gameCampaignAIClient,
		Sealer:                sealer,
		ProviderOAuthAdapters: providerOAuthAdapters,
	})
	if cfg.SessionGrantConfig != nil {
		service.SetAISessionGrantConfig(*cfg.SessionGrantConfig)
	}
	if gameMc != nil {
		gameConn := gameMc.Conn()
		dialer := gametools.NewDirectDialer(gametools.Clients{
			Interaction: gamev1.NewInteractionServiceClient(gameConn),
			Scene:       gamev1.NewSceneServiceClient(gameConn),
			Campaign:    gamev1.NewCampaignServiceClient(gameConn),
			Participant: gamev1.NewParticipantServiceClient(gameConn),
			Character:   gamev1.NewCharacterServiceClient(gameConn),
			Session:     gamev1.NewSessionServiceClient(gameConn),
			Snapshot:    gamev1.NewSnapshotServiceClient(gameConn),
			Daggerheart: pb.NewDaggerheartServiceClient(gameConn),
			Artifact:    artifactClientAdapter{server: campaignArtifactHandlers},
			Reference:   referenceClientAdapter{server: systemReferenceHandlers},
		})
		promptBuilder := buildPromptBuilder(instructionLoader)
		runnerCfg := cfg.campaignTurnRunnerConfig(dialer)
		runnerCfg.PromptBuilder = promptBuilder
		service.SetCampaignTurnRunner(orchestration.NewRunner(runnerCfg))
	}

	healthServer := health.NewServer()
	aiv1.RegisterCredentialServiceServer(grpcServer, credentialHandlers)
	aiv1.RegisterAgentServiceServer(grpcServer, service)
	aiv1.RegisterInvocationServiceServer(grpcServer, service)
	aiv1.RegisterCampaignOrchestrationServiceServer(grpcServer, service)
	aiv1.RegisterCampaignArtifactServiceServer(grpcServer, campaignArtifactHandlers)
	aiv1.RegisterSystemReferenceServiceServer(grpcServer, systemReferenceHandlers)
	aiv1.RegisterProviderGrantServiceServer(grpcServer, providerGrantHandlers)
	aiv1.RegisterAccessRequestServiceServer(grpcServer, service)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("ai.v1.CredentialService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("ai.v1.AgentService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("ai.v1.InvocationService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("ai.v1.CampaignOrchestrationService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("ai.v1.CampaignArtifactService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("ai.v1.SystemReferenceService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("ai.v1.ProviderGrantService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("ai.v1.AccessRequestService", grpc_health_v1.HealthCheckResponse_SERVING)

	return &Server{
		listener:   listener,
		grpcServer: grpcServer,
		health:     healthServer,
		store:      store,
		gameMc:     gameMc,
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
	return RunWithAddr(ctx, fmt.Sprintf(":%d", port))
}

// RunWithAddr creates and serves an AI server until the context ends.
func RunWithAddr(ctx context.Context, addr string) error {
	server, err := NewWithAddrContext(ctx, addr)
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
		return errors.New("context is required")
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
		if s.health != nil {
			s.health.Shutdown()
		}
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
		closeManagedConn(s.gameMc, "game")
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

// buildPromptBuilder loads instruction files and creates a configured prompt
// builder. If instruction loading fails, it returns a builder with no
// pre-loaded instructions (the builder falls back to inline defaults).
func buildPromptBuilder(loader *campaigncontext.InstructionLoader) orchestration.PromptBuilder {
	if loader == nil {
		return nil
	}
	skills, err := loader.LoadSkills(campaigncontext.DaggerheartSystem)
	if err != nil {
		log.Printf("ai: load skills instructions: %v (using inline fallback)", err)
		return nil
	}
	interaction, err := loader.LoadCoreInteraction()
	if err != nil {
		log.Printf("ai: load interaction instructions: %v (using inline fallback)", err)
		return nil
	}

	reg := orchestration.NewContextSourceRegistry()
	for _, src := range orchestration.CoreContextSources() {
		reg.Register(src)
	}
	for _, src := range orchestration.DaggerheartContextSources() {
		reg.Register(src)
	}

	return orchestration.NewPromptBuilder(orchestration.PromptBuilderConfig{
		Skills:              skills,
		InteractionContract: interaction,
		ContextSources:      reg,
	})
}

func closeManagedConn(mc *platformgrpc.ManagedConn, name string) {
	if mc == nil {
		return
	}
	if err := mc.Close(); err != nil {
		log.Printf("close ai %s managed conn: %v", name, err)
	}
}
