package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/campaigncontext"
	"github.com/louisbranch/fracturing.space/internal/services/ai/debugtrace"
	"github.com/louisbranch/fracturing.space/internal/services/ai/gamebridge"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providercatalog"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"github.com/louisbranch/fracturing.space/internal/services/shared/aisessiongrant"
)

// CampaignOrchestrationService handles campaign-turn orchestration operations.
type CampaignOrchestrationService struct {
	agentStore              storage.AgentStore
	campaignArtifactManager *campaigncontext.Manager
	campaignAuthStateReader CampaignAuthStateReader
	providerRegistry        *providercatalog.Registry
	campaignTurnRunner      orchestration.CampaignTurnRunner
	debugTraceStore         debugtrace.Store
	debugUpdateBroker       *CampaignDebugUpdateBroker
	sessionGrantConfig      *aisessiongrant.Config
	authMaterialResolver    *AuthMaterialResolver
	clock                   Clock
	idGenerator             IDGenerator
	logger                  *slog.Logger
}

// CampaignOrchestrationServiceConfig declares dependencies for the campaign
// orchestration service.
type CampaignOrchestrationServiceConfig struct {
	AgentStore              storage.AgentStore
	CampaignArtifactManager *campaigncontext.Manager
	CampaignAuthStateReader CampaignAuthStateReader
	ProviderRegistry        *providercatalog.Registry
	CampaignTurnRunner      orchestration.CampaignTurnRunner
	DebugTraceStore         debugtrace.Store
	DebugUpdateBroker       *CampaignDebugUpdateBroker
	SessionGrantConfig      *aisessiongrant.Config
	AuthMaterialResolver    *AuthMaterialResolver
	Clock                   Clock
	IDGenerator             IDGenerator
	Logger                  *slog.Logger
}

// NewCampaignOrchestrationService builds a campaign orchestration service from
// explicit deps.
func NewCampaignOrchestrationService(cfg CampaignOrchestrationServiceConfig) (*CampaignOrchestrationService, error) {
	if cfg.AgentStore == nil {
		return nil, fmt.Errorf("ai: NewCampaignOrchestrationService: agent store is required")
	}
	if err := RequireAuthMaterialResolver(cfg.AuthMaterialResolver, "NewCampaignOrchestrationService"); err != nil {
		return nil, err
	}
	if err := RequireProviderRegistry(cfg.ProviderRegistry, "NewCampaignOrchestrationService"); err != nil {
		return nil, err
	}

	var sessionGrantConfig *aisessiongrant.Config
	if cfg.SessionGrantConfig != nil {
		copied := *cfg.SessionGrantConfig
		sessionGrantConfig = &copied
	}

	return &CampaignOrchestrationService{
		agentStore:              cfg.AgentStore,
		campaignArtifactManager: cfg.CampaignArtifactManager,
		campaignAuthStateReader: cfg.CampaignAuthStateReader,
		providerRegistry:        cfg.ProviderRegistry,
		campaignTurnRunner:      cfg.CampaignTurnRunner,
		debugTraceStore:         cfg.DebugTraceStore,
		debugUpdateBroker:       cfg.DebugUpdateBroker,
		sessionGrantConfig:      sessionGrantConfig,
		authMaterialResolver:    cfg.AuthMaterialResolver,
		clock:                   withDefaultClock(cfg.Clock),
		idGenerator:             withDefaultIDGenerator(cfg.IDGenerator),
		logger:                  cfg.Logger,
	}, nil
}

// RunCampaignTurnInput is the domain input for running a campaign turn.
type RunCampaignTurnInput struct {
	SessionGrant    string
	Input           string
	ReasoningEffort string
	TurnToken       string
}

// RunCampaignTurnResult is the domain result of a campaign turn.
type RunCampaignTurnResult struct {
	OutputText string
	Provider   provider.Provider
	Model      string
	Usage      provider.Usage
}

