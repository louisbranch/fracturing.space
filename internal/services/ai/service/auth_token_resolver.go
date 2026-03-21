package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"github.com/louisbranch/fracturing.space/internal/services/ai/secret"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// providerGrantRefreshWindow is the pre-expiry window within which active
// provider grants are proactively refreshed.
const providerGrantRefreshWindow = 2 * time.Minute

// AuthTokenResolver resolves auth-reference tokens for agent invocations.
// It centralizes credential lookup, provider-grant validation, and grant
// refresh so multiple service consumers share one provider-auth seam.
type AuthTokenResolver struct {
	credentialStore       storage.CredentialStore
	providerGrantStore    storage.ProviderGrantStore
	providerOAuthAdapters map[provider.Provider]provider.OAuthAdapter
	sealer                secret.Sealer
	clock                 Clock
}

// AuthTokenResolverConfig declares dependencies for the auth token resolver.
type AuthTokenResolverConfig struct {
	CredentialStore       storage.CredentialStore
	ProviderGrantStore    storage.ProviderGrantStore
	ProviderOAuthAdapters map[provider.Provider]provider.OAuthAdapter
	Sealer                secret.Sealer
	Clock                 Clock
}

// NewAuthTokenResolver builds an auth token resolver from explicit deps.
func NewAuthTokenResolver(cfg AuthTokenResolverConfig) *AuthTokenResolver {
	return &AuthTokenResolver{
		credentialStore:       cfg.CredentialStore,
		providerGrantStore:    cfg.ProviderGrantStore,
		providerOAuthAdapters: copyOAuthAdapters(cfg.ProviderOAuthAdapters),
		sealer:                cfg.Sealer,
		clock:                 withDefaultClock(cfg.Clock),
	}
}

// ResolveAgentInvokeToken derives the provider access token for one persisted
// agent record without leaking auth-reference shape into caller code.
func (r *AuthTokenResolver) ResolveAgentInvokeToken(ctx context.Context, ownerUserID string, a agent.Agent) (string, error) {
	return r.ResolveAuthReferenceToken(ctx, ownerUserID, a.Provider, a.AuthReference)
}

// ResolveAuthReferenceToken opens the invoke-time credential secret for one
// validated auth reference and refreshes provider grants when needed.
func (r *AuthTokenResolver) ResolveAuthReferenceToken(ctx context.Context, ownerUserID string, requestedProvider provider.Provider, authReference agent.AuthReference) (string, error) {
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
		grant, err := r.resolveProviderGrantForInvocation(ctx, ownerUserID, authReference.ProviderGrantID(), requestedProvider)
		if err != nil {
			return "", err
		}
		tokenPlaintext, err := r.sealer.Open(grant.TokenCiphertext)
		if err != nil {
			return "", Wrapf(ErrKindInternal, err, "open provider token")
		}
		accessToken, err := providergrant.AccessTokenFromPayload(tokenPlaintext)
		if err != nil {
			return "", Errorf(ErrKindFailedPrecondition, "provider token payload is invalid: %v", err)
		}
		return accessToken, nil
	default:
		return "", Errorf(ErrKindFailedPrecondition, "agent auth reference is invalid")
	}
}

// resolveProviderGrantForInvocation returns one invoke-ready provider grant,
// refreshing it when policy allows and current state requires it.
func (r *AuthTokenResolver) resolveProviderGrantForInvocation(ctx context.Context, ownerUserID string, providerGrantID string, requestedProvider provider.Provider) (providergrant.ProviderGrant, error) {
	if r.providerGrantStore == nil {
		return providergrant.ProviderGrant{}, Errorf(ErrKindInternal, "provider grant store is not configured")
	}

	grant, err := r.providerGrantStore.GetProviderGrant(ctx, providerGrantID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return providergrant.ProviderGrant{}, Errorf(ErrKindFailedPrecondition, "provider grant is unavailable")
		}
		return providergrant.ProviderGrant{}, Wrapf(ErrKindInternal, err, "get provider grant")
	}
	if grant.OwnerUserID != ownerUserID {
		return providergrant.ProviderGrant{}, Errorf(ErrKindFailedPrecondition, "provider grant is unavailable")
	}
	if requestedProvider != "" && grant.Provider != requestedProvider {
		return providergrant.ProviderGrant{}, Errorf(ErrKindFailedPrecondition, "provider grant is unavailable")
	}

	now := r.clock().UTC()
	switch grant.Status {
	case providergrant.StatusActive:
		if grant.IsExpired(now) && !grant.RefreshSupported {
			return providergrant.ProviderGrant{}, Errorf(ErrKindFailedPrecondition, "provider grant is unavailable")
		}
		if grant.ShouldRefresh(now, providerGrantRefreshWindow) {
			refreshed, err := r.refreshProviderGrant(ctx, ownerUserID, providerGrantID)
			if err != nil {
				return providergrant.ProviderGrant{}, Errorf(ErrKindFailedPrecondition, "provider grant refresh failed")
			}
			grant = refreshed
		}
	case providergrant.StatusRefreshFailed, providergrant.StatusExpired:
		if !grant.RefreshSupported {
			return providergrant.ProviderGrant{}, Errorf(ErrKindFailedPrecondition, "provider grant is unavailable")
		}
		refreshed, err := r.refreshProviderGrant(ctx, ownerUserID, providerGrantID)
		if err != nil {
			return providergrant.ProviderGrant{}, Errorf(ErrKindFailedPrecondition, "provider grant refresh failed")
		}
		grant = refreshed
	default:
		return providergrant.ProviderGrant{}, Errorf(ErrKindFailedPrecondition, "provider grant is unavailable")
	}
	if !grant.Status.IsActive() {
		return providergrant.ProviderGrant{}, Errorf(ErrKindFailedPrecondition, "provider grant is unavailable")
	}
	return grant, nil
}

