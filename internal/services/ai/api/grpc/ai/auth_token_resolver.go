package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/credential"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// authTokenResolver centralizes auth-reference token lookup and refresh so
// multiple RPC roots can share one provider-auth seam.
type authTokenResolver struct {
	credentialStore       storage.CredentialStore
	providerGrantStore    storage.ProviderGrantStore
	providerOAuthAdapters map[provider.Provider]provider.OAuthAdapter
	sealer                SecretSealer
	clock                 func() time.Time
}

// newAuthTokenResolver binds the shared auth-token dependencies once so
// handler roots do not need to duplicate provider-grant refresh logic.
func newAuthTokenResolver(
	credentialStore storage.CredentialStore,
	providerGrantStore storage.ProviderGrantStore,
	providerOAuthAdapters map[provider.Provider]provider.OAuthAdapter,
	sealer SecretSealer,
	clock func() time.Time,
) authTokenResolver {
	if clock == nil {
		clock = time.Now
	}
	return authTokenResolver{
		credentialStore:       credentialStore,
		providerGrantStore:    providerGrantStore,
		providerOAuthAdapters: providerOAuthAdapters,
		sealer:                sealer,
		clock:                 clock,
	}
}

// authTokenResolverForRuntime rebuilds the resolver from the current handler
// state so post-construction mutations (e.g. test clock/adapter swaps) are
// visible to the shared auth-token logic.
func (h *InvocationHandlers) authTokenResolverForRuntime() authTokenResolver {
	return newAuthTokenResolver(
		h.credentialStore,
		h.providerGrantStore,
		h.providerOAuthAdapters,
		h.sealer,
		h.clock,
	)
}

// resolveAgentInvokeToken derives the provider access token for one persisted
// agent record without leaking auth-reference shape into caller code.
func (r authTokenResolver) resolveAgentInvokeToken(ctx context.Context, ownerUserID string, agentRecord storage.AgentRecord) (string, error) {
	authReference, err := agent.AuthReferenceFromRecord(agentRecord)
	if err != nil {
		return "", status.Error(codes.FailedPrecondition, "agent auth reference is invalid")
	}
	return r.resolveAuthReferenceToken(ctx, ownerUserID, providerFromString(agentRecord.Provider), authReference)
}

// resolveAuthReferenceToken opens the invoke-time credential secret for one
// validated auth reference and refreshes provider grants when needed.
func (r authTokenResolver) resolveAuthReferenceToken(ctx context.Context, ownerUserID string, requestedProvider provider.Provider, authReference agent.AuthReference) (string, error) {
	switch authReference.Kind {
	case agent.AuthReferenceKindCredential:
		if r.credentialStore == nil {
			return "", status.Error(codes.Internal, "credential store is not configured")
		}
		credentialRecord, err := r.credentialStore.GetCredential(ctx, authReference.CredentialID())
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return "", status.Error(codes.FailedPrecondition, "credential is unavailable")
			}
			return "", status.Errorf(codes.Internal, "get credential: %v", err)
		}
		if !credential.FromRecord(credentialRecord).IsUsableBy(ownerUserID, requestedProvider) {
			return "", status.Error(codes.FailedPrecondition, "credential must be active and owned by caller")
		}
		credentialSecret, err := r.sealer.Open(credentialRecord.SecretCiphertext)
		if err != nil {
			return "", status.Errorf(codes.Internal, "open credential secret: %v", err)
		}
		return credentialSecret, nil
	case agent.AuthReferenceKindProviderGrant:
		grantRecord, err := r.resolveProviderGrantForInvocation(ctx, ownerUserID, authReference.ProviderGrantID(), requestedProvider)
		if err != nil {
			return "", err
		}
		tokenPlaintext, err := r.sealer.Open(grantRecord.TokenCiphertext)
		if err != nil {
			return "", status.Errorf(codes.Internal, "open provider token: %v", err)
		}
		accessToken, err := providergrant.AccessTokenFromPayload(tokenPlaintext)
		if err != nil {
			return "", status.Errorf(codes.FailedPrecondition, "provider token payload is invalid: %v", err)
		}
		return accessToken, nil
	default:
		return "", status.Error(codes.FailedPrecondition, "agent auth reference is invalid")
	}
}

