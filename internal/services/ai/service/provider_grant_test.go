package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providerconnect"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provideroauth"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"github.com/louisbranch/fracturing.space/internal/test/mock/aifakes"
)

type noRevokeOAuthAdapter struct{}

func (noRevokeOAuthAdapter) BuildAuthorizationURL(provideroauth.AuthorizationURLInput) (string, error) {
	return "", errors.New("unexpected BuildAuthorizationURL call")
}

func (noRevokeOAuthAdapter) ExchangeAuthorizationCode(context.Context, provideroauth.AuthorizationCodeInput) (provideroauth.TokenExchangeResult, error) {
	return provideroauth.TokenExchangeResult{}, errors.New("unexpected ExchangeAuthorizationCode call")
}

func (noRevokeOAuthAdapter) RefreshToken(context.Context, provideroauth.RefreshTokenInput) (provideroauth.TokenExchangeResult, error) {
	return provideroauth.TokenExchangeResult{}, errors.New("unexpected RefreshToken call")
}

func TestProviderGrantServiceStartConnectUsesS256CodeChallenge(t *testing.T) {
	connectSessionStore := aifakes.NewProviderConnectSessionStore()
	oauthAdapter := &fakeOAuthAdapter{}

	idValues := []string{"session-1", "state-1"}
	codeVerifier := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~"
	svc := mustNewProviderGrantService(t, ProviderGrantServiceConfig{
		ProviderGrantStore:  aifakes.NewProviderGrantStore(),
		ConnectSessionStore: connectSessionStore,
		ConnectFinisher:     newProviderGrantTestConnectFinisher(aifakes.NewProviderGrantStore(), connectSessionStore),
		Sealer:              &aifakes.Sealer{},
		ProviderRegistry: mustProviderRegistryForTests(t, map[provider.Provider]provideroauth.Adapter{
			provider.OpenAI: oauthAdapter,
		}, nil, nil, nil),
		IDGenerator: func() (string, error) {
			if len(idValues) == 0 {
				return "", errors.New("unexpected id call")
			}
			value := idValues[0]
			idValues = idValues[1:]
			return value, nil
		},
		CodeVerifierGenerator: func() (string, error) { return codeVerifier, nil },
	})

	record, err := svc.StartConnect(context.Background(), StartConnectInput{
		OwnerUserID:     "user-1",
		Provider:        provider.OpenAI,
		RequestedScopes: []string{"responses.read"},
	})
	if err != nil {
		t.Fatalf("StartConnect: %v", err)
	}
	if record.ConnectSessionID != "session-1" {
		t.Fatalf("record.ConnectSessionID = %q, want %q", record.ConnectSessionID, "session-1")
	}
	if record.State != "state-1" {
		t.Fatalf("record.State = %q, want %q", record.State, "state-1")
	}

	expectedChallenge := pkceCodeChallengeS256(codeVerifier)
	if got := oauthAdapter.lastAuthorizationInput.CodeChallenge; got != expectedChallenge {
		t.Fatalf("oauthAdapter.lastAuthorizationInput.CodeChallenge = %q, want %q", got, expectedChallenge)
	}
	if got := oauthAdapter.lastAuthorizationInput.CodeChallenge; got == codeVerifier {
		t.Fatalf("oauthAdapter.lastAuthorizationInput.CodeChallenge = %q, must not equal the verifier", got)
	}

	stored := connectSessionStore.ConnectSessions["session-1"]
	if stored.CodeVerifierCiphertext != "enc:"+codeVerifier {
		t.Fatalf("stored.CodeVerifierCiphertext = %q, want %q", stored.CodeVerifierCiphertext, "enc:"+codeVerifier)
	}
}

