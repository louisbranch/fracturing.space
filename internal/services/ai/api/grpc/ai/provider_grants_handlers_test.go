package ai

import (
	"context"
	"errors"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"github.com/louisbranch/fracturing.space/internal/services/ai/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func TestStartProviderConnectRequiresUserID(t *testing.T) {
	svc := newProviderGrantHandlersWithStores(t, newFakeStore(), newFakeStore(), &fakeSealer{})
	_, err := svc.StartProviderConnect(context.Background(), &aiv1.StartProviderConnectRequest{
		Provider: aiv1.Provider_PROVIDER_OPENAI,
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestStartProviderConnectUsesS256CodeChallenge(t *testing.T) {
	store := newFakeStore()
	oauthAdapter := &fakeProviderOAuthAdapter{}

	idValues := []string{"session-1", "state-1"}
	codeVerifier := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~"

	svc := newProviderGrantHandlersWithOpts(t, store, store, &fakeSealer{}, providerGrantTestOpts{
		oauthAdapters: map[provider.Provider]provider.OAuthAdapter{
			provider.OpenAI: oauthAdapter,
		},
		idGenerator: func() (string, error) {
			if len(idValues) == 0 {
				return "", errors.New("unexpected id call")
			}
			value := idValues[0]
			idValues = idValues[1:]
			return value, nil
		},
		codeVerifierGenerator: func() (string, error) {
			return codeVerifier, nil
		},
	})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.StartProviderConnect(ctx, &aiv1.StartProviderConnectRequest{
		Provider:        aiv1.Provider_PROVIDER_OPENAI,
		RequestedScopes: []string{"responses.read"},
	})
	if err != nil {
		t.Fatalf("start provider connect: %v", err)
	}

	expectedChallenge := pkceCodeChallengeS256(codeVerifier)
	if got := oauthAdapter.lastAuthorizationInput.CodeChallenge; got != expectedChallenge {
		t.Fatalf("code challenge = %q, want %q", got, expectedChallenge)
	}
	if got := oauthAdapter.lastAuthorizationInput.CodeChallenge; got == codeVerifier {
		t.Fatalf("code challenge must not equal verifier: %q", got)
	}

	stored := store.ConnectSessions[resp.GetConnectSessionId()]
	if got := stored.CodeVerifierCiphertext; got != "enc:"+codeVerifier {
		t.Fatalf("stored code verifier ciphertext = %q, want %q", got, "enc:"+codeVerifier)
	}
}

func TestFinishProviderConnectCreatesProviderGrant(t *testing.T) {
	store := newFakeStore()

	idValues := []string{"session-1", "state-1", "grant-1"}
	svc := newProviderGrantHandlersWithOpts(t, store, store, &fakeSealer{}, providerGrantTestOpts{
		clock: func() time.Time { return time.Date(2026, 2, 15, 23, 30, 0, 0, time.UTC) },
		idGenerator: func() (string, error) {
			if len(idValues) == 0 {
				return "", errors.New("unexpected id call")
			}
			value := idValues[0]
			idValues = idValues[1:]
			return value, nil
		},
	})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	startResp, err := svc.StartProviderConnect(ctx, &aiv1.StartProviderConnectRequest{
		Provider:        aiv1.Provider_PROVIDER_OPENAI,
		RequestedScopes: []string{"responses.read"},
	})
	if err != nil {
		t.Fatalf("start provider connect: %v", err)
	}

	finishResp, err := svc.FinishProviderConnect(ctx, &aiv1.FinishProviderConnectRequest{
		ConnectSessionId:  startResp.GetConnectSessionId(),
		State:             startResp.GetState(),
		AuthorizationCode: "auth-code-1",
	})
	if err != nil {
		t.Fatalf("finish provider connect: %v", err)
	}
	if finishResp.GetProviderGrant().GetId() != "grant-1" {
		t.Fatalf("provider grant id = %q, want %q", finishResp.GetProviderGrant().GetId(), "grant-1")
	}
	stored := store.ProviderGrants["grant-1"]
	if stored.TokenCiphertext != "enc:token:auth-code-1" {
		t.Fatalf("stored token ciphertext = %q, want %q", stored.TokenCiphertext, "enc:token:auth-code-1")
	}
}

func TestFinishProviderConnectDoesNotSealRawAuthorizationCode(t *testing.T) {
	store := newFakeStore()

	idValues := []string{"session-1", "state-1", "grant-1"}
	svc := newProviderGrantHandlersWithOpts(t, store, store, &fakeSealer{}, providerGrantTestOpts{
		clock: func() time.Time { return time.Date(2026, 2, 15, 23, 30, 0, 0, time.UTC) },
		idGenerator: func() (string, error) {
			if len(idValues) == 0 {
				return "", errors.New("unexpected id call")
			}
			value := idValues[0]
			idValues = idValues[1:]
			return value, nil
		},
	})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	startResp, err := svc.StartProviderConnect(ctx, &aiv1.StartProviderConnectRequest{
		Provider:        aiv1.Provider_PROVIDER_OPENAI,
		RequestedScopes: []string{"responses.read"},
	})
	if err != nil {
		t.Fatalf("start provider connect: %v", err)
	}

	_, err = svc.FinishProviderConnect(ctx, &aiv1.FinishProviderConnectRequest{
		ConnectSessionId:  startResp.GetConnectSessionId(),
		State:             startResp.GetState(),
		AuthorizationCode: "auth-code-1",
	})
	if err != nil {
		t.Fatalf("finish provider connect: %v", err)
	}
	stored := store.ProviderGrants["grant-1"]
	if stored.TokenCiphertext == "enc:auth-code-1" {
		t.Fatalf("stored ciphertext should not seal raw authorization code: %q", stored.TokenCiphertext)
	}
}

func TestFinishProviderConnectKeepsSessionPendingOnExchangeFailure(t *testing.T) {
	store := newFakeStore()

	idValues := []string{"session-1", "state-1"}
	svc := newProviderGrantHandlersWithOpts(t, store, store, &fakeSealer{}, providerGrantTestOpts{
		oauthAdapters: map[provider.Provider]provider.OAuthAdapter{
			provider.OpenAI: &fakeProviderOAuthAdapter{
				exchangeErr: errors.New("exchange failed"),
			},
		},
		idGenerator: func() (string, error) {
			if len(idValues) == 0 {
				return "", errors.New("unexpected id call")
			}
			value := idValues[0]
			idValues = idValues[1:]
			return value, nil
		},
	})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	startResp, err := svc.StartProviderConnect(ctx, &aiv1.StartProviderConnectRequest{
		Provider:        aiv1.Provider_PROVIDER_OPENAI,
		RequestedScopes: []string{"responses.read"},
	})
	if err != nil {
		t.Fatalf("start provider connect: %v", err)
	}

	_, err = svc.FinishProviderConnect(ctx, &aiv1.FinishProviderConnectRequest{
		ConnectSessionId:  startResp.GetConnectSessionId(),
		State:             startResp.GetState(),
		AuthorizationCode: "auth-code-1",
	})
	assertStatusCode(t, err, codes.Internal)

	stored := store.ConnectSessions[startResp.GetConnectSessionId()]
	if stored.Status != "pending" {
		t.Fatalf("connect session status = %q, want %q", stored.Status, "pending")
	}
}

func TestListProviderGrantsReturnsOwnerRecords(t *testing.T) {
	store := newFakeStore()
	store.ProviderGrants["grant-1"] = providergrant.ProviderGrant{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        provider.OpenAI,
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: "enc:1",
		Status:          providergrant.StatusActive,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	store.ProviderGrants["grant-2"] = providergrant.ProviderGrant{
		ID:              "grant-2",
		OwnerUserID:     "user-2",
		Provider:        provider.OpenAI,
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: "enc:2",
		Status:          providergrant.StatusActive,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	svc := newProviderGrantHandlersWithStores(t, store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.ListProviderGrants(ctx, &aiv1.ListProviderGrantsRequest{PageSize: 10})
	if err != nil {
		t.Fatalf("list provider grants: %v", err)
	}
	if len(resp.GetProviderGrants()) != 1 {
		t.Fatalf("provider grants len = %d, want 1", len(resp.GetProviderGrants()))
	}
	if resp.GetProviderGrants()[0].GetId() != "grant-1" {
		t.Fatalf("provider grant id = %q, want %q", resp.GetProviderGrants()[0].GetId(), "grant-1")
	}
}

func TestListProviderGrantsFiltersByProvider(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 2, 0, 0, 0, time.UTC)
	store.ProviderGrants["grant-1"] = providergrant.ProviderGrant{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        provider.OpenAI,
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: "enc:1",
		Status:          providergrant.StatusActive,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	store.ProviderGrants["grant-2"] = providergrant.ProviderGrant{
		ID:              "grant-2",
		OwnerUserID:     "user-1",
		Provider:        provider.Provider("other"),
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: "enc:2",
		Status:          providergrant.StatusActive,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	store.ProviderGrants["grant-3"] = providergrant.ProviderGrant{
		ID:              "grant-3",
		OwnerUserID:     "user-2",
		Provider:        provider.OpenAI,
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: "enc:3",
		Status:          providergrant.StatusActive,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	svc := newProviderGrantHandlersWithStores(t, store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.ListProviderGrants(ctx, &aiv1.ListProviderGrantsRequest{
		PageSize: 10,
		Provider: aiv1.Provider_PROVIDER_OPENAI,
	})
	if err != nil {
		t.Fatalf("list provider grants: %v", err)
	}
	if len(resp.GetProviderGrants()) != 1 {
		t.Fatalf("provider grants len = %d, want 1", len(resp.GetProviderGrants()))
	}
	if got := resp.GetProviderGrants()[0].GetId(); got != "grant-1" {
		t.Fatalf("provider grant id = %q, want %q", got, "grant-1")
	}
}

func TestListProviderGrantsFiltersByStatus(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 2, 0, 0, 0, time.UTC)
	store.ProviderGrants["grant-1"] = providergrant.ProviderGrant{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        provider.OpenAI,
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: "enc:1",
		Status:          providergrant.StatusActive,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	store.ProviderGrants["grant-2"] = providergrant.ProviderGrant{
		ID:              "grant-2",
		OwnerUserID:     "user-1",
		Provider:        provider.OpenAI,
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: "enc:2",
		Status:          providergrant.StatusRevoked,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	store.ProviderGrants["grant-3"] = providergrant.ProviderGrant{
		ID:              "grant-3",
		OwnerUserID:     "user-2",
		Provider:        provider.OpenAI,
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: "enc:3",
		Status:          providergrant.StatusRevoked,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	svc := newProviderGrantHandlersWithStores(t, store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.ListProviderGrants(ctx, &aiv1.ListProviderGrantsRequest{
		PageSize: 10,
		Status:   aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_REVOKED,
	})
	if err != nil {
		t.Fatalf("list provider grants: %v", err)
	}
	if len(resp.GetProviderGrants()) != 1 {
		t.Fatalf("provider grants len = %d, want 1", len(resp.GetProviderGrants()))
	}
	if got := resp.GetProviderGrants()[0].GetId(); got != "grant-2" {
		t.Fatalf("provider grant id = %q, want %q", got, "grant-2")
	}
}

func TestRevokeProviderGrant(t *testing.T) {
	store := newFakeStore()
	store.ProviderGrants["grant-1"] = providergrant.ProviderGrant{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        provider.OpenAI,
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		Status:          providergrant.StatusActive,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	adapter := &fakeProviderOAuthAdapter{}
	svc := newProviderGrantHandlersWithOpts(t, store, store, &fakeSealer{}, providerGrantTestOpts{
		oauthAdapters: map[provider.Provider]provider.OAuthAdapter{
			provider.OpenAI: adapter,
		},
		clock: func() time.Time { return time.Date(2026, 2, 15, 23, 31, 0, 0, time.UTC) },
	})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))

	resp, err := svc.RevokeProviderGrant(ctx, &aiv1.RevokeProviderGrantRequest{ProviderGrantId: "grant-1"})
	if err != nil {
		t.Fatalf("revoke provider grant: %v", err)
	}
	if resp.GetProviderGrant().GetStatus() != aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_REVOKED {
		t.Fatalf("status = %v, want revoked", resp.GetProviderGrant().GetStatus())
	}
	if adapter.lastRevokedToken != "rt-1" {
		t.Fatalf("revoked token = %q, want %q", adapter.lastRevokedToken, "rt-1")
	}
}

func TestRevokeProviderGrantFailsWhenReferencedAgentIsBoundToActiveCampaigns(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.ProviderGrants["grant-1"] = providergrant.ProviderGrant{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        provider.OpenAI,
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		Status:          providergrant.StatusActive,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	store.Agents["agent-1"] = agent.Agent{
		ID:            "agent-1",
		OwnerUserID:   "user-1",
		Label:         "narrator",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: agent.ProviderGrantAuthReference("grant-1"),
		Status:        agent.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	svc := newProviderGrantHandlersWithOpts(t, store, store, &fakeSealer{}, providerGrantTestOpts{
		usageGuard: service.NewUsageGuard(store, &fakeCampaignAIAuthStateClient{
			usageByAgent: map[string]int32{"agent-1": 1},
		}),
	})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))

	_, err := svc.RevokeProviderGrant(ctx, &aiv1.RevokeProviderGrantRequest{ProviderGrantId: "grant-1"})
	assertStatusCode(t, err, codes.FailedPrecondition)

	if got := store.ProviderGrants["grant-1"].Status; got != "active" {
		t.Fatalf("provider grant status = %q, want active", got)
	}
}

func TestFinishProviderConnectAcrossServiceInstances(t *testing.T) {
	store := newFakeStore()
	sealer := &fakeSealer{}

	idValuesStart := []string{"session-1", "state-1"}
	svcStart := newProviderGrantHandlersWithOpts(t, store, store, sealer, providerGrantTestOpts{
		clock: func() time.Time { return time.Date(2026, 2, 15, 23, 40, 0, 0, time.UTC) },
		idGenerator: func() (string, error) {
			if len(idValuesStart) == 0 {
				return "", errors.New("unexpected id call")
			}
			value := idValuesStart[0]
			idValuesStart = idValuesStart[1:]
			return value, nil
		},
	})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	startResp, err := svcStart.StartProviderConnect(ctx, &aiv1.StartProviderConnectRequest{
		Provider:        aiv1.Provider_PROVIDER_OPENAI,
		RequestedScopes: []string{"responses.read"},
	})
	if err != nil {
		t.Fatalf("start provider connect: %v", err)
	}

	svcFinish := newProviderGrantHandlersWithOpts(t, store, store, sealer, providerGrantTestOpts{
		clock:       func() time.Time { return time.Date(2026, 2, 15, 23, 41, 0, 0, time.UTC) },
		idGenerator: func() (string, error) { return "grant-1", nil },
	})

	finishResp, err := svcFinish.FinishProviderConnect(ctx, &aiv1.FinishProviderConnectRequest{
		ConnectSessionId:  startResp.GetConnectSessionId(),
		State:             startResp.GetState(),
		AuthorizationCode: "auth-code-1",
	})
	if err != nil {
		t.Fatalf("finish provider connect: %v", err)
	}
	if finishResp.GetProviderGrant().GetId() != "grant-1" {
		t.Fatalf("provider grant id = %q, want %q", finishResp.GetProviderGrant().GetId(), "grant-1")
	}
}
