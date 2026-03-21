package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

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
	credentialStore         storage.CredentialStore
	agentStore              storage.AgentStore
	providerGrantStore      storage.ProviderGrantStore
	accessRequestStore      storage.AccessRequestStore
	providerModelAdapters   map[provider.Provider]provider.ModelAdapter
	authTokenResolver       *AuthTokenResolver
	accessibleAgentResolver *AccessibleAgentResolver
	usageGuard              *UsageGuard
	clock                   Clock
	idGenerator             IDGenerator
}

// AgentServiceConfig declares dependencies for the agent service.
type AgentServiceConfig struct {
	CredentialStore         storage.CredentialStore
	AgentStore              storage.AgentStore
	ProviderGrantStore      storage.ProviderGrantStore
	AccessRequestStore      storage.AccessRequestStore
	ProviderModelAdapters   map[provider.Provider]provider.ModelAdapter
	AuthTokenResolver       *AuthTokenResolver
	AccessibleAgentResolver *AccessibleAgentResolver
	UsageGuard              *UsageGuard
	Clock                   Clock
	IDGenerator             IDGenerator
}

// NewAgentService builds an agent service from explicit deps.
func NewAgentService(cfg AgentServiceConfig) (*AgentService, error) {
	if cfg.AgentStore == nil {
		return nil, fmt.Errorf("ai: NewAgentService: agent store is required")
	}
	if cfg.AuthTokenResolver == nil {
		return nil, fmt.Errorf("ai: NewAgentService: auth token resolver is required")
	}
	if cfg.AccessibleAgentResolver == nil {
		return nil, fmt.Errorf("ai: NewAgentService: accessible agent resolver is required")
	}

	providerModelAdapters := make(map[provider.Provider]provider.ModelAdapter, len(cfg.ProviderModelAdapters))
	for k, v := range cfg.ProviderModelAdapters {
		providerModelAdapters[k] = v
	}

	return &AgentService{
		credentialStore:         cfg.CredentialStore,
		agentStore:              cfg.AgentStore,
		providerGrantStore:      cfg.ProviderGrantStore,
		accessRequestStore:      cfg.AccessRequestStore,
		providerModelAdapters:   providerModelAdapters,
		authTokenResolver:       cfg.AuthTokenResolver,
		accessibleAgentResolver: cfg.AccessibleAgentResolver,
		usageGuard:              cfg.UsageGuard,
		clock:                   withDefaultClock(cfg.Clock),
		idGenerator:             withDefaultIDGenerator(cfg.IDGenerator),
	}, nil
}

// CreateAgentInput is the domain input for creating an agent.
type CreateAgentInput struct {
	OwnerUserID     string
	Label           string
	Instructions    string
	Provider        provider.Provider
	Model           string
	CredentialID    string
	ProviderGrantID string
}