func TestProviderGrantServiceFinishConnectCreatesProviderGrant(t *testing.T) {
	providerGrantStore := aifakes.NewProviderGrantStore()
	connectSessionStore := aifakes.NewProviderConnectSessionStore()
	now := time.Date(2026, 3, 23, 16, 40, 0, 0, time.UTC)

	idValues := []string{"session-1", "state-1", "grant-1"}
	svc := mustNewProviderGrantService(t, ProviderGrantServiceConfig{
		ProviderGrantStore:  providerGrantStore,
		ConnectSessionStore: connectSessionStore,
		ConnectFinisher:     newProviderGrantTestConnectFinisher(providerGrantStore, connectSessionStore),
		Sealer:              &aifakes.Sealer{},
		ProviderRegistry: mustProviderRegistryForTests(t, map[provider.Provider]provideroauth.Adapter{
			provider.OpenAI: &fakeOAuthAdapter{
				exchangeResult: provideroauth.TokenExchangeResult{
					TokenPayload: provideroauth.TokenPayload{
						AccessToken:  "token:auth-code-1",
						RefreshToken: "refresh:auth-code-1",
					},
				},
			},
		}, nil, nil, nil),
		Clock: func() time.Time { return now },
		IDGenerator: func() (string, error) {
			if len(idValues) == 0 {
				return "", errors.New("unexpected id call")
			}
			value := idValues[0]
			idValues = idValues[1:]
			return value, nil
		},
		CodeVerifierGenerator: func() (string, error) {
			return "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~", nil
		},
	})

	started, err := svc.StartConnect(context.Background(), StartConnectInput{
		OwnerUserID:     "user-1",
		Provider:        provider.OpenAI,
		RequestedScopes: []string{"responses.read"},
	})
	if err != nil {
		t.Fatalf("StartConnect: %v", err)
	}

	record, err := svc.FinishConnect(context.Background(), FinishConnectInput{
		OwnerUserID:       "user-1",
		ConnectSessionID:  started.ConnectSessionID,
		State:             started.State,
		AuthorizationCode: "auth-code-1",
	})
	if err != nil {
		t.Fatalf("FinishConnect: %v", err)
	}
	if record.ID != "grant-1" {
		t.Fatalf("record.ID = %q, want %q", record.ID, "grant-1")
	}
	if record.TokenCiphertext != `enc:{"access_token":"token:auth-code-1","refresh_token":"refresh:auth-code-1"}` {
		t.Fatalf("record.TokenCiphertext = %q", record.TokenCiphertext)
	}
	if record.TokenCiphertext == "enc:auth-code-1" {
		t.Fatalf("record.TokenCiphertext = %q, must not seal the raw authorization code", record.TokenCiphertext)
	}

	stored := providerGrantStore.ProviderGrants["grant-1"]
	if stored.TokenCiphertext != `enc:{"access_token":"token:auth-code-1","refresh_token":"refresh:auth-code-1"}` {
		t.Fatalf("stored.TokenCiphertext = %q", stored.TokenCiphertext)
	}

	session := connectSessionStore.ConnectSessions[started.ConnectSessionID]
	if session.Status != "completed" {
		t.Fatalf("session.Status = %q, want %q", session.Status, "completed")
	}
	if session.CompletedAt == nil || !session.CompletedAt.Equal(now) {
		t.Fatalf("session.CompletedAt = %v, want %v", session.CompletedAt, now)
	}
}

func TestProviderGrantServiceFinishConnectKeepsSessionPendingOnExchangeFailure(t *testing.T) {
	connectSessionStore := aifakes.NewProviderConnectSessionStore()

	idValues := []string{"session-1", "state-1"}
	svc := mustNewProviderGrantService(t, ProviderGrantServiceConfig{
		ProviderGrantStore:  aifakes.NewProviderGrantStore(),
		ConnectSessionStore: connectSessionStore,
		ConnectFinisher:     newProviderGrantTestConnectFinisher(aifakes.NewProviderGrantStore(), connectSessionStore),
		Sealer:              &aifakes.Sealer{},
		ProviderRegistry: mustProviderRegistryForTests(t, map[provider.Provider]provideroauth.Adapter{
			provider.OpenAI: &fakeOAuthAdapter{exchangeErr: errors.New("exchange failed")},
		}, nil, nil, nil),
		IDGenerator: func() (string, error) {
			if len(idValues) == 0 {
				return "", errors.New("unexpected id call")
			}
			value := idValues[0]
			idValues = idValues[1:]
			return value, nil
		},
		CodeVerifierGenerator: func() (string, error) {
			return "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~", nil
		},
	})

	started, err := svc.StartConnect(context.Background(), StartConnectInput{
		OwnerUserID:     "user-1",
		Provider:        provider.OpenAI,
		RequestedScopes: []string{"responses.read"},
	})
	if err != nil {
		t.Fatalf("StartConnect: %v", err)
	}

	_, err = svc.FinishConnect(context.Background(), FinishConnectInput{
		OwnerUserID:       "user-1",
		ConnectSessionID:  started.ConnectSessionID,
		State:             started.State,
		AuthorizationCode: "auth-code-1",
	})
	if got := ErrorKindOf(err); got != ErrKindInternal {
		t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindInternal)
	}

	session := connectSessionStore.ConnectSessions[started.ConnectSessionID]
	if session.Status != "pending" {
		t.Fatalf("session.Status = %q, want %q", session.Status, "pending")
	}
}