// refreshProviderGrant refreshes one persisted provider grant through the
// provider adapter and records the resulting lifecycle transition.
func (r authTokenResolver) refreshProviderGrant(ctx context.Context, ownerUserID string, providerGrantID string) (storage.ProviderGrantRecord, error) {
	if r.providerGrantStore == nil {
		return storage.ProviderGrantRecord{}, fmt.Errorf("provider grant store is not configured")
	}
	if r.sealer == nil {
		return storage.ProviderGrantRecord{}, fmt.Errorf("secret sealer is not configured")
	}

	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return storage.ProviderGrantRecord{}, fmt.Errorf("owner user id is required")
	}
	providerGrantID = strings.TrimSpace(providerGrantID)
	if providerGrantID == "" {
		return storage.ProviderGrantRecord{}, fmt.Errorf("provider grant id is required")
	}

	record, err := r.providerGrantStore.GetProviderGrant(ctx, providerGrantID)
	if err != nil {
		return storage.ProviderGrantRecord{}, fmt.Errorf("get provider grant: %w", err)
	}
	if strings.TrimSpace(record.OwnerUserID) != ownerUserID {
		return storage.ProviderGrantRecord{}, storage.ErrNotFound
	}
	if !record.RefreshSupported {
		return storage.ProviderGrantRecord{}, fmt.Errorf("provider grant does not support refresh")
	}

	providerID := providerFromString(record.Provider)
	adapter, ok := r.providerOAuthAdapters[providerID]
	if !ok || adapter == nil {
		return storage.ProviderGrantRecord{}, fmt.Errorf("provider oauth adapter is unavailable")
	}

	tokenPlaintext, err := r.sealer.Open(record.TokenCiphertext)
	if err != nil {
		return storage.ProviderGrantRecord{}, fmt.Errorf("open provider token: %w", err)
	}
	refreshToken, err := providergrant.RefreshTokenFromPayload(tokenPlaintext)
	if err != nil {
		return storage.ProviderGrantRecord{}, fmt.Errorf("extract refresh token: %w", err)
	}

	refreshedAt := r.clock().UTC()
	exchanged, err := adapter.RefreshToken(ctx, provider.RefreshTokenInput{RefreshToken: refreshToken})
	if err != nil {
		if markErr := r.markProviderGrantRefreshFailed(ctx, record, refreshedAt, err); markErr != nil {
			return storage.ProviderGrantRecord{}, markErr
		}
		return storage.ProviderGrantRecord{}, fmt.Errorf("refresh provider token: %w", err)
	}
	newTokenPlaintext := strings.TrimSpace(exchanged.TokenPlaintext)
	if newTokenPlaintext == "" {
		emptyResultErr := fmt.Errorf("provider returned empty token payload")
		if markErr := r.markProviderGrantRefreshFailed(ctx, record, refreshedAt, emptyResultErr); markErr != nil {
			return storage.ProviderGrantRecord{}, markErr
		}
		return storage.ProviderGrantRecord{}, emptyResultErr
	}
	tokenCiphertext, err := r.sealer.Seal(newTokenPlaintext)
	if err != nil {
		return storage.ProviderGrantRecord{}, fmt.Errorf("seal provider token: %w", err)
	}

	updatedGrant, err := providergrant.RecordRefreshSuccess(providergrant.FromRecord(record), tokenCiphertext, exchanged.ExpiresAt, refreshedAt)
	if err != nil {
		return storage.ProviderGrantRecord{}, fmt.Errorf("record provider grant refresh success: %w", err)
	}
	updated := record
	providergrant.ApplyLifecycle(&updated, updatedGrant)
	if err := r.providerGrantStore.PutProviderGrant(ctx, updated); err != nil {
		return storage.ProviderGrantRecord{}, fmt.Errorf("put provider grant: %w", err)
	}
	return updated, nil
}

