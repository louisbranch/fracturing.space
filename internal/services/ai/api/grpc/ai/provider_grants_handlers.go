package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ProviderGrantHandlers serves provider grant RPCs with explicit dependencies.
type ProviderGrantHandlers struct {
	aiv1.UnimplementedProviderGrantServiceServer

	providerGrantStore    storage.ProviderGrantStore
	connectSessionStore   storage.ProviderConnectSessionStore
	sealer                SecretSealer
	providerOAuthAdapters map[provider.Provider]ProviderOAuthAdapter
	clock                 func() time.Time
	idGenerator           func() (string, error)
	codeVerifierGenerator func() (string, error)
	usageGuard            authReferenceUsageGuard
}

// ProviderGrantHandlersConfig declares the dependencies for provider-grant RPCs.
type ProviderGrantHandlersConfig struct {
	ProviderGrantStore    storage.ProviderGrantStore
	ConnectSessionStore   storage.ProviderConnectSessionStore
	AgentStore            storage.AgentStore
	GameCampaignAIClient  gamev1.CampaignAIServiceClient
	Sealer                SecretSealer
	ProviderOAuthAdapters map[provider.Provider]ProviderOAuthAdapter
	Clock                 func() time.Time
	IDGenerator           func() (string, error)
	CodeVerifierGenerator func() (string, error)
}

// NewProviderGrantHandlers builds a provider-grant RPC server from explicit deps.
func NewProviderGrantHandlers(cfg ProviderGrantHandlersConfig) *ProviderGrantHandlers {
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	idGenerator := cfg.IDGenerator
	if idGenerator == nil {
		idGenerator = id.NewID
	}
	codeVerifierGenerator := cfg.CodeVerifierGenerator
	if codeVerifierGenerator == nil {
		codeVerifierGenerator = generatePKCECodeVerifier
	}
	return &ProviderGrantHandlers{
		providerGrantStore:    cfg.ProviderGrantStore,
		connectSessionStore:   cfg.ConnectSessionStore,
		sealer:                cfg.Sealer,
		providerOAuthAdapters: newProviderOAuthAdapters(cfg.ProviderOAuthAdapters),
		clock:                 clock,
		idGenerator:           idGenerator,
		codeVerifierGenerator: codeVerifierGenerator,
		usageGuard:            newAuthReferenceUsageGuard(cfg.AgentStore, cfg.GameCampaignAIClient),
	}
}

// StartProviderConnect starts a provider OAuth grant handshake.
func (h *ProviderGrantHandlers) StartProviderConnect(ctx context.Context, in *aiv1.StartProviderConnectRequest) (*aiv1.StartProviderConnectResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "start provider connect request is required")
	}
	if h == nil || h.providerGrantStore == nil {
		return nil, status.Error(codes.Internal, "provider grant store is not configured")
	}
	if h.connectSessionStore == nil {
		return nil, status.Error(codes.Internal, "provider connect session store is not configured")
	}
	if h.sealer == nil {
		return nil, status.Error(codes.Internal, "secret sealer is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	providerID, err := providerFromProto(in.GetProvider())
	if err != nil {
		return nil, err
	}

	sessionID, err := h.idGenerator()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate connect session id: %v", err)
	}
	state, err := h.idGenerator()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate connect state: %v", err)
	}
	codeVerifier, err := h.codeVerifierGenerator()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate code verifier: %v", err)
	}
	codeVerifier = strings.TrimSpace(codeVerifier)
	if !isValidPKCECodeVerifier(codeVerifier) {
		return nil, status.Error(codes.Internal, "generate code verifier: value is invalid")
	}
	codeChallenge := pkceCodeChallengeS256(codeVerifier)
	codeVerifierCiphertext, err := h.sealer.Seal(codeVerifier)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "seal code verifier: %v", err)
	}

	now := h.clock().UTC()
	expiresAt := now.Add(10 * time.Minute)
	record := storage.ProviderConnectSessionRecord{
		ID:                     sessionID,
		OwnerUserID:            userID,
		Provider:               string(providerID),
		Status:                 "pending",
		RequestedScopes:        normalizeScopes(in.GetRequestedScopes()),
		StateHash:              hashState(state),
		CodeVerifierCiphertext: codeVerifierCiphertext,
		CreatedAt:              now,
		UpdatedAt:              now,
		ExpiresAt:              expiresAt,
	}
	if err := h.connectSessionStore.PutProviderConnectSession(ctx, record); err != nil {
		return nil, status.Errorf(codes.Internal, "put provider connect session: %v", err)
	}

	adapter, ok := h.providerOAuthAdapters[providerID]
	if !ok || adapter == nil {
		return nil, status.Error(codes.FailedPrecondition, "provider oauth adapter is unavailable")
	}
	authorizationURL, err := adapter.BuildAuthorizationURL(ProviderAuthorizationURLInput{
		State:           state,
		CodeChallenge:   codeChallenge,
		RequestedScopes: record.RequestedScopes,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "build authorization url: %v", err)
	}
	return &aiv1.StartProviderConnectResponse{
		ConnectSessionId: sessionID,
		State:            state,
		AuthorizationUrl: authorizationURL,
		ExpiresAt:        timestamppb.New(expiresAt),
	}, nil
}

