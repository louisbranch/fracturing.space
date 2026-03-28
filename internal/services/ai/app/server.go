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
	"sync"

	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/services/ai/campaigncontext"
	"github.com/louisbranch/fracturing.space/internal/services/ai/campaigncontext/instructionset"
	"github.com/louisbranch/fracturing.space/internal/services/ai/openviking"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	orchdaggerheart "github.com/louisbranch/fracturing.space/internal/services/ai/orchestration/daggerheart"
	aisqlite "github.com/louisbranch/fracturing.space/internal/services/ai/storage/sqlite"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
)

type serverDependencies struct {
	listenTCP      func(string, string) (net.Listener, error)
	newManagedConn func(context.Context, platformgrpc.ManagedConnConfig) (*platformgrpc.ManagedConn, error)
	openStore      func(string) (*aisqlite.Store, error)
}

func defaultServerDependencies() serverDependencies {
	return serverDependencies{
		listenTCP:      net.Listen,
		newManagedConn: platformgrpc.NewManagedConn,
		openStore:      openAIStore,
	}
}

func (d serverDependencies) withDefaults() serverDependencies {
	if d.listenTCP == nil {
		d.listenTCP = net.Listen
	}
	if d.newManagedConn == nil {
		d.newManagedConn = platformgrpc.NewManagedConn
	}
	if d.openStore == nil {
		d.openStore = openAIStore
	}
	return d
}

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
	return newServerWithRuntimeConfig(ctx, addr, cfg, defaultServerDependencies())
}

func newServerWithRuntimeConfig(ctx context.Context, addr string, cfg runtimeConfig, deps serverDependencies) (*Server, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	deps = deps.withDefaults()

	listener, err := deps.listenTCP("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", addr, err)
	}

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(serviceIdentityValidationUnaryInterceptor(cfg.InternalServiceAllowlist)),
		grpc.ChainStreamInterceptor(serviceIdentityValidationStreamInterceptor(cfg.InternalServiceAllowlist)),
	)

	logger := slog.Default().With("service", "ai")

	runtimeDeps, err := buildRuntimeDeps(ctx, cfg, logger, deps)
	if err != nil {
		_ = listener.Close()
		return nil, err
	}

	handlers, err := buildHandlers(runtimeDeps)
	if err != nil {
		_ = listener.Close()
		runtimeDeps.close(logger)
		return nil, fmt.Errorf("build handlers: %w", err)
	}

	healthServer := health.NewServer()
	registerServices(grpcServer, healthServer, handlers)

	return &Server{
		listener:   listener,
		grpcServer: grpcServer,
		health:     healthServer,
		store:      runtimeDeps.store,
		gameMc:     runtimeDeps.gameMc,
		logger:     logger,
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
func buildPromptBuilder(loader *instructionset.Loader, augmenter orchestration.PromptAugmenter, mode openviking.IntegrationMode) orchestration.PromptBuilder {
	openVikingEnabled := augmenter != nil
	policy := promptContextPolicyFor(mode, openVikingEnabled)
	if augmenter != nil {
		augmenter = bestEffortPromptAugmenter{inner: augmenter}
	}
	return orchestration.NewPromptBuilder(orchestration.PromptBuilderConfig{
		Collector: promptContextPolicyRecorder{
			inner: buildPromptContextSources(policy),
			policy: orchestration.PromptContextPolicy{
				IncludeStory:  policy.IncludeStory,
				IncludeMemory: policy.IncludeMemory,
			},
		},
		Augmenter: augmenter,
		Renderer:  buildPromptRenderer(loader),
	})
}

type promptContextPolicy struct {
	IncludeStory  bool
	IncludeMemory bool
}

type promptContextPolicyRecorder struct {
	inner  orchestration.SessionBriefCollector
	policy orchestration.PromptContextPolicy
}

func (r promptContextPolicyRecorder) CollectBrief(ctx context.Context, sess orchestration.Session, input orchestration.PromptInput) (orchestration.SessionBrief, error) {
	orchestration.RecordPromptContextPolicy(ctx, r.policy)
	return r.inner.CollectBrief(ctx, sess, input)
}

func promptContextPolicyFor(mode openviking.IntegrationMode, openVikingEnabled bool) promptContextPolicy {
	includeStory := true
	includeMemory := true
	if openVikingEnabled {
		includeStory = !mode.SuppressStoryPrompt()
		includeMemory = !mode.SuppressMemoryPrompt()
	}
	return promptContextPolicy{
		IncludeStory:  includeStory,
		IncludeMemory: includeMemory,
	}
}

func buildPromptContextSources(policy promptContextPolicy) *orchestration.ContextSourceRegistry {
	reg := orchestration.NewCoreContextSourceRegistryWithConfig(orchestration.CoreContextSourceConfig{
		IncludeStory:      policy.IncludeStory,
		IncludeStoryIndex: !policy.IncludeStory,
		IncludeMemory:     policy.IncludeMemory,
	})
	reg.RegisterAll(orchdaggerheart.ContextSources()...)
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

type bestEffortPromptAugmenter struct {
	inner orchestration.PromptAugmenter
}

func (a bestEffortPromptAugmenter) Augment(ctx context.Context, sess orchestration.Session, brief orchestration.SessionBrief, input orchestration.PromptInput) (orchestration.BriefContribution, error) {
	if a.inner == nil {
		return orchestration.BriefContribution{}, nil
	}
	contribution, err := a.inner.Augment(ctx, sess, brief, input)
	if err != nil {
		orchestration.RecordPromptAugmentation(ctx, orchestration.PromptAugmentationDiagnostics{
			Degraded:          true,
			DegradationReason: err.Error(),
		})
		slog.Default().Warn("openviking prompt augmentation unavailable; continuing without supplemental context", "error", err)
		return orchestration.BriefContribution{}, nil
	}
	return contribution, nil
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
