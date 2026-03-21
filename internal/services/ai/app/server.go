package server

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
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
	"github.com/louisbranch/fracturing.space/internal/services/ai/campaigncontext/instructionset"
	"github.com/louisbranch/fracturing.space/internal/services/ai/campaigncontext/referencecorpus"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	orchdaggerheart "github.com/louisbranch/fracturing.space/internal/services/ai/orchestration/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration/gametools"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	openaiprovider "github.com/louisbranch/fracturing.space/internal/services/ai/provider/openai"
	"github.com/louisbranch/fracturing.space/internal/services/ai/secret"
	svcpkg "github.com/louisbranch/fracturing.space/internal/services/ai/service"
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
	logger     *slog.Logger
	closeOnce  sync.Once
}

// New creates a configured AI server using one startup context for dependency
// dialing and one parsed runtime config snapshot.
func New(ctx context.Context, addr string) (*Server, error) {
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
	providerOAuthAdapters := map[provider.Provider]provider.OAuthAdapter{
		provider.OpenAI: newDefaultOpenAIOAuthAdapter(),
	}
	if cfg.OpenAIOAuthConfig != nil {
		providerOAuthAdapters[provider.OpenAI] = openaiprovider.NewOAuthAdapter(*cfg.OpenAIOAuthConfig)
	}
	openAIAdapter := openaiprovider.NewInvokeAdapter(openaiprovider.InvokeConfig{
		ResponsesURL: cfg.OpenAIResponsesURL,
	})
	providerInvocationAdapters := map[provider.Provider]provider.InvocationAdapter{
		provider.OpenAI: openAIAdapter,
	}
	providerToolAdapters := map[provider.Provider]orchestration.Provider{
		provider.OpenAI: openAIAdapter,
	}
	providerModelAdapters := map[provider.Provider]provider.ModelAdapter{
		provider.OpenAI: openAIAdapter,
	}
	instructionLoader := instructionset.New(cfg.InstructionsRoot)
	campaignArtifactManager := campaigncontext.NewManager(campaigncontext.ManagerConfig{
		Store:         store,
		SkillsLoader:  instructionLoader,
		DefaultSystem: campaigncontext.DaggerheartSystem,
	})
	var referenceCorpus *referencecorpus.Corpus
	if strings.TrimSpace(cfg.DaggerheartReferenceRoot) != "" {
		referenceCorpus = referencecorpus.New(cfg.DaggerheartReferenceRoot)
	}
	systemReferenceHandlers := aiservice.NewSystemReferenceHandlers(referenceCorpus)

	logger := slog.Default().With("service", "ai")

	gameMc, gameClients := dialGameService(ctx, cfg.GameAddr, logger)
	handlers, err := buildHandlers(handlerDeps{
		store:                      store,
		sealer:                     sealer,
		cfg:                        cfg,
		providerOAuthAdapters:      providerOAuthAdapters,
		providerInvocationAdapters: providerInvocationAdapters,
		providerToolAdapters:       providerToolAdapters,
		providerModelAdapters:      providerModelAdapters,
		campaignArtifactManager:    campaignArtifactManager,
		referenceCorpus:            referenceCorpus,
		systemReferenceHandlers:    systemReferenceHandlers,
		instructionLoader:          instructionLoader,
		gameClients:                gameClients,
		gameMc:                     gameMc,
	})
	if err != nil {
		_ = listener.Close()
		_ = store.Close()
		closeManagedConn(gameMc, "game", logger)
		return nil, fmt.Errorf("build handlers: %w", err)
	}

	healthServer := health.NewServer()
	registerServices(grpcServer, healthServer, handlers)

	return &Server{
		listener:   listener,
		grpcServer: grpcServer,
		health:     healthServer,
		store:      store,
		gameMc:     gameMc,
		logger:     logger,
	}, nil
}

// gameServiceClients groups optional game service gRPC clients that are nil
// when the game service is unavailable.
type gameServiceClients struct {
	campaignAI    gamev1.CampaignAIServiceClient
	authorization gamev1.AuthorizationServiceClient
}

