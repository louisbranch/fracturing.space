package service

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/credential"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provideroauth"
	"github.com/louisbranch/fracturing.space/internal/test/mock/aifakes"
)

func TestAuthReferencePolicyListProviderModelsSortsAndFilters(t *testing.T) {
	credentialStore := aifakes.NewCredentialStore()
	now := time.Date(2026, 3, 23, 20, 0, 0, 0, time.UTC)
	credentialStore.Credentials["cred-1"] = credential.Credential{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		Label:            "main",
		SecretCiphertext: "enc:sk-1",
		Status:           credential.StatusActive,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	modelAdapter := &fakeModelAdapter{
		listModelsResult: []provider.Model{
			{ID: "gpt-4.1-mini"},
			{ID: ""},
			{ID: "gpt-5-mini"},
			{ID: "gpt-4o"},
		},
	}

	policy := mustNewAuthReferencePolicy(t, authReferencePolicyDeps{
		credentialStore: credentialStore,
		modelAdapters: map[provider.Provider]provider.ModelAdapter{
			provider.OpenAI: modelAdapter,
		},
	})

	models, err := policy.ListProviderModels(context.Background(), "user-1", provider.OpenAI, agent.CredentialAuthReference("cred-1"))
	if err != nil {
		t.Fatalf("ListProviderModels: %v", err)
	}
	if modelAdapter.lastInput.AuthToken != "sk-1" {
		t.Fatalf("modelAdapter.lastInput.AuthToken = %q, want %q", modelAdapter.lastInput.AuthToken, "sk-1")
	}
	if len(models) != 3 {
		t.Fatalf("len(models) = %d, want %d", len(models), 3)
	}
	if models[0].ID != "gpt-4.1-mini" || models[1].ID != "gpt-4o" || models[2].ID != "gpt-5-mini" {
		t.Fatalf("model order = %q, %q, %q", models[0].ID, models[1].ID, models[2].ID)
	}
}

func TestAuthReferencePolicyValidateModelAvailableRejectsMissingModel(t *testing.T) {
	credentialStore := aifakes.NewCredentialStore()
	now := time.Date(2026, 3, 23, 20, 5, 0, 0, time.UTC)
	credentialStore.Credentials["cred-1"] = credential.Credential{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		Label:            "main",
		SecretCiphertext: "enc:sk-1",
		Status:           credential.StatusActive,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	policy := mustNewAuthReferencePolicy(t, authReferencePolicyDeps{
		credentialStore: credentialStore,
		modelAdapters: map[provider.Provider]provider.ModelAdapter{
			provider.OpenAI: &fakeModelAdapter{listModelsResult: []provider.Model{{ID: "gpt-5-mini"}}},
		},
	})

	err := policy.ValidateModelAvailable(context.Background(), "user-1", provider.OpenAI, agent.CredentialAuthReference("cred-1"), "gpt-4o")
	if got := ErrorKindOf(err); got != ErrKindFailedPrecondition {
		t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindFailedPrecondition)
	}
}

func TestAuthReferencePolicyAuthState(t *testing.T) {
	credentialStore := aifakes.NewCredentialStore()
	providerGrantStore := aifakes.NewProviderGrantStore()
	now := time.Date(2026, 3, 23, 20, 10, 0, 0, time.UTC)
	credentialStore.Credentials["cred-ready"] = credential.Credential{
		ID:               "cred-ready",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		Label:            "ready",
		SecretCiphertext: "enc:sk-1",
		Status:           credential.StatusActive,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	credentialStore.Credentials["cred-revoked"] = credential.Credential{
		ID:          "cred-revoked",
		OwnerUserID: "user-1",
		Provider:    provider.OpenAI,
		Label:       "revoked",
		Status:      credential.StatusRevoked,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	providerGrantStore.ProviderGrants["grant-revoked"] = providergrant.ProviderGrant{
		ID:              "grant-revoked",
		OwnerUserID:     "user-1",
		Provider:        provider.OpenAI,
		TokenCiphertext: `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		Status:          providergrant.StatusRevoked,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	policy := mustNewAuthReferencePolicy(t, authReferencePolicyDeps{
		credentialStore:    credentialStore,
		providerGrantStore: providerGrantStore,
	})

	if got := policy.AuthState(context.Background(), agent.Agent{
		OwnerUserID:   "user-1",
		Provider:      provider.OpenAI,
		AuthReference: agent.CredentialAuthReference("cred-ready"),
	}); got != AgentAuthStateReady {
		t.Fatalf("AuthState(ready) = %v, want %v", got, AgentAuthStateReady)
	}
	if got := policy.AuthState(context.Background(), agent.Agent{
		OwnerUserID:   "user-1",
		Provider:      provider.OpenAI,
		AuthReference: agent.CredentialAuthReference("cred-revoked"),
	}); got != AgentAuthStateRevoked {
		t.Fatalf("AuthState(revoked credential) = %v, want %v", got, AgentAuthStateRevoked)
	}
	if got := policy.AuthState(context.Background(), agent.Agent{
		OwnerUserID:   "user-1",
		Provider:      provider.OpenAI,
		AuthReference: agent.ProviderGrantAuthReference("grant-revoked"),
	}); got != AgentAuthStateRevoked {
		t.Fatalf("AuthState(revoked grant) = %v, want %v", got, AgentAuthStateRevoked)
	}
	if got := policy.AuthState(context.Background(), agent.Agent{
		OwnerUserID:   "user-1",
		Provider:      provider.OpenAI,
		AuthReference: agent.CredentialAuthReference("cred-missing"),
	}); got != AgentAuthStateUnavailable {
		t.Fatalf("AuthState(missing) = %v, want %v", got, AgentAuthStateUnavailable)
	}
}

type authReferencePolicyDeps struct {
	credentialStore    *aifakes.CredentialStore
	providerGrantStore *aifakes.ProviderGrantStore
	modelAdapters      map[provider.Provider]provider.ModelAdapter
	oauthAdapters      map[provider.Provider]provideroauth.Adapter
	sealer             *aifakes.Sealer
	clock              Clock
}

func mustNewAuthReferencePolicy(t *testing.T, deps authReferencePolicyDeps) *AuthReferencePolicy {
	t.Helper()

	sealer := deps.sealer
	if sealer == nil {
		sealer = &aifakes.Sealer{}
	}
	providerGrantRuntime := NewProviderGrantRuntime(ProviderGrantRuntimeConfig{
		ProviderGrantStore: deps.providerGrantStore,
		ProviderRegistry:   mustProviderRegistryForTests(t, deps.oauthAdapters, nil, deps.modelAdapters, nil),
		Sealer:             sealer,
		Clock:              deps.clock,
	})
	resolver := NewAuthMaterialResolver(AuthMaterialResolverConfig{
		CredentialStore:      deps.credentialStore,
		Sealer:               sealer,
		ProviderGrantRuntime: providerGrantRuntime,
	})
	policy, err := NewAuthReferencePolicy(AuthReferencePolicyConfig{
		CredentialStore:      deps.credentialStore,
		ProviderGrantStore:   deps.providerGrantStore,
		ProviderRegistry:     mustProviderRegistryForTests(t, deps.oauthAdapters, nil, deps.modelAdapters, nil),
		AuthMaterialResolver: resolver,
	})
	if err != nil {
		t.Fatalf("NewAuthReferencePolicy: %v", err)
	}
	return policy
}
