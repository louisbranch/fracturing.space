package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// AgentAuthState represents the runtime auth-health of one agent.
type AgentAuthState int

const (
	// AgentAuthStateUnknown means the auth state could not be determined.
	AgentAuthStateUnknown AgentAuthState = iota
	// AgentAuthStateReady means the agent's auth reference is usable.
	AgentAuthStateReady
	// AgentAuthStateRevoked means the underlying auth reference was revoked.
	AgentAuthStateRevoked
	// AgentAuthStateUnavailable means the auth reference is missing or unusable.
	AgentAuthStateUnavailable
)

// AccessibleAgentsPage is a paginated result of accessible agents.
type AccessibleAgentsPage struct {
	Agents        []agent.Agent
	NextPageToken string
}

// AgentService handles agent lifecycle operations.
type AgentService struct {
	agentStore              storage.AgentStore
	authReferencePolicy     *AuthReferencePolicy
	accessibleAgentResolver *AccessibleAgentResolver
	usagePolicy             *UsagePolicy
	agentBindingUsageReader *AgentBindingUsageReader
	clock                   Clock
	idGenerator             IDGenerator
}

// AgentServiceConfig declares dependencies for the agent service.
type AgentServiceConfig struct {
	AgentStore              storage.AgentStore
	AuthReferencePolicy     *AuthReferencePolicy
	AccessibleAgentResolver *AccessibleAgentResolver
	UsagePolicy             *UsagePolicy
	AgentBindingUsageReader *AgentBindingUsageReader
	Clock                   Clock
	IDGenerator             IDGenerator
}

// NewAgentService builds an agent service from explicit deps.
func NewAgentService(cfg AgentServiceConfig) (*AgentService, error) {
	if cfg.AgentStore == nil {
		return nil, fmt.Errorf("ai: NewAgentService: agent store is required")
	}
	if cfg.AuthReferencePolicy == nil {
		return nil, fmt.Errorf("ai: NewAgentService: auth reference policy is required")
	}
	if cfg.AccessibleAgentResolver == nil {
		return nil, fmt.Errorf("ai: NewAgentService: accessible agent resolver is required")
	}

	return &AgentService{
		agentStore:              cfg.AgentStore,
		authReferencePolicy:     cfg.AuthReferencePolicy,
		accessibleAgentResolver: cfg.AccessibleAgentResolver,
		usagePolicy:             cfg.UsagePolicy,
		agentBindingUsageReader: cfg.AgentBindingUsageReader,
		clock:                   withDefaultClock(cfg.Clock),
		idGenerator:             withDefaultIDGenerator(cfg.IDGenerator),
	}, nil
}

// CreateAgentInput is the domain input for creating an agent.
type CreateAgentInput struct {
	OwnerUserID   string
	Label         string
	Instructions  string
	Provider      provider.Provider
	Model         string
	AuthReference agent.AuthReference
}

// Create creates a user-owned AI agent profile.
func (s *AgentService) Create(ctx context.Context, input CreateAgentInput) (agent.Agent, error) {
	authReference, err := agent.NormalizeAuthReference(input.AuthReference, true)
	if err != nil {
		return agent.Agent{}, Errorf(ErrKindInvalidArgument, "%s", err)
	}
	createInput, err := agent.NormalizeCreateInput(agent.CreateInput{
		OwnerUserID:   input.OwnerUserID,
		Label:         input.Label,
		Instructions:  input.Instructions,
		Provider:      input.Provider,
		Model:         input.Model,
		AuthReference: authReference,
	})
	if err != nil {
		return agent.Agent{}, Errorf(ErrKindInvalidArgument, "%s", err)
	}
	if err := s.authReferencePolicy.ValidateUsable(ctx, input.OwnerUserID, input.Provider, createInput.AuthReference); err != nil {
		return agent.Agent{}, err
	}
	if err := s.authReferencePolicy.ValidateModelAvailable(ctx, input.OwnerUserID, input.Provider, createInput.AuthReference, createInput.Model); err != nil {
		return agent.Agent{}, err
	}

	created, err := agent.Create(createInput, s.clock, s.idGenerator)
	if err != nil {
		return agent.Agent{}, Errorf(ErrKindInvalidArgument, "%s", err)
	}

	if err := s.agentStore.PutAgent(ctx, created); err != nil {
		if errors.Is(err, storage.ErrConflict) {
			return agent.Agent{}, Errorf(ErrKindAlreadyExists, "agent label already exists")
		}
		return agent.Agent{}, Wrapf(ErrKindInternal, err, "put agent")
	}

	return created, nil
}