func dialGameService(ctx context.Context, gameAddr string, logger *slog.Logger) (*platformgrpc.ManagedConn, gameServiceClients) {
	gameAddr = strings.TrimSpace(gameAddr)
	if gameAddr == "" {
		return nil, gameServiceClients{}
	}
	mc, err := newManagedConn(ctx, platformgrpc.ManagedConnConfig{
		Name: "game",
		Addr: gameAddr,
		Mode: platformgrpc.ModeOptional,
		Logf: slogPrintf(logger),
		DialOpts: append(
			platformgrpc.LenientDialOptions(),
			grpc.WithChainUnaryInterceptor(grpcauthctx.ServiceIDUnaryClientInterceptor(serviceaddr.ServiceAI)),
			grpc.WithChainStreamInterceptor(grpcauthctx.ServiceIDStreamClientInterceptor(serviceaddr.ServiceAI)),
		),
	})
	if err != nil {
		logger.Warn("game managed conn unavailable; agent usage guard disabled", "error", err)
		return nil, gameServiceClients{}
	}
	return mc, gameServiceClients{
		campaignAI:    gamev1.NewCampaignAIServiceClient(mc.Conn()),
		authorization: gamev1.NewAuthorizationServiceClient(mc.Conn()),
	}
}

// handlerDeps groups runtime dependencies for handler construction.
type handlerDeps struct {
	store                      *aisqlite.Store
	sealer                     secret.Sealer
	cfg                        runtimeConfig
	providerOAuthAdapters      map[provider.Provider]provider.OAuthAdapter
	providerInvocationAdapters map[provider.Provider]provider.InvocationAdapter
	providerToolAdapters       map[provider.Provider]orchestration.Provider
	providerModelAdapters      map[provider.Provider]provider.ModelAdapter
	campaignArtifactManager    *campaigncontext.Manager
	referenceCorpus            *referencecorpus.Corpus
	systemReferenceHandlers    *aiservice.SystemReferenceHandlers
	instructionLoader          *instructionset.Loader
	gameClients                gameServiceClients
	gameMc                     *platformgrpc.ManagedConn
}

// serviceHandlers collects all constructed gRPC service implementations.
type serviceHandlers struct {
	credentials           *aiservice.CredentialHandlers
	agents                *aiservice.AgentHandlers
	invocations           *aiservice.InvocationHandlers
	campaignOrchestration *aiservice.CampaignOrchestrationHandlers
	campaignArtifacts     *aiservice.CampaignArtifactHandlers
	systemReferences      *aiservice.SystemReferenceHandlers
	providerGrants        *aiservice.ProviderGrantHandlers
	accessRequests        *aiservice.AccessRequestHandlers
}