func TestProviderGrantServiceFinishConnectUsesAdapterAvailabilityForProvider(t *testing.T) {
	connectSessionStore := aifakes.NewProviderConnectSessionStore()
	now := time.Date(2026, 3, 23, 16, 42, 0, 0, time.UTC)
	connectSessionStore.ConnectSessions["session-1"] = providerconnect.Session{
		ID:                     "session-1",
		OwnerUserID:            "user-1",
		Provider:               provider.Provider("other"),
		Status:                 providerconnect.StatusPending,
		RequestedScopes:        []string{"responses.read"},
		StateHash:              hashState("state-1"),
		CodeVerifierCiphertext: "enc:verifier-1",
		CreatedAt:              now.Add(-time.Minute),
		UpdatedAt:              now.Add(-time.Minute),
		ExpiresAt:              now.Add(time.Minute),
	}

	svc := mustNewProviderGrantService(t, ProviderGrantServiceConfig{
		ProviderGrantStore:  aifakes.NewProviderGrantStore(),
		ConnectSessionStore: connectSessionStore,
		ConnectFinisher:     newProviderGrantTestConnectFinisher(aifakes.NewProviderGrantStore(), connectSessionStore),
		Sealer:              &aifakes.Sealer{},
		ProviderRegistry:    mustProviderRegistryForTests(t, map[provider.Provider]provideroauth.Adapter{}, nil, nil, nil),
		Clock:               func() time.Time { return now },
	})

	_, err := svc.FinishConnect(context.Background(), FinishConnectInput{
		OwnerUserID:       "user-1",
		ConnectSessionID:  "session-1",
		State:             "state-1",
		AuthorizationCode: "auth-code-1",
	})
	if got := ErrorKindOf(err); got != ErrKindFailedPrecondition {
		t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindFailedPrecondition)
	}
	if err == nil || err.Error() != "provider oauth adapter is unavailable" {
		t.Fatalf("err = %v, want provider oauth adapter is unavailable", err)
	}

	session := connectSessionStore.ConnectSessions["session-1"]
	if session.Status != providerconnect.StatusPending {
		t.Fatalf("session.Status = %q, want %q", session.Status, providerconnect.StatusPending)
	}
}

func TestProviderGrantServiceStartConnectRejectsUnavailableProvider(t *testing.T) {
	svc := mustNewProviderGrantService(t, ProviderGrantServiceConfig{
		ProviderGrantStore:  aifakes.NewProviderGrantStore(),
		ConnectSessionStore: aifakes.NewProviderConnectSessionStore(),
		ConnectFinisher:     newProviderGrantTestConnectFinisher(aifakes.NewProviderGrantStore(), aifakes.NewProviderConnectSessionStore()),
		Sealer:              &aifakes.Sealer{},
		ProviderRegistry: mustProviderRegistryForTests(t, map[provider.Provider]provideroauth.Adapter{
			provider.OpenAI: &fakeOAuthAdapter{},
		}, nil, nil, nil),
		IDGenerator: func() (string, error) { return "session-1", nil },
		CodeVerifierGenerator: func() (string, error) {
			return "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~", nil
		},
	})

	_, err := svc.StartConnect(context.Background(), StartConnectInput{
		OwnerUserID: "user-1",
		Provider:    provider.Anthropic,
	})
	if got := ErrorKindOf(err); got != ErrKindFailedPrecondition {
		t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindFailedPrecondition)
	}
}

