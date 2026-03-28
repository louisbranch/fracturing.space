package server

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	aiservice "github.com/louisbranch/fracturing.space/internal/services/ai/api/grpc/ai"
	"github.com/louisbranch/fracturing.space/internal/services/ai/campaigncontext"
	"github.com/louisbranch/fracturing.space/internal/services/ai/campaigncontext/instructionset"
	"github.com/louisbranch/fracturing.space/internal/services/ai/campaigncontext/referencecorpus"
	"github.com/louisbranch/fracturing.space/internal/services/ai/gamebridge"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration/gametools"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	anthropicprovider "github.com/louisbranch/fracturing.space/internal/services/ai/provider/anthropic"
	openaiprovider "github.com/louisbranch/fracturing.space/internal/services/ai/provider/openai"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providercatalog"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provideroauth"
	"github.com/louisbranch/fracturing.space/internal/services/ai/secret"
	aisqlite "github.com/louisbranch/fracturing.space/internal/services/ai/storage/sqlite"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	"google.golang.org/grpc"
)

// runtimeDeps groups the long-lived runtime collaborators built by the AI
// composition root before workflow assembly.
type runtimeDeps struct {
	cfg                     runtimeConfig
	store                   *aisqlite.Store
	sealer                  secret.Sealer
	providerRegistry        *providercatalog.Registry
	campaignArtifactManager *campaigncontext.Manager
	referenceCorpus         *referencecorpus.Corpus
	systemReferenceHandlers *aiservice.SystemReferenceHandlers
	instructionLoader       *instructionset.Loader
	gameBridge              *gamebridge.Gateway
	gameMc                  *platformgrpc.ManagedConn
}

func buildRuntimeDeps(ctx context.Context, cfg runtimeConfig, logger *slog.Logger) (runtimeDeps, error) {
	store, err := openAIStore(cfg.DBPath)
	if err != nil {
		return runtimeDeps{}, err
	}

	sealer, err := cfg.buildSealer()
	if err != nil {
		_ = store.Close()
		return runtimeDeps{}, err
	}

	openAIOAuthAdapter := provideroauth.Adapter(newDefaultOpenAIOAuthAdapter())
	if cfg.OpenAIOAuthConfig != nil {
		openAIOAuthAdapter = openaiprovider.NewOAuthAdapter(*cfg.OpenAIOAuthConfig)
	}

	openAIAdapter := openaiprovider.NewInvokeAdapter(openaiprovider.InvokeConfig{
		ResponsesURL: cfg.OpenAIResponsesURL,
	})
	anthropicAdapter := anthropicprovider.NewAdapter(anthropicprovider.Config{
		BaseURL: cfg.AnthropicBaseURL,
	})
	providerRegistry, err := providercatalog.New(
		providercatalog.Bundle{
			Provider:   provider.OpenAI,
			OAuth:      openAIOAuthAdapter,
			Invocation: openAIAdapter,
			Model:      openAIAdapter,
			Tool:       openAIAdapter,
		},
		providercatalog.Bundle{
			Provider:   provider.Anthropic,
			Invocation: anthropicAdapter,
			Model:      anthropicAdapter,
		},
	)
	if err != nil {
		_ = store.Close()
		return runtimeDeps{}, fmt.Errorf("build provider registry: %w", err)
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

	gameMc, gameBridge := dialGameService(ctx, cfg.GameAddr, cfg.InternalServiceAllowlist, logger)

	return runtimeDeps{
		cfg:                     cfg,
		store:                   store,
		sealer:                  sealer,
		providerRegistry:        providerRegistry,
		campaignArtifactManager: campaignArtifactManager,
		referenceCorpus:         referenceCorpus,
		systemReferenceHandlers: aiservice.NewSystemReferenceHandlers(referenceCorpus),
		instructionLoader:       instructionLoader,
		gameBridge:              gameBridge,
		gameMc:                  gameMc,
	}, nil
}

func (d runtimeDeps) close(logger *slog.Logger) {
	if d.store != nil {
		if err := d.store.Close(); err != nil && logger != nil {
			logger.Warn("close store", "error", err)
		}
	}
	closeManagedConn(d.gameMc, "game", logger)
}

func dialGameService(ctx context.Context, gameAddr string, internalServiceAllowlist map[string]struct{}, logger *slog.Logger) (*platformgrpc.ManagedConn, *gamebridge.Gateway) {
	gameAddr = strings.TrimSpace(gameAddr)
	if gameAddr == "" {
		return nil, gamebridge.New(gamebridge.Config{
			InternalServiceAllowlist: internalServiceAllowlist,
		})
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
		return nil, gamebridge.New(gamebridge.Config{
			InternalServiceAllowlist: internalServiceAllowlist,
		})
	}
	return mc, gamebridge.New(gamebridge.Config{
		CampaignAI:               gamev1.NewCampaignAIServiceClient(mc.Conn()),
		Authorization:            gamev1.NewAuthorizationServiceClient(mc.Conn()),
		InternalServiceAllowlist: internalServiceAllowlist,
	})
}

func buildCampaignTurnRunner(d runtimeDeps) orchestration.CampaignTurnRunner {
	if d.gameMc == nil {
		return nil
	}

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
	return orchestration.NewRunner(runnerCfg)
}