func buildHandlers(d handlerDeps) (serviceHandlers, error) {
	usageGuard := svcpkg.NewUsageGuard(d.store, d.gameClients.campaignAI)
	credentialService, err := svcpkg.NewCredentialService(svcpkg.CredentialServiceConfig{
		CredentialStore: d.store,
		Sealer:          d.sealer,
		UsageGuard:      usageGuard,
	})
	if err != nil {
		return serviceHandlers{}, fmt.Errorf("credential service: %w", err)
	}
	credentialHandlers, err := aiservice.NewCredentialHandlers(aiservice.CredentialHandlersConfig{
		CredentialService: credentialService,
	})
	if err != nil {
		return serviceHandlers{}, fmt.Errorf("credential handlers: %w", err)
	}
	campaignArtifactHandlers, err := aiservice.NewCampaignArtifactHandlers(aiservice.CampaignArtifactHandlersConfig{
		Manager:                  d.campaignArtifactManager,
		AuthorizationClient:      d.gameClients.authorization,
		InternalServiceAllowlist: d.cfg.InternalServiceAllowlist,
	})
	if err != nil {
		return serviceHandlers{}, fmt.Errorf("campaign artifact handlers: %w", err)
	}
	providerGrantService, err := svcpkg.NewProviderGrantService(svcpkg.ProviderGrantServiceConfig{
		ProviderGrantStore:    d.store,
		ConnectSessionStore:   d.store,
		Sealer:                d.sealer,
		ProviderOAuthAdapters: d.providerOAuthAdapters,
		UsageGuard:            usageGuard,
	})
	if err != nil {
		return serviceHandlers{}, fmt.Errorf("provider grant service: %w", err)
	}
	providerGrantHandlers, err := aiservice.NewProviderGrantHandlers(aiservice.ProviderGrantHandlersConfig{
		ProviderGrantService: providerGrantService,
	})
	if err != nil {
		return serviceHandlers{}, fmt.Errorf("provider grant handlers: %w", err)
	}
	accessRequestService, err := svcpkg.NewAccessRequestService(svcpkg.AccessRequestServiceConfig{
		AgentStore:         d.store,
		AccessRequestStore: d.store,
		AuditEventStore:    d.store,
	})
	if err != nil {
		return serviceHandlers{}, fmt.Errorf("access request service: %w", err)
	}
	accessRequestHandlers, err := aiservice.NewAccessRequestHandlers(aiservice.AccessRequestHandlersConfig{
		AccessRequestService: accessRequestService,
	})
	if err != nil {
		return serviceHandlers{}, fmt.Errorf("access request handlers: %w", err)
	}
	authTokenResolver := svcpkg.NewAuthTokenResolver(svcpkg.AuthTokenResolverConfig{
		CredentialStore:       d.store,
		ProviderGrantStore:    d.store,
		ProviderOAuthAdapters: d.providerOAuthAdapters,
		Sealer:                d.sealer,
	})
	accessibleAgentResolver := svcpkg.NewAccessibleAgentResolver(d.store, d.store)

	agentService, err := svcpkg.NewAgentService(svcpkg.AgentServiceConfig{
		CredentialStore:         d.store,
		AgentStore:              d.store,
		ProviderGrantStore:      d.store,
		AccessRequestStore:      d.store,
		ProviderModelAdapters:   d.providerModelAdapters,
		AuthTokenResolver:       authTokenResolver,
		AccessibleAgentResolver: accessibleAgentResolver,
		UsageGuard:              usageGuard,
	})
	if err != nil {
		return serviceHandlers{}, fmt.Errorf("agent service: %w", err)
	}
	agentHandlers, err := aiservice.NewAgentHandlers(aiservice.AgentHandlersConfig{
		AgentService: agentService,
	})
	if err != nil {
		return serviceHandlers{}, fmt.Errorf("agent handlers: %w", err)
	}

	invocationService, err := svcpkg.NewInvocationService(svcpkg.InvocationServiceConfig{
		AgentStore:                 d.store,
		AuditEventStore:            d.store,
		AccessibleAgentResolver:    accessibleAgentResolver,
		AuthTokenResolver:          authTokenResolver,
		ProviderInvocationAdapters: d.providerInvocationAdapters,
	})
	if err != nil {
		return serviceHandlers{}, fmt.Errorf("invocation service: %w", err)
	}
	invocationHandlers, err := aiservice.NewInvocationHandlers(aiservice.InvocationHandlersConfig{
		InvocationService: invocationService,
	})
	if err != nil {
		return serviceHandlers{}, fmt.Errorf("invocation handlers: %w", err)
	}

	var campaignTurnRunner orchestration.CampaignTurnRunner
	if d.gameMc != nil {
		gameConn := d.gameMc.Conn()
		dialer := gametools.NewDirectDialer(gametools.Clients{
			Interaction: gamev1.NewInteractionServiceClient(gameConn),
			Scene:       gamev1.NewSceneServiceClient(gameConn),
			Campaign:    gamev1.NewCampaignServiceClient(gameConn),
			Participant: gamev1.NewParticipantServiceClient(gameConn),
			Character:   gamev1.NewCharacterServiceClient(gameConn),
			Session:     gamev1.NewSessionServiceClient(gameConn),
			Snapshot:    gamev1.NewSnapshotServiceClient(gameConn),
			Daggerheart: pb.NewDaggerheartServiceClient(gameConn),
			Artifact:    d.campaignArtifactManager,
			Reference:   d.referenceCorpus,
		})
		promptBuilder := buildPromptBuilder(d.instructionLoader)
		runnerCfg := d.cfg.campaignTurnRunnerConfig(dialer)
		runnerCfg.PromptBuilder = promptBuilder
		runnerCfg.ToolPolicy = orchestration.NewStaticToolPolicy(gametools.ProductionToolNames())
		campaignTurnRunner = orchestration.NewRunner(runnerCfg)
	}

	campaignOrchestrationService, err := svcpkg.NewCampaignOrchestrationService(svcpkg.CampaignOrchestrationServiceConfig{
		AgentStore:              d.store,
		CampaignArtifactManager: d.campaignArtifactManager,
		GameCampaignAIClient:    d.gameClients.campaignAI,
		ProviderToolAdapters:    d.providerToolAdapters,
		CampaignTurnRunner:      campaignTurnRunner,
		SessionGrantConfig:      d.cfg.SessionGrantConfig,
		AuthTokenResolver:       authTokenResolver,
	})
	if err != nil {
		return serviceHandlers{}, fmt.Errorf("campaign orchestration service: %w", err)
	}
	campaignOrchestrationHandlers, err := aiservice.NewCampaignOrchestrationHandlers(aiservice.CampaignOrchestrationHandlersConfig{
		CampaignOrchestrationService: campaignOrchestrationService,
	})
	if err != nil {
		return serviceHandlers{}, fmt.Errorf("campaign orchestration handlers: %w", err)
	}

	return serviceHandlers{
		credentials:           credentialHandlers,
		agents:                agentHandlers,
		invocations:           invocationHandlers,
		campaignOrchestration: campaignOrchestrationHandlers,
		campaignArtifacts:     campaignArtifactHandlers,
		systemReferences:      d.systemReferenceHandlers,
		providerGrants:        providerGrantHandlers,
		accessRequests:        accessRequestHandlers,
	}, nil
}