// Create creates a user-owned AI agent profile.
func (s *AgentService) Create(ctx context.Context, input CreateAgentInput) (agent.Agent, error) {
	authReference, err := agent.AuthReferenceFromIDs(input.CredentialID, input.ProviderGrantID, true)
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
	if err := s.validateAgentAuthReferenceForProvider(ctx, input.OwnerUserID, input.Provider, createInput.AuthReference); err != nil {
		return agent.Agent{}, err
	}
	if err := s.validateProviderModelAvailable(ctx, input.OwnerUserID, input.Provider, createInput.AuthReference, createInput.Model); err != nil {
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
	OwnerUserID     string
	Provider        provider.Provider
	CredentialID    string
	ProviderGrantID string
}

// ListProviderModels returns provider-backed model options for one owned auth
// reference.
func (s *AgentService) ListProviderModels(ctx context.Context, input ListProviderModelsInput) ([]provider.Model, error) {
	authReference, err := agent.AuthReferenceFromIDs(input.CredentialID, input.ProviderGrantID, true)
	if err != nil {
		return nil, Errorf(ErrKindInvalidArgument, "%s", err)
	}
	token, err := s.authTokenResolver.ResolveAuthReferenceToken(
		ctx,
		input.OwnerUserID,
		input.Provider,
		authReference,
	)
	if err != nil {
		return nil, err
	}

	adapter, ok := s.providerModelAdapters[input.Provider]
	if !ok || adapter == nil {
		return nil, Errorf(ErrKindFailedPrecondition, "provider model adapter is unavailable")
	}
	models, err := adapter.ListModels(ctx, provider.ListModelsInput{CredentialSecret: token})
	if err != nil {
		return nil, Wrapf(ErrKindInternal, err, "list provider models")
	}
	sort.Slice(models, func(i int, j int) bool {
		if models[i].Created != models[j].Created {
			return models[i].Created > models[j].Created
		}
		return strings.Compare(models[i].ID, models[j].ID) > 0
	})

	// Filter out models with empty IDs.
	result := make([]provider.Model, 0, len(models))
	for _, model := range models {
		if model.ID == "" {
			continue
		}
		result = append(result, model)
	}
	return result, nil
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
	if err := s.validateAgentAuthReferenceForProvider(ctx, userID, a.Provider, a.AuthReference); err != nil {
		return agent.Agent{}, err
	}
	return a, nil
}

// UpdateAgentInput is the domain input for updating an agent.
type UpdateAgentInput struct {
	OwnerUserID     string
	AgentID         string
	Label           string
	Instructions    string
	Model           string
	CredentialID    string
	ProviderGrantID string
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
	if s.usageGuard != nil {
		if err := s.usageGuard.EnsureAgentNotBoundToActiveCampaigns(ctx, existing.ID); err != nil {
			return agent.Agent{}, err
		}
	}

	label := firstNonEmpty(input.Label, existing.Label)
	instructions := firstNonEmpty(input.Instructions, existing.Instructions)
	model := firstNonEmpty(input.Model, existing.Model)
	authReference := existing.AuthReference
	if input.CredentialID != "" || input.ProviderGrantID != "" {
		authReference, err = agent.AuthReferenceFromIDs(input.CredentialID, input.ProviderGrantID, true)
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
	if err := s.validateAgentAuthReferenceForProvider(ctx, input.OwnerUserID, existing.Provider, normalized.AuthReference); err != nil {
		return agent.Agent{}, err
	}
	if normalized.Model != existing.Model ||
		normalized.AuthReference != existing.AuthReference {
		if err := s.validateProviderModelAvailable(ctx, input.OwnerUserID, existing.Provider, normalized.AuthReference, normalized.Model); err != nil {
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
	if s.usageGuard != nil {
		if err := s.usageGuard.EnsureAgentNotBoundToActiveCampaigns(ctx, agentID); err != nil {
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
	switch a.AuthReference.Kind {
	case agent.AuthReferenceKindCredential:
		if s.credentialStore == nil {
			return AgentAuthStateUnavailable
		}
		credentialRecord, err := s.credentialStore.GetCredential(ctx, a.AuthReference.CredentialID())
		if err != nil {
			return AgentAuthStateUnavailable
		}
		switch {
		case credentialRecord.IsUsableBy(a.OwnerUserID, a.Provider):
			return AgentAuthStateReady
		case credentialRecord.Status.IsRevoked():
			return AgentAuthStateRevoked
		default:
			return AgentAuthStateUnavailable
		}
	case agent.AuthReferenceKindProviderGrant:
		if s.providerGrantStore == nil {
			return AgentAuthStateUnavailable
		}
		grant, err := s.providerGrantStore.GetProviderGrant(ctx, a.AuthReference.ProviderGrantID())
		if err != nil {
			return AgentAuthStateUnavailable
		}
		switch {
		case grant.IsUsableBy(a.OwnerUserID, a.Provider):
			return AgentAuthStateReady
		case grant.Status.IsRevoked():
			return AgentAuthStateRevoked
		default:
			return AgentAuthStateUnavailable
		}
	default:
		return AgentAuthStateUnavailable
	}
}

// GetActiveCampaignCount returns the number of DRAFT/ACTIVE campaigns bound
// to one AI agent. Returns zero when the usage guard is unavailable.
func (s *AgentService) GetActiveCampaignCount(ctx context.Context, agentID string) (int32, error) {
	if s.usageGuard == nil {
		return 0, nil
	}
	return s.usageGuard.ActiveCampaignCount(ctx, agentID)
}

// validateAgentAuthReferenceForProvider checks that the auth reference is
// usable by the owner for the requested provider.
func (s *AgentService) validateAgentAuthReferenceForProvider(ctx context.Context, ownerUserID string, requestedProvider provider.Provider, authReference agent.AuthReference) error {
	switch authReference.Kind {
	case agent.AuthReferenceKindCredential:
		if s.credentialStore == nil {
			return Errorf(ErrKindInternal, "credential store is not configured")
		}
		credentialRecord, err := s.credentialStore.GetCredential(ctx, authReference.CredentialID())
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return Errorf(ErrKindFailedPrecondition, "credential is unavailable")
			}
			return Wrapf(ErrKindInternal, err, "get credential")
		}
		if !credentialRecord.IsUsableBy(ownerUserID, requestedProvider) {
			return Errorf(ErrKindFailedPrecondition, "credential must be active and owned by caller")
		}
		return nil
	case agent.AuthReferenceKindProviderGrant:
		if s.providerGrantStore == nil {
			return Errorf(ErrKindInternal, "provider grant store is not configured")
		}
		grant, err := s.providerGrantStore.GetProviderGrant(ctx, authReference.ProviderGrantID())
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return Errorf(ErrKindFailedPrecondition, "provider grant is unavailable")
			}
			return Wrapf(ErrKindInternal, err, "get provider grant")
		}
		if !grant.IsUsableBy(ownerUserID, requestedProvider) {
			return Errorf(ErrKindFailedPrecondition, "provider grant must be active and owned by caller")
		}
		return nil
	default:
		return Errorf(ErrKindInvalidArgument, "exactly one agent auth reference is required")
	}
}

// validateProviderModelAvailable checks that a specific model is available
// through the given auth reference.
func (s *AgentService) validateProviderModelAvailable(ctx context.Context, ownerUserID string, requestedProvider provider.Provider, authReference agent.AuthReference, model string) error {
	if model == "" {
		return Errorf(ErrKindInvalidArgument, "model is required")
	}

	token, err := s.authTokenResolver.ResolveAuthReferenceToken(ctx, ownerUserID, requestedProvider, authReference)
	if err != nil {
		return err
	}
	adapter, ok := s.providerModelAdapters[requestedProvider]
	if !ok || adapter == nil {
		return Errorf(ErrKindFailedPrecondition, "provider model adapter is unavailable")
	}
	models, err := adapter.ListModels(ctx, provider.ListModelsInput{CredentialSecret: token})
	if err != nil {
		return Wrapf(ErrKindInternal, err, "list provider models")
	}
	for _, candidate := range models {
		if candidate.ID == model {
			return nil
		}
	}
	return Errorf(ErrKindFailedPrecondition, "model is unavailable for the selected auth reference")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
