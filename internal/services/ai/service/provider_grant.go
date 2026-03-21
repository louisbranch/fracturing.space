package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"github.com/louisbranch/fracturing.space/internal/services/ai/secret"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// CodeVerifierGenerator returns a PKCE code verifier string.
type CodeVerifierGenerator = func() (string, error)

// ProviderGrantService handles provider-grant lifecycle operations including
// the OAuth connect handshake, listing, and revocation.
type ProviderGrantService struct {
	providerGrantStore    storage.ProviderGrantStore
	connectSessionStore   storage.ProviderConnectSessionStore
	sealer                secret.Sealer
	providerOAuthAdapters map[provider.Provider]provider.OAuthAdapter
	usageGuard            *UsageGuard
	clock                 Clock
	idGenerator           IDGenerator
	codeVerifierGenerator CodeVerifierGenerator
}

// ProviderGrantServiceConfig declares dependencies for the provider grant service.
type ProviderGrantServiceConfig struct {
	ProviderGrantStore    storage.ProviderGrantStore
	ConnectSessionStore   storage.ProviderConnectSessionStore
	Sealer                secret.Sealer
	ProviderOAuthAdapters map[provider.Provider]provider.OAuthAdapter
	UsageGuard            *UsageGuard
	Clock                 Clock
	IDGenerator           IDGenerator
	CodeVerifierGenerator CodeVerifierGenerator
}

// NewProviderGrantService builds a provider grant service from explicit deps.
func NewProviderGrantService(cfg ProviderGrantServiceConfig) (*ProviderGrantService, error) {
	if cfg.ProviderGrantStore == nil {
		return nil, fmt.Errorf("ai: NewProviderGrantService: provider grant store is required")
	}
	if cfg.ConnectSessionStore == nil {
		return nil, fmt.Errorf("ai: NewProviderGrantService: connect session store is required")
	}
	if cfg.Sealer == nil {
		return nil, fmt.Errorf("ai: NewProviderGrantService: sealer is required")
	}
	cvg := cfg.CodeVerifierGenerator
	if cvg == nil {
		cvg = generatePKCECodeVerifier
	}
	return &ProviderGrantService{
		providerGrantStore:    cfg.ProviderGrantStore,
		connectSessionStore:   cfg.ConnectSessionStore,
		sealer:                cfg.Sealer,
		providerOAuthAdapters: copyOAuthAdapters(cfg.ProviderOAuthAdapters),
		usageGuard:            cfg.UsageGuard,
		clock:                 withDefaultClock(cfg.Clock),
		idGenerator:           withDefaultIDGenerator(cfg.IDGenerator),
		codeVerifierGenerator: cvg,
	}, nil
}

// StartConnectInput is the domain input for starting a provider connect flow.
type StartConnectInput struct {
	OwnerUserID     string
	Provider        provider.Provider
	RequestedScopes []string
}

// StartConnectOutput is the domain result of starting a provider connect flow.
type StartConnectOutput struct {
	ConnectSessionID string
	State            string
	AuthorizationURL string
	ExpiresAt        time.Time
}

// StartConnect initiates an OAuth connect handshake: generates PKCE material,
// persists the connect session, and returns the authorization URL.
func (s *ProviderGrantService) StartConnect(ctx context.Context, input StartConnectInput) (StartConnectOutput, error) {
	sessionID, err := s.idGenerator()
	if err != nil {
		return StartConnectOutput{}, Wrapf(ErrKindInternal, err, "generate connect session id")
	}
	state, err := s.idGenerator()
	if err != nil {
		return StartConnectOutput{}, Wrapf(ErrKindInternal, err, "generate connect state")
	}
	codeVerifier, err := s.codeVerifierGenerator()
	if err != nil {
		return StartConnectOutput{}, Wrapf(ErrKindInternal, err, "generate code verifier")
	}
	if !isValidPKCECodeVerifier(codeVerifier) {
		return StartConnectOutput{}, Errorf(ErrKindInternal, "generate code verifier: value is invalid")
	}
	codeChallenge := pkceCodeChallengeS256(codeVerifier)
	codeVerifierCiphertext, err := s.sealer.Seal(codeVerifier)
	if err != nil {
		return StartConnectOutput{}, Wrapf(ErrKindInternal, err, "seal code verifier")
	}

	now := s.clock().UTC()
	expiresAt := now.Add(10 * time.Minute)
	record := storage.ProviderConnectSessionRecord{
		ID:                     sessionID,
		OwnerUserID:            input.OwnerUserID,
		Provider:               string(input.Provider),
		Status:                 "pending",
		RequestedScopes:        providergrant.NormalizeScopes(input.RequestedScopes),
		StateHash:              hashState(state),
		CodeVerifierCiphertext: codeVerifierCiphertext,
		CreatedAt:              now,
		UpdatedAt:              now,
		ExpiresAt:              expiresAt,
	}
	if err := s.connectSessionStore.PutProviderConnectSession(ctx, record); err != nil {
		return StartConnectOutput{}, Wrapf(ErrKindInternal, err, "put provider connect session")
	}

	adapter, ok := s.providerOAuthAdapters[input.Provider]
	if !ok || adapter == nil {
		return StartConnectOutput{}, Errorf(ErrKindFailedPrecondition, "provider oauth adapter is unavailable")
	}
	authorizationURL, err := adapter.BuildAuthorizationURL(provider.AuthorizationURLInput{
		State:           state,
		CodeChallenge:   codeChallenge,
		RequestedScopes: record.RequestedScopes,
	})
	if err != nil {
		return StartConnectOutput{}, Wrapf(ErrKindInternal, err, "build authorization url")
	}
	return StartConnectOutput{
		ConnectSessionID: sessionID,
		State:            state,
		AuthorizationURL: authorizationURL,
		ExpiresAt:        expiresAt,
	}, nil
}