func registerServices(grpcServer *grpc.Server, healthServer *health.Server, h serviceHandlers) {
	aiv1.RegisterCredentialServiceServer(grpcServer, h.credentials)
	aiv1.RegisterAgentServiceServer(grpcServer, h.agents)
	aiv1.RegisterInvocationServiceServer(grpcServer, h.invocations)
	aiv1.RegisterCampaignOrchestrationServiceServer(grpcServer, h.campaignOrchestration)
	aiv1.RegisterCampaignArtifactServiceServer(grpcServer, h.campaignArtifacts)
	aiv1.RegisterSystemReferenceServiceServer(grpcServer, h.systemReferences)
	aiv1.RegisterProviderGrantServiceServer(grpcServer, h.providerGrants)
	aiv1.RegisterAccessRequestServiceServer(grpcServer, h.accessRequests)
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
}

// Addr returns the listener address for the AI server.
func (s *Server) Addr() string {
	if s == nil || s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// Run creates and serves an AI server until the context ends.
func Run(ctx context.Context, addr string) error {
	server, err := New(ctx, addr)
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

	s.logger.Info("server listening", "addr", s.listener.Addr())
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
				s.logger.Warn("close listener", "error", err)
			}
		}
		if s.store != nil {
			if err := s.store.Close(); err != nil {
				s.logger.Warn("close store", "error", err)
			}
		}
		closeManagedConn(s.gameMc, "game", s.logger)
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
// builder. Missing instruction content degrades explicitly to inline renderer
// defaults while preserving the full context-source registry.
func buildPromptBuilder(loader *instructionset.Loader) orchestration.PromptBuilder {
	return orchestration.NewPromptBuilder(orchestration.PromptBuilderConfig{
		Collector: buildPromptContextSources(),
		Renderer:  buildPromptRenderer(loader),
	})
}

func buildPromptContextSources() *orchestration.ContextSourceRegistry {
	reg := orchestration.NewContextSourceRegistry()
	for _, src := range orchestration.CoreContextSources() {
		reg.Register(src)
	}
	for _, src := range orchdaggerheart.ContextSources() {
		reg.Register(src)
	}
	return reg
}

func buildPromptRenderer(loader *instructionset.Loader) orchestration.PromptRenderer {
	policy := orchestration.DefaultPromptRenderPolicy()
	policy.Instructions = loadPromptInstructions(loader)
	return orchestration.NewBriefPromptRenderer(orchestration.BriefPromptRendererConfig{
		Policy: policy,
	})
}

func loadPromptInstructions(loader *instructionset.Loader) orchestration.PromptInstructions {
	if loader == nil {
		return orchestration.PromptInstructions{}
	}

	var instructions orchestration.PromptInstructions
	skills, err := loader.LoadSkills(campaigncontext.DaggerheartSystem)
	if err != nil {
		slog.Default().Warn("load skills instructions; using inline fallback", "error", err)
	} else {
		instructions.Skills = skills
	}

	interaction, err := loader.LoadCoreInteraction()
	if err != nil {
		slog.Default().Warn("load interaction instructions; using inline fallback", "error", err)
	} else {
		instructions.InteractionContract = interaction
	}

	return instructions
}

func closeManagedConn(mc *platformgrpc.ManagedConn, name string, logger *slog.Logger) {
	if mc == nil {
		return
	}
	if err := mc.Close(); err != nil {
		logger.Warn("close managed conn", "conn", name, "error", err)
	}
}

// slogPrintf adapts an slog.Logger to the func(string, ...any) callback
// signature used by platformgrpc.ManagedConnConfig.Logf.
func slogPrintf(logger *slog.Logger) func(string, ...any) {
	return func(format string, args ...any) {
		logger.Info(fmt.Sprintf(format, args...))
	}
}