// List returns a page of agents owned by the given user.
func (s *AgentService) List(ctx context.Context, ownerUserID string, pageSize int, pageToken string) (agent.Page, error) {
	page, err := s.agentStore.ListAgentsByOwner(ctx, ownerUserID, pageSize, pageToken)
	if err != nil {
		return agent.Page{}, Wrapf(ErrKindInternal, err, "list agents")
	}
	return page, nil
}

// ListProviderModelsInput is the domain input for listing provider models.
type ListProviderModelsInput struct {
	OwnerUserID   string
	Provider      provider.Provider
	AuthReference agent.AuthReference
}

// ListProviderModels returns provider-backed model options for one owned auth
// reference.
func (s *AgentService) ListProviderModels(ctx context.Context, input ListProviderModelsInput) ([]provider.Model, error) {
	authReference, err := agent.NormalizeAuthReference(input.AuthReference, true)
	if err != nil {
		return nil, Errorf(ErrKindInvalidArgument, "%s", err)
	}
	return s.authReferencePolicy.ListProviderModels(
		ctx,
		input.OwnerUserID,
		input.Provider,
		authReference,
	)
}

// ListAccessible returns a page of agents the caller can invoke, combining
// owned agents with approved shared invoke access via a single paginated query.
func (s *AgentService) ListAccessible(ctx context.Context, userID string, pageSize int, pageToken string) (AccessibleAgentsPage, error) {
	page, err := s.accessibleAgentResolver.ListAccessibleAgents(ctx, userID, pageSize, pageToken)
	if err != nil {
		return AccessibleAgentsPage{}, err
	}
	return AccessibleAgentsPage{
		Agents:        page.Agents,
		NextPageToken: page.NextPageToken,
	}, nil
}

// GetAccessible returns one agent by ID when the caller can invoke it.
func (s *AgentService) GetAccessible(ctx context.Context, userID string, agentID string) (agent.Agent, error) {
	if agentID == "" {
		return agent.Agent{}, Errorf(ErrKindInvalidArgument, "agent_id is required")
	}

	a, err := s.agentStore.GetAgent(ctx, agentID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return agent.Agent{}, Errorf(ErrKindNotFound, "agent not found")
		}
		return agent.Agent{}, Wrapf(ErrKindInternal, err, "get agent")
	}

	authResult, err := s.accessibleAgentResolver.IsAuthorizedToInvokeAgent(ctx, userID, a)
	if err != nil {
		return agent.Agent{}, err
	}
	if !authResult.Authorized {
		return agent.Agent{}, Errorf(ErrKindNotFound, "agent not found")
	}
	return a, nil
}

// ValidateCampaignAgentBinding verifies owner-scoped bind eligibility for one
// agent.
func (s *AgentService) ValidateCampaignAgentBinding(ctx context.Context, userID string, agentID string) (agent.Agent, error) {
	if agentID == "" {
		return agent.Agent{}, Errorf(ErrKindInvalidArgument, "agent_id is required")
	}

	a, err := s.agentStore.GetAgent(ctx, agentID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return agent.Agent{}, Errorf(ErrKindNotFound, "agent not found")
		}
		return agent.Agent{}, Wrapf(ErrKindInternal, err, "get agent")
	}
	if a.OwnerUserID != userID {
		return agent.Agent{}, Errorf(ErrKindNotFound, "agent not found")
	}
	if !a.Status.IsActive() {
		return agent.Agent{}, Errorf(ErrKindFailedPrecondition, "agent is not active")
	}
	if err := s.authReferencePolicy.ValidateUsable(ctx, userID, a.Provider, a.AuthReference); err != nil {
		return agent.Agent{}, err
	}
	return a, nil
}

// UpdateAgentInput is the domain input for updating an agent.
type UpdateAgentInput struct {
	OwnerUserID   string
	AgentID       string
	Label         string
	Instructions  string
	Model         string
	AuthReference agent.AuthReference
}