// markProviderGrantRefreshFailed records the failed refresh transition so the
// next caller sees the latest provider-grant lifecycle state.
func (r authTokenResolver) markProviderGrantRefreshFailed(ctx context.Context, record storage.ProviderGrantRecord, refreshedAt time.Time, refreshErr error) error {
	message := "provider token refresh failed"
	if refreshErr != nil && strings.TrimSpace(refreshErr.Error()) != "" {
		message = strings.TrimSpace(refreshErr.Error())
	}
	updatedGrant, err := providergrant.RecordRefreshFailure(providergrant.FromRecord(record), message, refreshedAt)
	if err != nil {
		return fmt.Errorf("record provider grant refresh failed: %w", err)
	}
	updated := record
	providergrant.ApplyLifecycle(&updated, updatedGrant)
	if err := r.providerGrantStore.PutProviderGrant(ctx, updated); err != nil {
		return fmt.Errorf("mark provider grant refresh failed: %w", err)
	}
	return nil
}

// resolveProviderGrantForInvocation returns one invoke-ready provider grant,
// refreshing it when policy allows and current state requires it.
func (r authTokenResolver) resolveProviderGrantForInvocation(ctx context.Context, ownerUserID string, providerGrantID string, requestedProvider provider.Provider) (storage.ProviderGrantRecord, error) {
	if r.providerGrantStore == nil {
		return storage.ProviderGrantRecord{}, status.Error(codes.Internal, "provider grant store is not configured")
	}

	record, err := r.providerGrantStore.GetProviderGrant(ctx, providerGrantID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return storage.ProviderGrantRecord{}, status.Error(codes.FailedPrecondition, "provider grant is unavailable")
		}
		return storage.ProviderGrantRecord{}, status.Errorf(codes.Internal, "get provider grant: %v", err)
	}
	if strings.TrimSpace(record.OwnerUserID) != strings.TrimSpace(ownerUserID) {
		return storage.ProviderGrantRecord{}, status.Error(codes.FailedPrecondition, "provider grant is unavailable")
	}
	if requestedProvider != "" && providerFromString(record.Provider) != requestedProvider {
		return storage.ProviderGrantRecord{}, status.Error(codes.FailedPrecondition, "provider grant is unavailable")
	}

	now := r.clock().UTC()
	grant := providergrant.FromRecord(record)
	switch grant.Status {
	case providergrant.StatusActive:
		if grant.IsExpired(now) && !record.RefreshSupported {
			return storage.ProviderGrantRecord{}, status.Error(codes.FailedPrecondition, "provider grant is unavailable")
		}
		if grant.ShouldRefresh(now, providerGrantRefreshWindow) {
			refreshed, err := r.refreshProviderGrant(ctx, ownerUserID, providerGrantID)
			if err != nil {
				return storage.ProviderGrantRecord{}, status.Error(codes.FailedPrecondition, "provider grant refresh failed")
			}
			record = refreshed
		}
	case providergrant.StatusRefreshFailed, providergrant.StatusExpired:
		if !record.RefreshSupported {
			return storage.ProviderGrantRecord{}, status.Error(codes.FailedPrecondition, "provider grant is unavailable")
		}
		refreshed, err := r.refreshProviderGrant(ctx, ownerUserID, providerGrantID)
		if err != nil {
			return storage.ProviderGrantRecord{}, status.Error(codes.FailedPrecondition, "provider grant refresh failed")
		}
		record = refreshed
	default:
		return storage.ProviderGrantRecord{}, status.Error(codes.FailedPrecondition, "provider grant is unavailable")
	}
	if !providergrant.ParseStatus(record.Status).IsActive() {
		return storage.ProviderGrantRecord{}, status.Error(codes.FailedPrecondition, "provider grant is unavailable")
	}
	return record, nil
}
