package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// InvocationService handles agent invocation operations.
type InvocationService struct {
	agentStore                 storage.AgentStore
	auditEventStore            storage.AuditEventStore
	accessibleAgentResolver    *AccessibleAgentResolver
	authTokenResolver          *AuthTokenResolver
	providerInvocationAdapters map[provider.Provider]provider.InvocationAdapter
	clock                      Clock
}

// InvocationServiceConfig declares dependencies for the invocation service.
type InvocationServiceConfig struct {
	AgentStore                 storage.AgentStore
	AuditEventStore            storage.AuditEventStore
	AccessibleAgentResolver    *AccessibleAgentResolver
	AuthTokenResolver          *AuthTokenResolver
	ProviderInvocationAdapters map[provider.Provider]provider.InvocationAdapter
	Clock                      Clock
}

// NewInvocationService builds an invocation service from explicit deps.
func NewInvocationService(cfg InvocationServiceConfig) (*InvocationService, error) {
	if cfg.AgentStore == nil {
		return nil, fmt.Errorf("ai: NewInvocationService: agent store is required")
	}
	if cfg.AuthTokenResolver == nil {
		return nil, fmt.Errorf("ai: NewInvocationService: auth token resolver is required")
	}
	if cfg.AccessibleAgentResolver == nil {
		return nil, fmt.Errorf("ai: NewInvocationService: accessible agent resolver is required")
	}

	providerInvocationAdapters := make(map[provider.Provider]provider.InvocationAdapter, len(cfg.ProviderInvocationAdapters))
	for k, v := range cfg.ProviderInvocationAdapters {
		providerInvocationAdapters[k] = v
	}

	return &InvocationService{
		agentStore:                 cfg.AgentStore,
		auditEventStore:            cfg.AuditEventStore,
		accessibleAgentResolver:    cfg.AccessibleAgentResolver,
		authTokenResolver:          cfg.AuthTokenResolver,
		providerInvocationAdapters: providerInvocationAdapters,
		clock:                      withDefaultClock(cfg.Clock),
	}, nil
}

// InvokeAgentInput is the domain input for invoking an agent.
type InvokeAgentInput struct {
	CallerUserID    string
	AgentID         string
	Input           string
	ReasoningEffort string
}

// InvokeAgentResult is the domain result of an agent invocation.
type InvokeAgentResult struct {
	OutputText string
	Provider   provider.Provider
	Model      string
	Usage      provider.Usage
}

// InvokeAgent executes one provider call using an owned or shared-access agent.
func (s *InvocationService) InvokeAgent(ctx context.Context, input InvokeAgentInput) (InvokeAgentResult, error) {
	if input.AgentID == "" {
		return InvokeAgentResult{}, Errorf(ErrKindInvalidArgument, "agent_id is required")
	}
	if input.Input == "" {
		return InvokeAgentResult{}, Errorf(ErrKindInvalidArgument, "input is required")
	}

	agentRecord, err := s.agentStore.GetAgent(ctx, input.AgentID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return InvokeAgentResult{}, Errorf(ErrKindNotFound, "agent not found")
		}
		return InvokeAgentResult{}, Wrapf(ErrKindInternal, err, "get agent")
	}

	authResult, err := s.accessibleAgentResolver.IsAuthorizedToInvokeAgent(ctx, input.CallerUserID, agentRecord)
	if err != nil {
		return InvokeAgentResult{}, err
	}
	if !authResult.Authorized {
		return InvokeAgentResult{}, Errorf(ErrKindNotFound, "agent not found")
	}

	adapter, ok := s.providerInvocationAdapters[agentRecord.Provider]
	if !ok || adapter == nil {
		return InvokeAgentResult{}, Errorf(ErrKindFailedPrecondition, "provider invocation adapter is unavailable")
	}

	invokeToken, err := s.authTokenResolver.ResolveAgentInvokeToken(ctx, agentRecord.OwnerUserID, agentRecord)
	if err != nil {
		return InvokeAgentResult{}, err
	}
	result, err := adapter.Invoke(ctx, provider.InvokeInput{
		Model:            agentRecord.Model,
		Input:            input.Input,
		Instructions:     agentRecord.Instructions,
		ReasoningEffort:  input.ReasoningEffort,
		CredentialSecret: invokeToken,
	})
	if err != nil {
		return InvokeAgentResult{}, Wrapf(ErrKindInternal, err, "invoke provider")
	}
	if result.OutputText == "" {
		return InvokeAgentResult{}, Errorf(ErrKindInternal, "provider returned empty output")
	}
	if authResult.SharedAccess {
		if err := s.putAuditEvent(ctx, storage.AuditEventRecord{
			EventName:       "agent.invoke.shared",
			ActorUserID:     input.CallerUserID,
			OwnerUserID:     agentRecord.OwnerUserID,
			RequesterUserID: input.CallerUserID,
			AgentID:         agentRecord.ID,
			AccessRequestID: authResult.AccessRequestID,
			Outcome:         "success",
			CreatedAt:       s.clock().UTC(),
		}); err != nil {
			return InvokeAgentResult{}, Wrapf(ErrKindInternal, err, "put audit event")
		}
	}
	return InvokeAgentResult{
		OutputText: result.OutputText,
		Provider:   agentRecord.Provider,
		Model:      agentRecord.Model,
		Usage:      result.Usage,
	}, nil
}

// putAuditEvent persists one audit event record.
func (s *InvocationService) putAuditEvent(ctx context.Context, record storage.AuditEventRecord) error {
	if s.auditEventStore == nil {
		return fmt.Errorf("audit event store is not configured")
	}
	record.CreatedAt = record.CreatedAt.UTC()
	return s.auditEventStore.PutAuditEvent(ctx, record)
}