// refreshProviderGrant refreshes one persisted provider grant through the
// provider adapter and records the resulting lifecycle transition.
func (r *AuthTokenResolver) refreshProviderGrant(ctx context.Context, ownerUserID string, providerGrantID string) (providergrant.ProviderGrant, error) {
	if r.providerGrantStore == nil {
		return providergrant.ProviderGrant{}, fmt.Errorf("provider grant store is not configured")
	}
	if r.sealer == nil {
		return providergrant.ProviderGrant{}, fmt.Errorf("secret sealer is not configured")
	}

	if ownerUserID == "" {
		return providergrant.ProviderGrant{}, fmt.Errorf("owner user id is required")
	}
	if providerGrantID == "" {
		return providergrant.ProviderGrant{}, fmt.Errorf("provider grant id is required")
	}

	grant, err := r.providerGrantStore.GetProviderGrant(ctx, providerGrantID)
	if err != nil {
		return providergrant.ProviderGrant{}, fmt.Errorf("get provider grant: %w", err)
	}
	if grant.OwnerUserID != ownerUserID {
		return providergrant.ProviderGrant{}, storage.ErrNotFound
	}
	if !grant.RefreshSupported {
		return providergrant.ProviderGrant{}, fmt.Errorf("provider grant does not support refresh")
	}

	adapter, ok := r.providerOAuthAdapters[grant.Provider]
	if !ok || adapter == nil {
		return providergrant.ProviderGrant{}, fmt.Errorf("provider oauth adapter is unavailable")
	}

	tokenPlaintext, err := r.sealer.Open(grant.TokenCiphertext)
	if err != nil {
		return providergrant.ProviderGrant{}, fmt.Errorf("open provider token: %w", err)
	}
	refreshToken, err := providergrant.RefreshTokenFromPayload(tokenPlaintext)
	if err != nil {
		return providergrant.ProviderGrant{}, fmt.Errorf("extract refresh token: %w", err)
	}

	refreshedAt := r.clock().UTC()
	exchanged, err := adapter.RefreshToken(ctx, provider.RefreshTokenInput{RefreshToken: refreshToken})
	if err != nil {
		if markErr := r.markProviderGrantRefreshFailed(ctx, grant, refreshedAt, err); markErr != nil {
			return providergrant.ProviderGrant{}, markErr
		}
		return providergrant.ProviderGrant{}, fmt.Errorf("refresh provider token: %w", err)
	}
	if exchanged.TokenPlaintext == "" {
		emptyResultErr := fmt.Errorf("provider returned empty token payload")
		if markErr := r.markProviderGrantRefreshFailed(ctx, grant, refreshedAt, emptyResultErr); markErr != nil {
			return providergrant.ProviderGrant{}, markErr
		}
		return providergrant.ProviderGrant{}, emptyResultErr
	}
	tokenCiphertext, err := r.sealer.Seal(exchanged.TokenPlaintext)
	if err != nil {
		return providergrant.ProviderGrant{}, fmt.Errorf("seal provider token: %w", err)
	}

	updated, err := providergrant.RecordRefreshSuccess(grant, tokenCiphertext, exchanged.ExpiresAt, refreshedAt)
	if err != nil {
		return providergrant.ProviderGrant{}, fmt.Errorf("record provider grant refresh success: %w", err)
	}
	if err := r.providerGrantStore.PutProviderGrant(ctx, updated); err != nil {
		return providergrant.ProviderGrant{}, fmt.Errorf("put provider grant: %w", err)
	}
	return updated, nil
}

// markProviderGrantRefreshFailed records the failed refresh transition so the
// next caller sees the latest provider-grant lifecycle state.
func (r *AuthTokenResolver) markProviderGrantRefreshFailed(ctx context.Context, grant providergrant.ProviderGrant, refreshedAt time.Time, refreshErr error) error {
	message := "provider token refresh failed"
	if refreshErr != nil && refreshErr.Error() != "" {
		message = refreshErr.Error()
	}
	updated, err := providergrant.RecordRefreshFailure(grant, message, refreshedAt)
	if err != nil {
		return fmt.Errorf("record provider grant refresh failed: %w", err)
	}
	if err := r.providerGrantStore.PutProviderGrant(ctx, updated); err != nil {
		return fmt.Errorf("mark provider grant refresh failed: %w", err)
	}
	return nil
}