// FinishProviderConnect completes a provider OAuth grant handshake.
func (h *ProviderGrantHandlers) FinishProviderConnect(ctx context.Context, in *aiv1.FinishProviderConnectRequest) (*aiv1.FinishProviderConnectResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "finish provider connect request is required")
	}
	if h == nil || h.providerGrantStore == nil {
		return nil, status.Error(codes.Internal, "provider grant store is not configured")
	}
	if h.connectSessionStore == nil {
		return nil, status.Error(codes.Internal, "provider connect session store is not configured")
	}
	if h.sealer == nil {
		return nil, status.Error(codes.Internal, "secret sealer is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	sessionID := strings.TrimSpace(in.GetConnectSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "connect_session_id is required")
	}
	state := strings.TrimSpace(in.GetState())
	if state == "" {
		return nil, status.Error(codes.InvalidArgument, "state is required")
	}
	authorizationCode := strings.TrimSpace(in.GetAuthorizationCode())
	if authorizationCode == "" {
		return nil, status.Error(codes.InvalidArgument, "authorization_code is required")
	}

	session, err := h.connectSessionStore.GetProviderConnectSession(ctx, sessionID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "connect session not found")
		}
		return nil, status.Errorf(codes.Internal, "get connect session: %v", err)
	}
	if strings.TrimSpace(session.OwnerUserID) != userID {
		return nil, status.Error(codes.NotFound, "connect session not found")
	}
	if !strings.EqualFold(strings.TrimSpace(session.Status), "pending") {
		return nil, status.Error(codes.FailedPrecondition, "connect session is no longer pending")
	}
	if h.clock().UTC().After(session.ExpiresAt) {
		return nil, status.Error(codes.FailedPrecondition, "connect session expired")
	}
	if hashState(state) != strings.TrimSpace(session.StateHash) {
		return nil, status.Error(codes.FailedPrecondition, "state mismatch")
	}

	providerID := providerFromString(session.Provider)
	if providerID != provider.OpenAI {
		return nil, status.Error(codes.FailedPrecondition, "provider is unavailable")
	}
	adapter, ok := h.providerOAuthAdapters[providerID]
	if !ok || adapter == nil {
		return nil, status.Error(codes.FailedPrecondition, "provider oauth adapter is unavailable")
	}
	codeVerifier, err := h.sealer.Open(session.CodeVerifierCiphertext)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "open code verifier: %v", err)
	}
	exchanged, err := adapter.ExchangeAuthorizationCode(ctx, ProviderAuthorizationCodeInput{
		AuthorizationCode: authorizationCode,
		CodeVerifier:      codeVerifier,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "exchange authorization code: %v", err)
	}
	tokenPlaintext := strings.TrimSpace(exchanged.TokenPlaintext)
	if tokenPlaintext == "" {
		return nil, status.Error(codes.Internal, "provider returned empty token payload")
	}
	tokenCiphertext, err := h.sealer.Seal(tokenPlaintext)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "seal provider token: %v", err)
	}

	created, err := providergrant.Create(providergrant.CreateInput{
		OwnerUserID:      userID,
		Provider:         providerID,
		GrantedScopes:    session.RequestedScopes,
		TokenCiphertext:  tokenCiphertext,
		RefreshSupported: exchanged.RefreshSupported,
		ExpiresAt:        exchanged.ExpiresAt,
	}, h.clock, h.idGenerator)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	record := storage.ProviderGrantRecord{
		ID:               created.ID,
		OwnerUserID:      created.OwnerUserID,
		Provider:         string(created.Provider),
		GrantedScopes:    created.GrantedScopes,
		TokenCiphertext:  created.TokenCiphertext,
		RefreshSupported: created.RefreshSupported,
		Status:           string(created.Status),
		LastRefreshError: strings.TrimSpace(exchanged.LastRefreshError),
		CreatedAt:        created.CreatedAt,
		UpdatedAt:        created.UpdatedAt,
		RevokedAt:        created.RevokedAt,
		ExpiresAt:        created.ExpiresAt,
		LastRefreshedAt:  created.RefreshedAt,
	}
	if err := h.providerGrantStore.PutProviderGrant(ctx, record); err != nil {
		return nil, status.Errorf(codes.Internal, "put provider grant: %v", err)
	}

	completedAt := h.clock().UTC()
	if err := h.connectSessionStore.CompleteProviderConnectSession(ctx, userID, sessionID, completedAt); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "connect session not found")
		}
		return nil, status.Errorf(codes.Internal, "complete connect session: %v", err)
	}
	return &aiv1.FinishProviderConnectResponse{ProviderGrant: providerGrantToProto(record)}, nil
}

