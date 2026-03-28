package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providercatalog"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provideroauth"
	"github.com/louisbranch/fracturing.space/internal/services/ai/secret"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// providerGrantRefreshWindow is the pre-expiry window within which active
// provider grants are proactively refreshed.
const providerGrantRefreshWindow = 2 * time.Minute

// ProviderGrantRuntime owns invoke-time provider-grant refresh policy,
// provider refresh calls, and lifecycle-state persistence.
type ProviderGrantRuntime struct {
	providerGrantStore storage.ProviderGrantStore
	providerRegistry   *providercatalog.Registry
	sealer             secret.Sealer
	clock              Clock
}

// ProviderGrantRuntimeConfig declares dependencies for provider-grant runtime.
type ProviderGrantRuntimeConfig struct {
	ProviderGrantStore storage.ProviderGrantStore
	ProviderRegistry   *providercatalog.Registry
	Sealer             secret.Sealer
	Clock              Clock
}

// NewProviderGrantRuntime builds provider-grant runtime from explicit deps.
func NewProviderGrantRuntime(cfg ProviderGrantRuntimeConfig) *ProviderGrantRuntime {
	return &ProviderGrantRuntime{
		providerGrantStore: cfg.ProviderGrantStore,
		providerRegistry:   cfg.ProviderRegistry,
		sealer:             cfg.Sealer,
		clock:              withDefaultClock(cfg.Clock),
	}
}

// ResolveAccessToken returns one invoke-ready provider access token,
// refreshing the underlying grant when policy requires it.
func (r *ProviderGrantRuntime) ResolveAccessToken(ctx context.Context, ownerUserID string, providerGrantID string, requestedProvider provider.Provider) (string, error) {
	grant, err := r.resolveGrantForInvocation(ctx, ownerUserID, providerGrantID, requestedProvider)
	if err != nil {
		return "", err
	}
	tokenPlaintext, err := r.sealer.Open(grant.TokenCiphertext)
	if err != nil {
		return "", Wrapf(ErrKindInternal, err, "open provider token")
	}
	accessToken, err := provideroauth.AccessTokenFromPayload(tokenPlaintext)
	if err != nil {
		return "", Errorf(ErrKindFailedPrecondition, "provider token payload is invalid: %v", err)
	}
	return accessToken, nil
}

// resolveGrantForInvocation returns one invoke-ready provider grant,
// refreshing it when policy allows and current state requires it.
func (r *ProviderGrantRuntime) resolveGrantForInvocation(ctx context.Context, ownerUserID string, providerGrantID string, requestedProvider provider.Provider) (providergrant.ProviderGrant, error) {
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
			refreshed, err := r.RefreshGrant(ctx, ownerUserID, providerGrantID)
			if err != nil {
				return providergrant.ProviderGrant{}, Errorf(ErrKindFailedPrecondition, "provider grant refresh failed")
			}
			grant = refreshed
		}
	case providergrant.StatusRefreshFailed, providergrant.StatusExpired:
		if !grant.RefreshSupported {
			return providergrant.ProviderGrant{}, Errorf(ErrKindFailedPrecondition, "provider grant is unavailable")
		}
		refreshed, err := r.RefreshGrant(ctx, ownerUserID, providerGrantID)
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

// RefreshGrant refreshes one persisted provider grant through the provider
// adapter and records the resulting lifecycle transition.
func (r *ProviderGrantRuntime) RefreshGrant(ctx context.Context, ownerUserID string, providerGrantID string) (providergrant.ProviderGrant, error) {
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

	adapter, ok := r.providerRegistry.OAuthAdapter(grant.Provider)
	if !ok || adapter == nil {
		return providergrant.ProviderGrant{}, fmt.Errorf("provider oauth adapter is unavailable")
	}

	tokenPlaintext, err := r.sealer.Open(grant.TokenCiphertext)
	if err != nil {
		return providergrant.ProviderGrant{}, fmt.Errorf("open provider token: %w", err)
	}
	refreshToken, err := provideroauth.RefreshTokenFromPayload(tokenPlaintext)
	if err != nil {
		return providergrant.ProviderGrant{}, fmt.Errorf("extract refresh token: %w", err)
	}

	refreshedAt := r.clock().UTC()
	exchanged, err := adapter.RefreshToken(ctx, provideroauth.RefreshTokenInput{RefreshToken: refreshToken})
	if err != nil {
		if markErr := r.markRefreshFailed(ctx, grant, refreshedAt, err); markErr != nil {
			return providergrant.ProviderGrant{}, markErr
		}
		return providergrant.ProviderGrant{}, fmt.Errorf("refresh provider token: %w", err)
	}
	refreshedTokenPlaintext, err := provideroauth.EncodeTokenPayload(exchanged.TokenPayload)
	if err != nil {
		encodeErr := fmt.Errorf("encode provider token payload: %w", err)
		if markErr := r.markRefreshFailed(ctx, grant, refreshedAt, encodeErr); markErr != nil {
			return providergrant.ProviderGrant{}, markErr
		}
		return providergrant.ProviderGrant{}, encodeErr
	}
	tokenCiphertext, err := r.sealer.Seal(refreshedTokenPlaintext)
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

// markRefreshFailed records the failed refresh transition so the next caller
// sees the latest provider-grant lifecycle state.
func (r *ProviderGrantRuntime) markRefreshFailed(ctx context.Context, grant providergrant.ProviderGrant, refreshedAt time.Time, refreshErr error) error {
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