// RunCampaignTurn validates a game-issued session grant and executes one GM turn.
// Orchestration errors are returned raw for the transport layer to map.
func (s *CampaignOrchestrationService) RunCampaignTurn(ctx context.Context, input RunCampaignTurnInput) (RunCampaignTurnResult, error) {
	if s.campaignTurnRunner == nil {
		return RunCampaignTurnResult{}, Errorf(ErrKindFailedPrecondition, "campaign turn runner is unavailable")
	}
	if s.sessionGrantConfig == nil {
		return RunCampaignTurnResult{}, Errorf(ErrKindFailedPrecondition, "ai session grant validation is unavailable")
	}
	if s.campaignAuthStateReader == nil {
		return RunCampaignTurnResult{}, Errorf(ErrKindFailedPrecondition, "campaign ai auth state client is unavailable")
	}
	if input.SessionGrant == "" {
		return RunCampaignTurnResult{}, Errorf(ErrKindInvalidArgument, "session_grant is required")
	}

	claims, err := aisessiongrant.Validate(*s.sessionGrantConfig, input.SessionGrant)
	if err != nil {
		switch {
		case errors.Is(err, aisessiongrant.ErrExpired):
			return RunCampaignTurnResult{}, Errorf(ErrKindPermissionDenied, "session grant is expired")
		case errors.Is(err, aisessiongrant.ErrInvalid):
			return RunCampaignTurnResult{}, Errorf(ErrKindPermissionDenied, "session grant is invalid")
		default:
			return RunCampaignTurnResult{}, Wrapf(ErrKindInternal, err, "validate session grant")
		}
	}

	state, err := s.campaignAuthStateReader.CampaignAuthState(ctx, claims.CampaignID)
	if err != nil {
		if errors.Is(err, gamebridge.ErrCampaignAuthStateUnavailable) {
			return RunCampaignTurnResult{}, Errorf(ErrKindFailedPrecondition, "campaign ai auth state client is unavailable")
		}
		return RunCampaignTurnResult{}, Wrapf(ErrKindInternal, err, "get campaign ai auth state")
	}
	if staleGrant(claims, state) {
		return RunCampaignTurnResult{}, Errorf(ErrKindFailedPrecondition, "campaign ai session grant is stale")
	}

	agentID := strings.TrimSpace(state.GetAiAgentId())
	if agentID == "" {
		return RunCampaignTurnResult{}, Errorf(ErrKindFailedPrecondition, "campaign ai runtime is unavailable")
	}

	agentRecord, err := s.agentStore.GetAgent(ctx, agentID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return RunCampaignTurnResult{}, Errorf(ErrKindFailedPrecondition, "campaign ai runtime is unavailable")
		}
		return RunCampaignTurnResult{}, Wrapf(ErrKindInternal, err, "get campaign ai runtime")
	}
	if !agentRecord.Status.IsActive() {
		return RunCampaignTurnResult{}, Errorf(ErrKindFailedPrecondition, "campaign ai runtime is inactive")
	}
	if s.campaignArtifactManager != nil {
		if _, err := s.campaignArtifactManager.EnsureDefaultArtifacts(ctx, claims.CampaignID, ""); err != nil {
			return RunCampaignTurnResult{}, Wrapf(ErrKindInternal, err, "ensure campaign artifacts")
		}
	}

	adapter, ok := s.providerRegistry.ToolAdapter(agentRecord.Provider)
	if !ok || adapter == nil {
		return RunCampaignTurnResult{}, Errorf(ErrKindFailedPrecondition, "campaign ai provider adapter is unavailable")
	}

	token, err := s.authMaterialResolver.ResolveAgentInvokeToken(ctx, agentRecord.OwnerUserID, agentRecord)
	if err != nil {
		return RunCampaignTurnResult{}, err
	}

	now := s.clock().UTC()
	traceRecorder := newCampaignDebugTraceRecorder(ctx, s.debugTraceStore, s.clock, s.debugUpdateBroker, s.idGenerator, s.logger, debugtrace.Turn{
		CampaignID:    claims.CampaignID,
		SessionID:     claims.SessionID,
		TurnToken:     strings.TrimSpace(input.TurnToken),
		ParticipantID: strings.TrimSpace(state.GetParticipantId()),
		Provider:      agentRecord.Provider,
		Model:         agentRecord.Model,
		StartedAt:     now,
		UpdatedAt:     now,
	})

	result, err := s.campaignTurnRunner.Run(ctx, orchestration.Input{
		CampaignID:      claims.CampaignID,
		SessionID:       claims.SessionID,
		ParticipantID:   strings.TrimSpace(state.GetParticipantId()),
		Input:           input.Input,
		Model:           agentRecord.Model,
		ReasoningEffort: input.ReasoningEffort,
		Instructions:    agentRecord.Instructions,
		AuthToken:       token,
		Provider:        adapter,
		TraceRecorder:   traceRecorder,
	})
	if traceRecorder != nil {
		traceRecorder.Finish(ctx, err)
	}
	if err != nil {
		// Return the raw orchestration error for the transport layer to map.
		return RunCampaignTurnResult{}, err
	}
	if result.OutputText == "" {
		return RunCampaignTurnResult{}, orchestration.ErrEmptyOutput
	}
	return RunCampaignTurnResult{
		OutputText: result.OutputText,
		Provider:   agentRecord.Provider,
		Model:      agentRecord.Model,
		Usage:      result.Usage,
	}, nil
}

// staleGrant checks whether the session grant claims are stale relative to
// the current campaign auth state.
func staleGrant(claims aisessiongrant.Claims, state *gamev1.GetCampaignAIAuthStateResponse) bool {
	if state == nil {
		return true
	}
	if strings.TrimSpace(state.GetCampaignId()) != strings.TrimSpace(claims.CampaignID) {
		return true
	}
	if strings.TrimSpace(state.GetActiveSessionId()) != strings.TrimSpace(claims.SessionID) {
		return true
	}
	if strings.TrimSpace(state.GetParticipantId()) != strings.TrimSpace(claims.ParticipantID) {
		return true
	}
	return state.GetAuthEpoch() != claims.AuthEpoch
}