// FinishConnectInput is the domain input for completing a provider connect flow.
type FinishConnectInput struct {
	OwnerUserID       string
	ConnectSessionID  string
	State             string
	AuthorizationCode string
}

// FinishConnect completes the OAuth handshake: validates the session, exchanges
// the authorization code for tokens, and creates the provider grant.
func (s *ProviderGrantService) FinishConnect(ctx context.Context, input FinishConnectInput) (providergrant.ProviderGrant, error) {
	if input.ConnectSessionID == "" {
		return providergrant.ProviderGrant{}, Errorf(ErrKindInvalidArgument, "connect_session_id is required")
	}
	if input.State == "" {
		return providergrant.ProviderGrant{}, Errorf(ErrKindInvalidArgument, "state is required")
	}
	if input.AuthorizationCode == "" {
		return providergrant.ProviderGrant{}, Errorf(ErrKindInvalidArgument, "authorization_code is required")
	}

	session, err := s.connectSessionStore.GetProviderConnectSession(ctx, input.ConnectSessionID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return providergrant.ProviderGrant{}, Errorf(ErrKindNotFound, "connect session not found")
		}
		return providergrant.ProviderGrant{}, Wrapf(ErrKindInternal, err, "get connect session")
	}
	if session.OwnerUserID != input.OwnerUserID {
		return providergrant.ProviderGrant{}, Errorf(ErrKindNotFound, "connect session not found")
	}
	if !strings.EqualFold(session.Status, "pending") {
		return providergrant.ProviderGrant{}, Errorf(ErrKindFailedPrecondition, "connect session is no longer pending")
	}
	if s.clock().UTC().After(session.ExpiresAt) {
		return providergrant.ProviderGrant{}, Errorf(ErrKindFailedPrecondition, "connect session expired")
	}
	if hashState(input.State) != session.StateHash {
		return providergrant.ProviderGrant{}, Errorf(ErrKindFailedPrecondition, "state mismatch")
	}

	providerID := normalizeProviderString(session.Provider)
	if providerID != provider.OpenAI {
		return providergrant.ProviderGrant{}, Errorf(ErrKindFailedPrecondition, "provider is unavailable")
	}
	adapter, ok := s.providerOAuthAdapters[providerID]
	if !ok || adapter == nil {
		return providergrant.ProviderGrant{}, Errorf(ErrKindFailedPrecondition, "provider oauth adapter is unavailable")
	}
	codeVerifier, err := s.sealer.Open(session.CodeVerifierCiphertext)
	if err != nil {
		return providergrant.ProviderGrant{}, Wrapf(ErrKindInternal, err, "open code verifier")
	}
	exchanged, err := adapter.ExchangeAuthorizationCode(ctx, provider.AuthorizationCodeInput{
		AuthorizationCode: input.AuthorizationCode,
		CodeVerifier:      codeVerifier,
	})
	if err != nil {
		return providergrant.ProviderGrant{}, Wrapf(ErrKindInternal, err, "exchange authorization code")
	}
	if exchanged.TokenPlaintext == "" {
		return providergrant.ProviderGrant{}, Errorf(ErrKindInternal, "provider returned empty token payload")
	}
	tokenCiphertext, err := s.sealer.Seal(exchanged.TokenPlaintext)
	if err != nil {
		return providergrant.ProviderGrant{}, Wrapf(ErrKindInternal, err, "seal provider token")
	}

	created, err := providergrant.Create(providergrant.CreateInput{
		OwnerUserID:      input.OwnerUserID,
		Provider:         providerID,
		GrantedScopes:    session.RequestedScopes,
		TokenCiphertext:  tokenCiphertext,
		RefreshSupported: exchanged.RefreshSupported,
		ExpiresAt:        exchanged.ExpiresAt,
	}, s.clock, s.idGenerator)
	if err != nil {
		return providergrant.ProviderGrant{}, Errorf(ErrKindInvalidArgument, "%s", err)
	}

	if err := s.providerGrantStore.PutProviderGrant(ctx, created); err != nil {
		return providergrant.ProviderGrant{}, Wrapf(ErrKindInternal, err, "put provider grant")
	}

	completedAt := s.clock().UTC()
	if err := s.connectSessionStore.CompleteProviderConnectSession(ctx, input.OwnerUserID, input.ConnectSessionID, completedAt); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return providergrant.ProviderGrant{}, Errorf(ErrKindNotFound, "connect session not found")
		}
		return providergrant.ProviderGrant{}, Wrapf(ErrKindInternal, err, "complete connect session")
	}
	return created, nil
}