// Update updates mutable fields on one user-owned agent.
func (s *AgentService) Update(ctx context.Context, input UpdateAgentInput) (agent.Agent, error) {
	if input.AgentID == "" {
		return agent.Agent{}, Errorf(ErrKindInvalidArgument, "agent_id is required")
	}

	existing, err := s.agentStore.GetAgent(ctx, input.AgentID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return agent.Agent{}, Errorf(ErrKindNotFound, "agent not found")
		}
		return agent.Agent{}, Wrapf(ErrKindInternal, err, "get agent")
	}
	if existing.OwnerUserID != input.OwnerUserID {
		return agent.Agent{}, Errorf(ErrKindNotFound, "agent not found")
	}
	if s.usagePolicy != nil {
		if err := s.usagePolicy.EnsureAgentNotBoundToActiveCampaigns(ctx, existing.ID); err != nil {
			return agent.Agent{}, err
		}
	}

	label := firstNonEmpty(input.Label, existing.Label)
	instructions := firstNonEmpty(input.Instructions, existing.Instructions)
	model := firstNonEmpty(input.Model, existing.Model)
	authReference := existing.AuthReference
	if !input.AuthReference.IsZero() {
		authReference, err = agent.NormalizeAuthReference(input.AuthReference, true)
		if err != nil {
			return agent.Agent{}, Errorf(ErrKindInvalidArgument, "%s", err)
		}
	}
	normalized, err := agent.NormalizeUpdateInput(agent.UpdateInput{
		ID:            existing.ID,
		OwnerUserID:   existing.OwnerUserID,
		Label:         label,
		Instructions:  instructions,
		Model:         model,
		AuthReference: authReference,
	})
	if err != nil {
		return agent.Agent{}, Errorf(ErrKindInvalidArgument, "%s", err)
	}
	if err := s.authReferencePolicy.ValidateUsable(ctx, input.OwnerUserID, existing.Provider, normalized.AuthReference); err != nil {
		return agent.Agent{}, err
	}
	if normalized.Model != existing.Model ||
		normalized.AuthReference != existing.AuthReference {
		if err := s.authReferencePolicy.ValidateModelAvailable(ctx, input.OwnerUserID, existing.Provider, normalized.AuthReference, normalized.Model); err != nil {
			return agent.Agent{}, err
		}
	}

	updated := existing
	updated.Label = normalized.Label
	updated.Instructions = normalized.Instructions
	updated.Model = normalized.Model
	updated.AuthReference = normalized.AuthReference
	updated.UpdatedAt = s.clock().UTC()
	if err := s.agentStore.PutAgent(ctx, updated); err != nil {
		if errors.Is(err, storage.ErrConflict) {
			return agent.Agent{}, Errorf(ErrKindAlreadyExists, "agent label already exists")
		}
		return agent.Agent{}, Wrapf(ErrKindInternal, err, "put agent")
	}

	return updated, nil
}

// Delete deletes one user-owned agent profile.
func (s *AgentService) Delete(ctx context.Context, userID string, agentID string) error {
	if agentID == "" {
		return Errorf(ErrKindInvalidArgument, "agent_id is required")
	}
	if s.usagePolicy != nil {
		if err := s.usagePolicy.EnsureAgentNotBoundToActiveCampaigns(ctx, agentID); err != nil {
			return err
		}
	}

	if err := s.agentStore.DeleteAgent(ctx, userID, agentID); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return Errorf(ErrKindNotFound, "agent not found")
		}
		return Wrapf(ErrKindInternal, err, "delete agent")
	}
	return nil
}

// GetAuthState derives a non-mutating runtime auth-health view for an agent.
// This is a best-effort check — no error is returned.
func (s *AgentService) GetAuthState(ctx context.Context, a agent.Agent) AgentAuthState {
	return s.authReferencePolicy.AuthState(ctx, a)
}

// GetActiveCampaignCount returns the number of DRAFT/ACTIVE campaigns bound
// to one AI agent. Returns zero when the usage guard is unavailable.
func (s *AgentService) GetActiveCampaignCount(ctx context.Context, agentID string) (int32, error) {
	if s.agentBindingUsageReader == nil {
		return 0, nil
	}
	return s.agentBindingUsageReader.ActiveCampaignCount(ctx, agentID)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