// ListProviderGrants returns a page of provider grants owned by the caller.
func (h *ProviderGrantHandlers) ListProviderGrants(ctx context.Context, in *aiv1.ListProviderGrantsRequest) (*aiv1.ListProviderGrantsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list provider grants request is required")
	}
	if h == nil || h.providerGrantStore == nil {
		return nil, status.Error(codes.Internal, "provider grant store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	filter, err := providerGrantFilterFromRequest(in)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	page, err := h.providerGrantStore.ListProviderGrantsByOwner(ctx, userID, clampPageSize(in.GetPageSize()), in.GetPageToken(), filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list provider grants: %v", err)
	}

	resp := &aiv1.ListProviderGrantsResponse{
		NextPageToken:  page.NextPageToken,
		ProviderGrants: make([]*aiv1.ProviderGrant, 0, len(page.ProviderGrants)),
	}
	for _, rec := range page.ProviderGrants {
		resp.ProviderGrants = append(resp.ProviderGrants, providerGrantToProto(rec))
	}
	return resp, nil
}

// RevokeProviderGrant revokes one provider grant owned by the caller.
func (h *ProviderGrantHandlers) RevokeProviderGrant(ctx context.Context, in *aiv1.RevokeProviderGrantRequest) (*aiv1.RevokeProviderGrantResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "revoke provider grant request is required")
	}
	if h == nil || h.providerGrantStore == nil {
		return nil, status.Error(codes.Internal, "provider grant store is not configured")
	}
	if h.sealer == nil {
		return nil, status.Error(codes.Internal, "secret sealer is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	providerGrantID := strings.TrimSpace(in.GetProviderGrantId())
	if providerGrantID == "" {
		return nil, status.Error(codes.InvalidArgument, "provider_grant_id is required")
	}
	if err := h.usageGuard.ensureProviderGrantNotBoundToActiveCampaigns(ctx, userID, providerGrantID); err != nil {
		return nil, err
	}

	record, err := h.providerGrantStore.GetProviderGrant(ctx, providerGrantID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "provider grant not found")
		}
		return nil, status.Errorf(codes.Internal, "get provider grant: %v", err)
	}
	if strings.TrimSpace(record.OwnerUserID) != userID {
		return nil, status.Error(codes.NotFound, "provider grant not found")
	}

	providerID := providerFromString(record.Provider)
	if adapter, ok := h.providerOAuthAdapters[providerID]; ok && adapter != nil {
		tokenPlaintext, err := h.sealer.Open(record.TokenCiphertext)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "open provider token: %v", err)
		}
		tokenForRevoke, err := revokeTokenFromTokenPayload(tokenPlaintext)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "derive provider revoke token: %v", err)
		}
		if err := adapter.RevokeToken(ctx, ProviderRevokeTokenInput{Token: tokenForRevoke}); err != nil {
			return nil, status.Errorf(codes.Internal, "revoke provider token: %v", err)
		}
	}

	revokedAt := h.clock().UTC()
	if err := h.providerGrantStore.RevokeProviderGrant(ctx, userID, providerGrantID, revokedAt); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "provider grant not found")
		}
		return nil, status.Errorf(codes.Internal, "revoke provider grant: %v", err)
	}

	record, err = h.providerGrantStore.GetProviderGrant(ctx, providerGrantID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "provider grant not found")
		}
		return nil, status.Errorf(codes.Internal, "get provider grant: %v", err)
	}
	if strings.TrimSpace(record.OwnerUserID) != userID {
		return nil, status.Error(codes.NotFound, "provider grant not found")
	}
	return &aiv1.RevokeProviderGrantResponse{ProviderGrant: providerGrantToProto(record)}, nil
}

