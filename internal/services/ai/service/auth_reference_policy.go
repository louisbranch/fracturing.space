package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providercatalog"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// AuthReferencePolicy centralizes reusable auth-reference policy for agent
// workflows: validating owner/provider usability, deriving auth readiness, and
// checking model availability for the selected auth material.
type AuthReferencePolicy struct {
	credentialStore      storage.CredentialStore
	providerGrantStore   storage.ProviderGrantStore
	providerRegistry     *providercatalog.Registry
	authMaterialResolver *AuthMaterialResolver
}

// AuthReferencePolicyConfig declares dependencies for auth-reference policy.
type AuthReferencePolicyConfig struct {
	CredentialStore      storage.CredentialStore
	ProviderGrantStore   storage.ProviderGrantStore
	ProviderRegistry     *providercatalog.Registry
	AuthMaterialResolver *AuthMaterialResolver
}

// NewAuthReferencePolicy builds auth-reference policy from explicit deps.
func NewAuthReferencePolicy(cfg AuthReferencePolicyConfig) (*AuthReferencePolicy, error) {
	if cfg.AuthMaterialResolver == nil {
		return nil, fmt.Errorf("ai: NewAuthReferencePolicy: auth material resolver is required")
	}
	if err := RequireProviderRegistry(cfg.ProviderRegistry, "NewAuthReferencePolicy"); err != nil {
		return nil, err
	}
	return &AuthReferencePolicy{
		credentialStore:      cfg.CredentialStore,
		providerGrantStore:   cfg.ProviderGrantStore,
		providerRegistry:     cfg.ProviderRegistry,
		authMaterialResolver: cfg.AuthMaterialResolver,
	}, nil
}

// ValidateUsable checks that the auth reference is usable by the owner for the
// requested provider.
func (p *AuthReferencePolicy) ValidateUsable(ctx context.Context, ownerUserID string, requestedProvider provider.Provider, authReference agent.AuthReference) error {
	switch authReference.Kind {
	case agent.AuthReferenceKindCredential:
		if p.credentialStore == nil {
			return Errorf(ErrKindInternal, "credential store is not configured")
		}
		credentialRecord, err := p.credentialStore.GetCredential(ctx, authReference.CredentialID())
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
		if p.providerGrantStore == nil {
			return Errorf(ErrKindInternal, "provider grant store is not configured")
		}
		grant, err := p.providerGrantStore.GetProviderGrant(ctx, authReference.ProviderGrantID())
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

// ListProviderModels returns provider-backed model options for one owned auth
// reference.
func (p *AuthReferencePolicy) ListProviderModels(ctx context.Context, ownerUserID string, requestedProvider provider.Provider, authReference agent.AuthReference) ([]provider.Model, error) {
	adapter, ok := p.providerRegistry.ModelAdapter(requestedProvider)
	if !ok || adapter == nil {
		return nil, Errorf(ErrKindFailedPrecondition, "provider model adapter is unavailable")
	}

	token, err := p.authMaterialResolver.ResolveAuthReferenceToken(ctx, ownerUserID, requestedProvider, authReference)
	if err != nil {
		return nil, err
	}
	models, err := adapter.ListModels(ctx, provider.ListModelsInput{AuthToken: token})
	if err != nil {
		return nil, Wrapf(ErrKindInternal, err, "list provider models")
	}

	sort.Slice(models, func(i int, j int) bool {
		return strings.Compare(models[i].ID, models[j].ID) < 0
	})

	filtered := make([]provider.Model, 0, len(models))
	for _, model := range models {
		if model.ID == "" {
			continue
		}
		filtered = append(filtered, model)
	}
	return filtered, nil
}

// ValidateModelAvailable checks that a specific model is available through the
// given auth reference.
func (p *AuthReferencePolicy) ValidateModelAvailable(ctx context.Context, ownerUserID string, requestedProvider provider.Provider, authReference agent.AuthReference, model string) error {
	if model == "" {
		return Errorf(ErrKindInvalidArgument, "model is required")
	}
	models, err := p.ListProviderModels(ctx, ownerUserID, requestedProvider, authReference)
	if err != nil {
		return err
	}
	for _, candidate := range models {
		if candidate.ID == model {
			return nil
		}
	}
	return Errorf(ErrKindFailedPrecondition, "model is unavailable for the selected auth reference")
}

// AuthState derives a non-mutating runtime auth-health view for one agent.
// This is a best-effort check; missing stores or records return unavailable.
func (p *AuthReferencePolicy) AuthState(ctx context.Context, a agent.Agent) AgentAuthState {
	switch a.AuthReference.Kind {
	case agent.AuthReferenceKindCredential:
		if p.credentialStore == nil {
			return AgentAuthStateUnavailable
		}
		credentialRecord, err := p.credentialStore.GetCredential(ctx, a.AuthReference.CredentialID())
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
		if p.providerGrantStore == nil {
			return AgentAuthStateUnavailable
		}
		grant, err := p.providerGrantStore.GetProviderGrant(ctx, a.AuthReference.ProviderGrantID())
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