func TestProviderGrantServiceListFiltersByOwnerProviderAndStatus(t *testing.T) {
	providerGrantStore := aifakes.NewProviderGrantStore()
	now := time.Date(2026, 3, 23, 16, 45, 0, 0, time.UTC)
	providerGrantStore.ProviderGrants["grant-1"] = providergrant.ProviderGrant{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        provider.OpenAI,
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: "enc:1",
		Status:          providergrant.StatusActive,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	providerGrantStore.ProviderGrants["grant-2"] = providergrant.ProviderGrant{
		ID:              "grant-2",
		OwnerUserID:     "user-1",
		Provider:        provider.Provider("other"),
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: "enc:2",
		Status:          providergrant.StatusActive,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	providerGrantStore.ProviderGrants["grant-3"] = providergrant.ProviderGrant{
		ID:              "grant-3",
		OwnerUserID:     "user-1",
		Provider:        provider.OpenAI,
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: "enc:3",
		Status:          providergrant.StatusRevoked,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	providerGrantStore.ProviderGrants["grant-4"] = providergrant.ProviderGrant{
		ID:              "grant-4",
		OwnerUserID:     "user-2",
		Provider:        provider.OpenAI,
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: "enc:4",
		Status:          providergrant.StatusActive,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	svc := mustNewProviderGrantService(t, ProviderGrantServiceConfig{
		ProviderGrantStore:  providerGrantStore,
		ConnectSessionStore: aifakes.NewProviderConnectSessionStore(),
		ConnectFinisher:     newProviderGrantTestConnectFinisher(providerGrantStore, aifakes.NewProviderConnectSessionStore()),
		Sealer:              &aifakes.Sealer{},
		ProviderRegistry:    mustProviderRegistryForTests(t, map[provider.Provider]provideroauth.Adapter{provider.OpenAI: nil}, nil, nil, nil),
	})

	page, err := svc.List(context.Background(), "user-1", 10, "", providergrant.Filter{
		Provider: provider.OpenAI,
		Status:   providergrant.StatusActive,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.ProviderGrants) != 1 {
		t.Fatalf("len(page.ProviderGrants) = %d, want 1", len(page.ProviderGrants))
	}
	if page.ProviderGrants[0].ID != "grant-1" {
		t.Fatalf("page.ProviderGrants[0].ID = %q, want %q", page.ProviderGrants[0].ID, "grant-1")
	}
}

func TestProviderGrantServiceRevokeRevokesOwnedGrant(t *testing.T) {
	providerGrantStore := aifakes.NewProviderGrantStore()
	now := time.Date(2026, 3, 23, 16, 50, 0, 0, time.UTC)
	providerGrantStore.ProviderGrants["grant-1"] = providergrant.ProviderGrant{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        provider.OpenAI,
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		Status:          providergrant.StatusActive,
		CreatedAt:       now.Add(-time.Hour),
		UpdatedAt:       now.Add(-time.Hour),
	}

	oauthAdapter := &fakeOAuthAdapter{}
	svc := mustNewProviderGrantService(t, ProviderGrantServiceConfig{
		ProviderGrantStore:  providerGrantStore,
		ConnectSessionStore: aifakes.NewProviderConnectSessionStore(),
		ConnectFinisher:     newProviderGrantTestConnectFinisher(providerGrantStore, aifakes.NewProviderConnectSessionStore()),
		Sealer:              &aifakes.Sealer{},
		ProviderRegistry: mustProviderRegistryForTests(t, map[provider.Provider]provideroauth.Adapter{
			provider.OpenAI: oauthAdapter,
		}, nil, nil, nil),
		Clock: func() time.Time { return now },
	})

	record, err := svc.Revoke(context.Background(), "user-1", "grant-1")
	if err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if record.Status != providergrant.StatusRevoked {
		t.Fatalf("record.Status = %q, want %q", record.Status, providergrant.StatusRevoked)
	}
	if oauthAdapter.lastRevokedToken != "rt-1" {
		t.Fatalf("oauthAdapter.lastRevokedToken = %q, want %q", oauthAdapter.lastRevokedToken, "rt-1")
	}
	if stored := providerGrantStore.ProviderGrants["grant-1"]; stored.Status != providergrant.StatusRevoked {
		t.Fatalf("stored.Status = %q, want %q", stored.Status, providergrant.StatusRevoked)
	}
}

func TestProviderGrantServiceRevokeRejectsActiveCampaignBinding(t *testing.T) {
	providerGrantStore := aifakes.NewProviderGrantStore()
	agentStore := aifakes.NewAgentStore()
	now := time.Date(2026, 3, 23, 16, 55, 0, 0, time.UTC)
	providerGrantStore.ProviderGrants["grant-1"] = providergrant.ProviderGrant{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        provider.OpenAI,
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		Status:          providergrant.StatusActive,
		CreatedAt:       now.Add(-time.Hour),
		UpdatedAt:       now.Add(-time.Hour),
	}
	agentStore.Agents["agent-1"] = agent.Agent{
		ID:            "agent-1",
		OwnerUserID:   "user-1",
		Label:         "narrator",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: agent.ProviderGrantAuthReference("grant-1"),
		Status:        agent.StatusActive,
		CreatedAt:     now.Add(-time.Hour),
		UpdatedAt:     now.Add(-time.Hour),
	}

	usageReader := NewAgentBindingUsageReader(&fakeCampaignUsageReader{
		usageByAgent: map[string]int32{"agent-1": 1},
	})
	svc := mustNewProviderGrantService(t, ProviderGrantServiceConfig{
		ProviderGrantStore:  providerGrantStore,
		ConnectSessionStore: aifakes.NewProviderConnectSessionStore(),
		ConnectFinisher:     newProviderGrantTestConnectFinisher(providerGrantStore, aifakes.NewProviderConnectSessionStore()),
		Sealer:              &aifakes.Sealer{},
		ProviderRegistry:    mustProviderRegistryForTests(t, map[provider.Provider]provideroauth.Adapter{provider.OpenAI: nil}, nil, nil, nil),
		UsagePolicy: NewUsagePolicy(UsagePolicyConfig{
			AgentBindingUsageReader:  usageReader,
			AuthReferenceUsageReader: NewAuthReferenceUsageReader(agentStore, usageReader),
		}),
		Clock: func() time.Time { return now },
	})

	_, err := svc.Revoke(context.Background(), "user-1", "grant-1")
	if got := ErrorKindOf(err); got != ErrKindFailedPrecondition {
		t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindFailedPrecondition)
	}
	if stored := providerGrantStore.ProviderGrants["grant-1"]; stored.Status != providergrant.StatusActive {
		t.Fatalf("stored.Status = %q, want %q", stored.Status, providergrant.StatusActive)
	}
}

func TestProviderGrantServiceRevokeSucceedsWithoutUpstreamTokenRevoker(t *testing.T) {
	providerGrantStore := aifakes.NewProviderGrantStore()
	now := time.Date(2026, 3, 23, 16, 52, 0, 0, time.UTC)
	providerGrantStore.ProviderGrants["grant-1"] = providergrant.ProviderGrant{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        provider.OpenAI,
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		Status:          providergrant.StatusActive,
		CreatedAt:       now.Add(-time.Hour),
		UpdatedAt:       now.Add(-time.Hour),
	}

	svc := mustNewProviderGrantService(t, ProviderGrantServiceConfig{
		ProviderGrantStore:  providerGrantStore,
		ConnectSessionStore: aifakes.NewProviderConnectSessionStore(),
		ConnectFinisher:     newProviderGrantTestConnectFinisher(providerGrantStore, aifakes.NewProviderConnectSessionStore()),
		Sealer:              &aifakes.Sealer{},
		ProviderRegistry: mustProviderRegistryForTests(t, map[provider.Provider]provideroauth.Adapter{
			provider.OpenAI: noRevokeOAuthAdapter{},
		}, nil, nil, nil),
		Clock: func() time.Time { return now },
	})

	record, err := svc.Revoke(context.Background(), "user-1", "grant-1")
	if err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if record.Status != providergrant.StatusRevoked {
		t.Fatalf("record.Status = %q, want %q", record.Status, providergrant.StatusRevoked)
	}
	if stored := providerGrantStore.ProviderGrants["grant-1"]; stored.Status != providergrant.StatusRevoked {
		t.Fatalf("stored.Status = %q, want %q", stored.Status, providergrant.StatusRevoked)
	}
}

func TestProviderGrantServiceFinishConnectAcrossServiceInstances(t *testing.T) {
	providerGrantStore := aifakes.NewProviderGrantStore()
	connectSessionStore := aifakes.NewProviderConnectSessionStore()
	sealer := &aifakes.Sealer{}

	idValuesStart := []string{"session-1", "state-1"}
	svcStart := mustNewProviderGrantService(t, ProviderGrantServiceConfig{
		ProviderGrantStore:  providerGrantStore,
		ConnectSessionStore: connectSessionStore,
		ConnectFinisher:     newProviderGrantTestConnectFinisher(providerGrantStore, connectSessionStore),
		Sealer:              sealer,
		ProviderRegistry: mustProviderRegistryForTests(t, map[provider.Provider]provideroauth.Adapter{
			provider.OpenAI: &fakeOAuthAdapter{},
		}, nil, nil, nil),
		Clock: func() time.Time { return time.Date(2026, 3, 23, 17, 0, 0, 0, time.UTC) },
		IDGenerator: func() (string, error) {
			if len(idValuesStart) == 0 {
				return "", errors.New("unexpected id call")
			}
			value := idValuesStart[0]
			idValuesStart = idValuesStart[1:]
			return value, nil
		},
		CodeVerifierGenerator: func() (string, error) {
			return "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~", nil
		},
	})

	started, err := svcStart.StartConnect(context.Background(), StartConnectInput{
		OwnerUserID:     "user-1",
		Provider:        provider.OpenAI,
		RequestedScopes: []string{"responses.read"},
	})
	if err != nil {
		t.Fatalf("svcStart.StartConnect: %v", err)
	}

	svcFinish := mustNewProviderGrantService(t, ProviderGrantServiceConfig{
		ProviderGrantStore:  providerGrantStore,
		ConnectSessionStore: connectSessionStore,
		ConnectFinisher:     newProviderGrantTestConnectFinisher(providerGrantStore, connectSessionStore),
		Sealer:              sealer,
		ProviderRegistry: mustProviderRegistryForTests(t, map[provider.Provider]provideroauth.Adapter{
			provider.OpenAI: &fakeOAuthAdapter{},
		}, nil, nil, nil),
		Clock:       func() time.Time { return time.Date(2026, 3, 23, 17, 1, 0, 0, time.UTC) },
		IDGenerator: func() (string, error) { return "grant-1", nil },
	})

	record, err := svcFinish.FinishConnect(context.Background(), FinishConnectInput{
		OwnerUserID:       "user-1",
		ConnectSessionID:  started.ConnectSessionID,
		State:             started.State,
		AuthorizationCode: "auth-code-1",
	})
	if err != nil {
		t.Fatalf("svcFinish.FinishConnect: %v", err)
	}
	if record.ID != "grant-1" {
		t.Fatalf("record.ID = %q, want %q", record.ID, "grant-1")
	}
}

func mustNewProviderGrantService(t *testing.T, cfg ProviderGrantServiceConfig) *ProviderGrantService {
	t.Helper()

	svc, err := NewProviderGrantService(cfg)
	if err != nil {
		t.Fatalf("NewProviderGrantService: %v", err)
	}
	return svc
}

type providerGrantTestConnectFinisher struct {
	providerGrantStore  storage.ProviderGrantStore
	connectSessionStore providerconnect.Store
}

func newProviderGrantTestConnectFinisher(providerGrantStore storage.ProviderGrantStore, connectSessionStore providerconnect.Store) ProviderConnectFinisher {
	return providerGrantTestConnectFinisher{
		providerGrantStore:  providerGrantStore,
		connectSessionStore: connectSessionStore,
	}
}

func (f providerGrantTestConnectFinisher) FinishProviderConnect(ctx context.Context, grant providergrant.ProviderGrant, completedSession providerconnect.Session) error {
	session, err := f.connectSessionStore.GetProviderConnectSession(ctx, completedSession.ID)
	if err != nil {
		return err
	}
	if session.OwnerUserID != completedSession.OwnerUserID || session.Status != providerconnect.StatusPending || completedSession.Status != providerconnect.StatusCompleted {
		return storage.ErrNotFound
	}
	if err := f.providerGrantStore.PutProviderGrant(ctx, grant); err != nil {
		return err
	}
	return f.connectSessionStore.CompleteProviderConnectSession(ctx, completedSession)
}