func (s *Service) refreshProviderGrant(ctx context.Context, ownerUserID string, providerGrantID string) (storage.ProviderGrantRecord, error) {
	if s.providerGrantStore == nil {
		return storage.ProviderGrantRecord{}, fmt.Errorf("provider grant store is not configured")
	}
	if s.sealer == nil {
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

	record, err := s.providerGrantStore.GetProviderGrant(ctx, providerGrantID)
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
	adapter, ok := s.providerOAuthAdapters[providerID]
	if !ok || adapter == nil {
		return storage.ProviderGrantRecord{}, fmt.Errorf("provider oauth adapter is unavailable")
	}

	tokenPlaintext, err := s.sealer.Open(record.TokenCiphertext)
	if err != nil {
		return storage.ProviderGrantRecord{}, fmt.Errorf("open provider token: %w", err)
	}
	refreshToken, err := refreshTokenFromTokenPayload(tokenPlaintext)
	if err != nil {
		return storage.ProviderGrantRecord{}, fmt.Errorf("extract refresh token: %w", err)
	}

	refreshedAt := s.clock().UTC()
	exchanged, err := adapter.RefreshToken(ctx, ProviderRefreshTokenInput{
		RefreshToken: refreshToken,
	})
	if err != nil {
		if markErr := s.markProviderGrantRefreshFailed(ctx, record, refreshedAt, err); markErr != nil {
			return storage.ProviderGrantRecord{}, markErr
		}
		return storage.ProviderGrantRecord{}, fmt.Errorf("refresh provider token: %w", err)
	}
	newTokenPlaintext := strings.TrimSpace(exchanged.TokenPlaintext)
	if newTokenPlaintext == "" {
		emptyResultErr := fmt.Errorf("provider returned empty token payload")
		if markErr := s.markProviderGrantRefreshFailed(ctx, record, refreshedAt, emptyResultErr); markErr != nil {
			return storage.ProviderGrantRecord{}, markErr
		}
		return storage.ProviderGrantRecord{}, emptyResultErr
	}
	tokenCiphertext, err := s.sealer.Seal(newTokenPlaintext)
	if err != nil {
		return storage.ProviderGrantRecord{}, fmt.Errorf("seal provider token: %w", err)
	}

	if err := s.providerGrantStore.UpdateProviderGrantToken(ctx, storage.UpdateProviderGrantTokenInput{
		OwnerUserID:      ownerUserID,
		ProviderGrantID:  providerGrantID,
		TokenCiphertext:  tokenCiphertext,
		RefreshedAt:      refreshedAt,
		ExpiresAt:        exchanged.ExpiresAt,
		Status:           string(providergrant.StatusActive),
		LastRefreshError: strings.TrimSpace(exchanged.LastRefreshError),
	}); err != nil {
		return storage.ProviderGrantRecord{}, fmt.Errorf("update provider grant token: %w", err)
	}
	updated, err := s.providerGrantStore.GetProviderGrant(ctx, providerGrantID)
	if err != nil {
		return storage.ProviderGrantRecord{}, fmt.Errorf("get provider grant: %w", err)
	}
	if strings.TrimSpace(updated.OwnerUserID) != ownerUserID {
		return storage.ProviderGrantRecord{}, storage.ErrNotFound
	}
	return updated, nil
}

func (s *Service) markProviderGrantRefreshFailed(ctx context.Context, record storage.ProviderGrantRecord, refreshedAt time.Time, refreshErr error) error {
	message := "provider token refresh failed"
	if refreshErr != nil && strings.TrimSpace(refreshErr.Error()) != "" {
		message = strings.TrimSpace(refreshErr.Error())
	}
	if err := s.providerGrantStore.UpdateProviderGrantToken(ctx, storage.UpdateProviderGrantTokenInput{
		OwnerUserID:      record.OwnerUserID,
		ProviderGrantID:  record.ID,
		TokenCiphertext:  record.TokenCiphertext,
		RefreshedAt:      refreshedAt,
		ExpiresAt:        record.ExpiresAt,
		Status:           string(providergrant.StatusRefreshFailed),
		LastRefreshError: message,
	}); err != nil {
		return fmt.Errorf("mark provider grant refresh failed: %w", err)
	}
	return nil
}

func (s *Service) resolveProviderGrantForInvocation(ctx context.Context, ownerUserID string, providerGrantID string, requestedProvider provider.Provider) (storage.ProviderGrantRecord, error) {
	if s.providerGrantStore == nil {
		return storage.ProviderGrantRecord{}, status.Error(codes.Internal, "provider grant store is not configured")
	}

	record, err := s.providerGrantStore.GetProviderGrant(ctx, providerGrantID)
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

	now := s.clock().UTC()
	grant := providergrant.ProviderGrant{
		OwnerUserID:      record.OwnerUserID,
		Provider:         providerFromString(record.Provider),
		RefreshSupported: record.RefreshSupported,
		Status:           providergrant.ParseStatus(record.Status),
		ExpiresAt:        record.ExpiresAt,
	}
	switch grant.Status {
	case providergrant.StatusActive:
		if grant.IsExpired(now) && !record.RefreshSupported {
			return storage.ProviderGrantRecord{}, status.Error(codes.FailedPrecondition, "provider grant is unavailable")
		}
		if grant.ShouldRefresh(now, providerGrantRefreshWindow) {
			refreshed, err := s.refreshProviderGrant(ctx, ownerUserID, providerGrantID)
			if err != nil {
				return storage.ProviderGrantRecord{}, status.Error(codes.FailedPrecondition, "provider grant refresh failed")
			}
			record = refreshed
		}
	case providergrant.StatusRefreshFailed, providergrant.StatusExpired:
		if !record.RefreshSupported {
			return storage.ProviderGrantRecord{}, status.Error(codes.FailedPrecondition, "provider grant is unavailable")
		}
		refreshed, err := s.refreshProviderGrant(ctx, ownerUserID, providerGrantID)
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

func providerGrantFilterFromRequest(in *aiv1.ListProviderGrantsRequest) (storage.ProviderGrantFilter, error) {
	filter := storage.ProviderGrantFilter{}
	switch in.GetProvider() {
	case aiv1.Provider_PROVIDER_UNSPECIFIED:
	case aiv1.Provider_PROVIDER_OPENAI:
		filter.Provider = "openai"
	default:
		return storage.ProviderGrantFilter{}, fmt.Errorf("provider filter is invalid")
	}

	switch in.GetStatus() {
	case aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_UNSPECIFIED:
	case aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_ACTIVE:
		filter.Status = "active"
	case aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_REVOKED:
		filter.Status = "revoked"
	case aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_EXPIRED:
		filter.Status = "expired"
	case aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_REFRESH_FAILED:
		filter.Status = "refresh_failed"
	default:
		return storage.ProviderGrantFilter{}, fmt.Errorf("status filter is invalid")
	}
	return filter, nil
}
