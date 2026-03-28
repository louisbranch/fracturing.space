package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/secret"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// AuthMaterialResolver resolves invoke-time auth material for validated auth
// references without owning provider-grant lifecycle mutations itself.
type AuthMaterialResolver struct {
	credentialStore      storage.CredentialStore
	sealer               secret.Sealer
	providerGrantRuntime *ProviderGrantRuntime
}

// AuthMaterialResolverConfig declares dependencies for auth-material
// resolution.
type AuthMaterialResolverConfig struct {
	CredentialStore      storage.CredentialStore
	Sealer               secret.Sealer
	ProviderGrantRuntime *ProviderGrantRuntime
}

// NewAuthMaterialResolver builds an auth-material resolver from explicit deps.
func NewAuthMaterialResolver(cfg AuthMaterialResolverConfig) *AuthMaterialResolver {
	return &AuthMaterialResolver{
		credentialStore:      cfg.CredentialStore,
		sealer:               cfg.Sealer,
		providerGrantRuntime: cfg.ProviderGrantRuntime,
	}
}

// ResolveAgentInvokeToken derives the provider access token for one persisted
// agent record without leaking auth-reference shape into caller code.
func (r *AuthMaterialResolver) ResolveAgentInvokeToken(ctx context.Context, ownerUserID string, a agent.Agent) (string, error) {
	return r.ResolveAuthReferenceToken(ctx, ownerUserID, a.Provider, a.AuthReference)
}

// ResolveAuthReferenceToken opens the invoke-time credential secret for one
// validated auth reference and delegates provider-grant runtime behavior to the
// dedicated grant runtime seam.
func (r *AuthMaterialResolver) ResolveAuthReferenceToken(ctx context.Context, ownerUserID string, requestedProvider provider.Provider, authReference agent.AuthReference) (string, error) {
	switch authReference.Kind {
	case agent.AuthReferenceKindCredential:
		if r.credentialStore == nil {
			return "", Errorf(ErrKindInternal, "credential store is not configured")
		}
		cred, err := r.credentialStore.GetCredential(ctx, authReference.CredentialID())
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return "", Errorf(ErrKindFailedPrecondition, "credential is unavailable")
			}
			return "", Wrapf(ErrKindInternal, err, "get credential")
		}
		if !cred.IsUsableBy(ownerUserID, requestedProvider) {
			return "", Errorf(ErrKindFailedPrecondition, "credential must be active and owned by caller")
		}
		credentialSecret, err := r.sealer.Open(cred.SecretCiphertext)
		if err != nil {
			return "", Wrapf(ErrKindInternal, err, "open credential secret")
		}
		return credentialSecret, nil
	case agent.AuthReferenceKindProviderGrant:
		if r.providerGrantRuntime == nil {
			return "", Errorf(ErrKindInternal, "provider grant runtime is not configured")
		}
		return r.providerGrantRuntime.ResolveAccessToken(ctx, ownerUserID, authReference.ProviderGrantID(), requestedProvider)
	default:
		return "", Errorf(ErrKindFailedPrecondition, "agent auth reference is invalid")
	}
}

// RequireAuthMaterialResolver is a composition-time guard for services that
// need invoke-time auth material.
func RequireAuthMaterialResolver(resolver *AuthMaterialResolver, consumer string) error {
	if resolver != nil {
		return nil
	}
	return fmt.Errorf("ai: %s: auth material resolver is required", consumer)
}