// List returns a page of provider grants owned by the given user.
func (s *ProviderGrantService) List(ctx context.Context, ownerUserID string, pageSize int, pageToken string, filter providergrant.Filter) (providergrant.Page, error) {
	page, err := s.providerGrantStore.ListProviderGrantsByOwner(ctx, ownerUserID, pageSize, pageToken, filter)
	if err != nil {
		return providergrant.Page{}, Wrapf(ErrKindInternal, err, "list provider grants")
	}
	return page, nil
}

// Revoke revokes a provider grant owned by the given user. It also calls the
// upstream provider to revoke the token when an adapter is available.
func (s *ProviderGrantService) Revoke(ctx context.Context, ownerUserID, providerGrantID string) (providergrant.ProviderGrant, error) {
	if providerGrantID == "" {
		return providergrant.ProviderGrant{}, Errorf(ErrKindInvalidArgument, "provider_grant_id is required")
	}
	if s.usageGuard != nil {
		if err := s.usageGuard.EnsureProviderGrantNotBoundToActiveCampaigns(ctx, ownerUserID, providerGrantID); err != nil {
			return providergrant.ProviderGrant{}, err
		}
	}

	grant, err := s.providerGrantStore.GetProviderGrant(ctx, providerGrantID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return providergrant.ProviderGrant{}, Errorf(ErrKindNotFound, "provider grant not found")
		}
		return providergrant.ProviderGrant{}, Wrapf(ErrKindInternal, err, "get provider grant")
	}
	if grant.OwnerUserID != ownerUserID {
		return providergrant.ProviderGrant{}, Errorf(ErrKindNotFound, "provider grant not found")
	}

	if adapter, ok := s.providerOAuthAdapters[grant.Provider]; ok && adapter != nil {
		tokenPlaintext, err := s.sealer.Open(grant.TokenCiphertext)
		if err != nil {
			return providergrant.ProviderGrant{}, Wrapf(ErrKindInternal, err, "open provider token")
		}
		tokenForRevoke, err := providergrant.RevokeTokenFromPayload(tokenPlaintext)
		if err != nil {
			return providergrant.ProviderGrant{}, Wrapf(ErrKindInternal, err, "derive provider revoke token")
		}
		if err := adapter.RevokeToken(ctx, provider.RevokeTokenInput{Token: tokenForRevoke}); err != nil {
			return providergrant.ProviderGrant{}, Wrapf(ErrKindInternal, err, "revoke provider token")
		}
	}

	revoked, err := providergrant.Revoke(grant, s.clock)
	if err != nil {
		return providergrant.ProviderGrant{}, Errorf(ErrKindFailedPrecondition, "%s", err)
	}
	if err := s.providerGrantStore.PutProviderGrant(ctx, revoked); err != nil {
		return providergrant.ProviderGrant{}, Wrapf(ErrKindInternal, err, "put provider grant")
	}
	return revoked, nil
}

// --- PKCE and state helpers (business logic, moved from transport) ---

// generatePKCECodeVerifier returns an RFC 7636-compliant verifier string with
// cryptographic entropy suitable for S256 code challenge derivation.
func generatePKCECodeVerifier() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("read pkce entropy: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func pkceCodeChallengeS256(codeVerifier string) string {
	sum := sha256.Sum256([]byte(codeVerifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func isValidPKCECodeVerifier(value string) bool {
	if len(value) < 43 || len(value) > 128 {
		return false
	}
	for _, r := range value {
		switch {
		case r >= 'A' && r <= 'Z':
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-', r == '.', r == '_', r == '~':
		default:
			return false
		}
	}
	return true
}

func hashState(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// normalizeProviderString converts a raw provider string to a typed provider,
// returning empty on failure.
func normalizeProviderString(value string) provider.Provider {
	normalized, err := provider.Normalize(value)
	if err != nil {
		return ""
	}
	return normalized
}

// copyOAuthAdapters returns a defensive copy of the adapter map.
func copyOAuthAdapters(adapters map[provider.Provider]provider.OAuthAdapter) map[provider.Provider]provider.OAuthAdapter {
	if adapters == nil {
		return nil
	}
	copied := make(map[provider.Provider]provider.OAuthAdapter, len(adapters))
	for k, v := range adapters {
		copied[k] = v
	}
	return copied
}
