package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/accessrequest"
	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/auditevent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/credential"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provideroauth"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"github.com/louisbranch/fracturing.space/internal/test/mock/aifakes"
)

type fakeInvocationAdapter struct {
	invokeErr    error
	invokeResult provider.InvokeResult
	lastInput    provider.InvokeInput
}

func (f *fakeInvocationAdapter) Invoke(_ context.Context, input provider.InvokeInput) (provider.InvokeResult, error) {
	f.lastInput = input
	if f.invokeErr != nil {
		return provider.InvokeResult{}, f.invokeErr
	}
	return f.invokeResult, nil
}

type trackingAccessRequestStore struct {
	*aifakes.AccessRequestStore
	listByRequesterCalls int
	targetedLookupCalls  int
}

func newTrackingAccessRequestStore() *trackingAccessRequestStore {
	return &trackingAccessRequestStore{AccessRequestStore: aifakes.NewAccessRequestStore()}
}

func (s *trackingAccessRequestStore) ListAccessRequestsByRequester(ctx context.Context, requesterUserID string, pageSize int, pageToken string) (accessrequest.Page, error) {
	s.listByRequesterCalls++
	return s.AccessRequestStore.ListAccessRequestsByRequester(ctx, requesterUserID, pageSize, pageToken)
}

func (s *trackingAccessRequestStore) GetApprovedInvokeAccessByRequesterForAgent(ctx context.Context, requesterUserID string, ownerUserID string, agentID string) (accessrequest.AccessRequest, error) {
	s.targetedLookupCalls++
	return s.AccessRequestStore.GetApprovedInvokeAccessByRequesterForAgent(ctx, requesterUserID, ownerUserID, agentID)
}

func TestInvocationServiceInvokeAgentOwnedCredentialSuccess(t *testing.T) {
	agentStore := aifakes.NewAgentStore()
	credentialStore := aifakes.NewCredentialStore()
	now := time.Date(2026, 3, 23, 18, 0, 0, 0, time.UTC)
	agentStore.Agents["agent-1"] = agent.Agent{
		ID:            "agent-1",
		OwnerUserID:   "user-1",
		Label:         "narrator",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		Instructions:  "Keep the scene moving.",
		AuthReference: agent.CredentialAuthReference("cred-1"),
		Status:        agent.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	credentialStore.Credentials["cred-1"] = credential.Credential{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		Label:            "Main",
		SecretCiphertext: "enc:sk-1",
		Status:           credential.StatusActive,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	invokeAdapter := &fakeInvocationAdapter{
		invokeResult: provider.InvokeResult{
			OutputText: "Hello from AI",
			Usage:      provider.Usage{InputTokens: 12, OutputTokens: 7, ReasoningTokens: 3, TotalTokens: 19},
		},
	}
	svc := mustNewInvocationService(t, invocationServiceDeps{
		agentStore:      agentStore,
		credentialStore: credentialStore,
		invokeAdapters: map[provider.Provider]provider.InvocationAdapter{
			provider.OpenAI: invokeAdapter,
		},
		clock: func() time.Time { return now },
	})

	result, err := svc.InvokeAgent(context.Background(), InvokeAgentInput{
		CallerUserID:    "user-1",
		AgentID:         "agent-1",
		Input:           "Say hello",
		ReasoningEffort: "low",
	})
	if err != nil {
		t.Fatalf("InvokeAgent: %v", err)
	}
	if result.OutputText != "Hello from AI" {
		t.Fatalf("result.OutputText = %q, want %q", result.OutputText, "Hello from AI")
	}
	if result.Provider != provider.OpenAI {
		t.Fatalf("result.Provider = %q, want %q", result.Provider, provider.OpenAI)
	}
	if result.Model != "gpt-4o-mini" {
		t.Fatalf("result.Model = %q, want %q", result.Model, "gpt-4o-mini")
	}
	if result.Usage.TotalTokens != 19 {
		t.Fatalf("result.Usage.TotalTokens = %d, want %d", result.Usage.TotalTokens, 19)
	}
	if invokeAdapter.lastInput.AuthToken != "sk-1" {
		t.Fatalf("invokeAdapter.lastInput.AuthToken = %q, want %q", invokeAdapter.lastInput.AuthToken, "sk-1")
	}
	if invokeAdapter.lastInput.Model != "gpt-4o-mini" {
		t.Fatalf("invokeAdapter.lastInput.Model = %q, want %q", invokeAdapter.lastInput.Model, "gpt-4o-mini")
	}
	if invokeAdapter.lastInput.Input != "Say hello" {
		t.Fatalf("invokeAdapter.lastInput.Input = %q, want %q", invokeAdapter.lastInput.Input, "Say hello")
	}
	if invokeAdapter.lastInput.Instructions != "Keep the scene moving." {
		t.Fatalf("invokeAdapter.lastInput.Instructions = %q, want %q", invokeAdapter.lastInput.Instructions, "Keep the scene moving.")
	}
	if invokeAdapter.lastInput.ReasoningEffort != "low" {
		t.Fatalf("invokeAdapter.lastInput.ReasoningEffort = %q, want %q", invokeAdapter.lastInput.ReasoningEffort, "low")
	}
}

func TestInvocationServiceInvokeAgentOwnedAnthropicCredentialSuccess(t *testing.T) {
	agentStore := aifakes.NewAgentStore()
	credentialStore := aifakes.NewCredentialStore()
	now := time.Date(2026, 3, 23, 18, 2, 0, 0, time.UTC)
	agentStore.Agents["agent-1"] = agent.Agent{
		ID:            "agent-1",
		OwnerUserID:   "user-1",
		Label:         "oracle",
		Provider:      provider.Anthropic,
		Model:         "claude-sonnet-4-5",
		Instructions:  "Answer concisely.",
		AuthReference: agent.CredentialAuthReference("cred-1"),
		Status:        agent.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	credentialStore.Credentials["cred-1"] = credential.Credential{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         provider.Anthropic,
		Label:            "Claude",
		SecretCiphertext: "enc:sk-ant-1",
		Status:           credential.StatusActive,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	invokeAdapter := &fakeInvocationAdapter{
		invokeResult: provider.InvokeResult{
			OutputText: "Hello from Claude",
			Usage:      provider.Usage{InputTokens: 8, OutputTokens: 6, TotalTokens: 14},
		},
	}
	svc := mustNewInvocationService(t, invocationServiceDeps{
		agentStore:      agentStore,
		credentialStore: credentialStore,
		invokeAdapters: map[provider.Provider]provider.InvocationAdapter{
			provider.Anthropic: invokeAdapter,
		},
		clock: func() time.Time { return now },
	})

	result, err := svc.InvokeAgent(context.Background(), InvokeAgentInput{
		CallerUserID: "user-1",
		AgentID:      "agent-1",
		Input:        "Say hello",
	})
	if err != nil {
		t.Fatalf("InvokeAgent: %v", err)
	}
	if result.Provider != provider.Anthropic {
		t.Fatalf("result.Provider = %q, want %q", result.Provider, provider.Anthropic)
	}
	if result.Model != "claude-sonnet-4-5" {
		t.Fatalf("result.Model = %q, want %q", result.Model, "claude-sonnet-4-5")
	}
	if invokeAdapter.lastInput.AuthToken != "sk-ant-1" {
		t.Fatalf("invokeAdapter.lastInput.AuthToken = %q, want %q", invokeAdapter.lastInput.AuthToken, "sk-ant-1")
	}
}

func TestInvocationServiceInvokeAgentProviderGrantPathWithoutCredentialStore(t *testing.T) {
	agentStore := aifakes.NewAgentStore()
	providerGrantStore := aifakes.NewProviderGrantStore()
	now := time.Date(2026, 3, 23, 18, 5, 0, 0, time.UTC)
	agentStore.Agents["agent-1"] = agent.Agent{
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
	providerGrantStore.ProviderGrants["grant-1"] = providergrant.ProviderGrant{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        provider.OpenAI,
		TokenCiphertext: `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		Status:          providergrant.StatusActive,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	invokeAdapter := &fakeInvocationAdapter{invokeResult: provider.InvokeResult{OutputText: "Hello"}}
	svc := mustNewInvocationService(t, invocationServiceDeps{
		agentStore:         agentStore,
		providerGrantStore: providerGrantStore,
		invokeAdapters: map[provider.Provider]provider.InvocationAdapter{
			provider.OpenAI: invokeAdapter,
		},
		clock: func() time.Time { return now },
	})

	result, err := svc.InvokeAgent(context.Background(), InvokeAgentInput{
		CallerUserID: "user-1",
		AgentID:      "agent-1",
		Input:        "Say hello",
	})
	if err != nil {
		t.Fatalf("InvokeAgent: %v", err)
	}
	if result.OutputText != "Hello" {
		t.Fatalf("result.OutputText = %q, want %q", result.OutputText, "Hello")
	}
	if invokeAdapter.lastInput.AuthToken != "at-1" {
		t.Fatalf("invokeAdapter.lastInput.AuthToken = %q, want %q", invokeAdapter.lastInput.AuthToken, "at-1")
	}
}

func TestInvocationServiceInvokeAgentCredentialPathWithoutCredentialStoreFails(t *testing.T) {
	agentStore := aifakes.NewAgentStore()
	now := time.Date(2026, 3, 23, 18, 10, 0, 0, time.UTC)
	agentStore.Agents["agent-1"] = agent.Agent{
		ID:            "agent-1",
		OwnerUserID:   "user-1",
		Label:         "narrator",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: agent.CredentialAuthReference("cred-1"),
		Status:        agent.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	svc := mustNewInvocationService(t, invocationServiceDeps{
		agentStore: agentStore,
		invokeAdapters: map[provider.Provider]provider.InvocationAdapter{
			provider.OpenAI: &fakeInvocationAdapter{invokeResult: provider.InvokeResult{OutputText: "Hello"}},
		},
		clock: func() time.Time { return now },
	})

	_, err := svc.InvokeAgent(context.Background(), InvokeAgentInput{
		CallerUserID: "user-1",
		AgentID:      "agent-1",
		Input:        "Say hello",
	})
	if got := ErrorKindOf(err); got != ErrKindInternal {
		t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindInternal)
	}
}

func TestInvocationServiceInvokeAgentWithProviderGrantRefreshesNearExpiry(t *testing.T) {
	agentStore := aifakes.NewAgentStore()
	providerGrantStore := aifakes.NewProviderGrantStore()
	now := time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC)
	expiresAt := now.Add(30 * time.Second)
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
	providerGrantStore.ProviderGrants["grant-1"] = providergrant.ProviderGrant{
		ID:               "grant-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		GrantedScopes:    []string{"responses.read"},
		TokenCiphertext:  `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		RefreshSupported: true,
		Status:           providergrant.StatusActive,
		ExpiresAt:        &expiresAt,
		CreatedAt:        now.Add(-time.Hour),
		UpdatedAt:        now.Add(-time.Hour),
	}

	oauthAdapter := &fakeOAuthAdapter{
		refreshResult: provideroauth.TokenExchangeResult{
			TokenPayload: provideroauth.TokenPayload{AccessToken: "at-2", RefreshToken: "rt-2"},
			ExpiresAt:    ptrTime(now.Add(time.Hour)),
		},
	}
	invokeAdapter := &fakeInvocationAdapter{invokeResult: provider.InvokeResult{OutputText: "Hello from refreshed grant"}}
	svc := mustNewInvocationService(t, invocationServiceDeps{
		agentStore:         agentStore,
		providerGrantStore: providerGrantStore,
		oauthAdapters: map[provider.Provider]provideroauth.Adapter{
			provider.OpenAI: oauthAdapter,
		},
		invokeAdapters: map[provider.Provider]provider.InvocationAdapter{
			provider.OpenAI: invokeAdapter,
		},
		clock: func() time.Time { return now },
	})

	result, err := svc.InvokeAgent(context.Background(), InvokeAgentInput{
		CallerUserID: "user-1",
		AgentID:      "agent-1",
		Input:        "Say hello",
	})
	if err != nil {
		t.Fatalf("InvokeAgent: %v", err)
	}
	if oauthAdapter.lastRefreshToken != "rt-1" {
		t.Fatalf("oauthAdapter.lastRefreshToken = %q, want %q", oauthAdapter.lastRefreshToken, "rt-1")
	}
	if invokeAdapter.lastInput.AuthToken != "at-2" {
		t.Fatalf("invokeAdapter.lastInput.AuthToken = %q, want %q", invokeAdapter.lastInput.AuthToken, "at-2")
	}
	if result.OutputText != "Hello from refreshed grant" {
		t.Fatalf("result.OutputText = %q, want %q", result.OutputText, "Hello from refreshed grant")
	}
}

func TestInvocationServiceInvokeAgentRefreshFailedGrantWithoutRefreshSupportFails(t *testing.T) {
	agentStore := aifakes.NewAgentStore()
	providerGrantStore := aifakes.NewProviderGrantStore()
	now := time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC)
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
	providerGrantStore.ProviderGrants["grant-1"] = providergrant.ProviderGrant{
		ID:               "grant-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		TokenCiphertext:  `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		RefreshSupported: false,
		Status:           providergrant.StatusRefreshFailed,
		CreatedAt:        now.Add(-time.Hour),
		UpdatedAt:        now.Add(-time.Hour),
	}

	invokeAdapter := &fakeInvocationAdapter{invokeResult: provider.InvokeResult{OutputText: "Hello"}}
	svc := mustNewInvocationService(t, invocationServiceDeps{
		agentStore:         agentStore,
		providerGrantStore: providerGrantStore,
		invokeAdapters: map[provider.Provider]provider.InvocationAdapter{
			provider.OpenAI: invokeAdapter,
		},
		clock: func() time.Time { return now },
	})

	_, err := svc.InvokeAgent(context.Background(), InvokeAgentInput{
		CallerUserID: "user-1",
		AgentID:      "agent-1",
		Input:        "Say hello",
	})
	if got := ErrorKindOf(err); got != ErrKindFailedPrecondition {
		t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindFailedPrecondition)
	}
	if got := strings.TrimSpace(invokeAdapter.lastInput.AuthToken); got != "" {
		t.Fatalf("invokeAdapter.lastInput.AuthToken = %q, want empty", got)
	}
}

func TestInvocationServiceInvokeAgentExpiredGrantWithoutRefreshSupportFails(t *testing.T) {
	agentStore := aifakes.NewAgentStore()
	providerGrantStore := aifakes.NewProviderGrantStore()
	now := time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC)
	expiresAt := now.Add(-time.Minute)
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
	providerGrantStore.ProviderGrants["grant-1"] = providergrant.ProviderGrant{
		ID:               "grant-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		TokenCiphertext:  `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		RefreshSupported: false,
		Status:           providergrant.StatusActive,
		ExpiresAt:        &expiresAt,
		CreatedAt:        now.Add(-time.Hour),
		UpdatedAt:        now.Add(-time.Hour),
	}

	invokeAdapter := &fakeInvocationAdapter{invokeResult: provider.InvokeResult{OutputText: "Hello"}}
	svc := mustNewInvocationService(t, invocationServiceDeps{
		agentStore:         agentStore,
		providerGrantStore: providerGrantStore,
		invokeAdapters: map[provider.Provider]provider.InvocationAdapter{
			provider.OpenAI: invokeAdapter,
		},
		clock: func() time.Time { return now },
	})

	_, err := svc.InvokeAgent(context.Background(), InvokeAgentInput{
		CallerUserID: "user-1",
		AgentID:      "agent-1",
		Input:        "Say hello",
	})
	if got := ErrorKindOf(err); got != ErrKindFailedPrecondition {
		t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindFailedPrecondition)
	}
	if got := strings.TrimSpace(invokeAdapter.lastInput.AuthToken); got != "" {
		t.Fatalf("invokeAdapter.lastInput.AuthToken = %q, want empty", got)
	}
}

func TestInvocationServiceInvokeAgentRefreshFailedGrantRefreshesAndInvokes(t *testing.T) {
	agentStore := aifakes.NewAgentStore()
	providerGrantStore := aifakes.NewProviderGrantStore()
	now := time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC)
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
	providerGrantStore.ProviderGrants["grant-1"] = providergrant.ProviderGrant{
		ID:               "grant-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		TokenCiphertext:  `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		RefreshSupported: true,
		Status:           providergrant.StatusRefreshFailed,
		CreatedAt:        now.Add(-time.Hour),
		UpdatedAt:        now.Add(-time.Hour),
	}

	invokeAdapter := &fakeInvocationAdapter{invokeResult: provider.InvokeResult{OutputText: "Hello"}}
	svc := mustNewInvocationService(t, invocationServiceDeps{
		agentStore:         agentStore,
		providerGrantStore: providerGrantStore,
		oauthAdapters: map[provider.Provider]provideroauth.Adapter{
			provider.OpenAI: &fakeOAuthAdapter{
				refreshResult: provideroauth.TokenExchangeResult{
					TokenPayload: provideroauth.TokenPayload{AccessToken: "at-2", RefreshToken: "rt-2"},
					ExpiresAt:    ptrTime(now.Add(time.Hour)),
				},
			},
		},
		invokeAdapters: map[provider.Provider]provider.InvocationAdapter{
			provider.OpenAI: invokeAdapter,
		},
		clock: func() time.Time { return now },
	})

	if _, err := svc.InvokeAgent(context.Background(), InvokeAgentInput{
		CallerUserID: "user-1",
		AgentID:      "agent-1",
		Input:        "Say hello",
	}); err != nil {
		t.Fatalf("InvokeAgent: %v", err)
	}
	if invokeAdapter.lastInput.AuthToken != "at-2" {
		t.Fatalf("invokeAdapter.lastInput.AuthToken = %q, want %q", invokeAdapter.lastInput.AuthToken, "at-2")
	}
}

func TestInvocationServiceInvokeAgentWithProviderGrantMissingAccessTokenFails(t *testing.T) {
	agentStore := aifakes.NewAgentStore()
	providerGrantStore := aifakes.NewProviderGrantStore()
	now := time.Date(2026, 3, 23, 18, 15, 0, 0, time.UTC)
	agentStore.Agents["agent-1"] = agent.Agent{
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
	providerGrantStore.ProviderGrants["grant-1"] = providergrant.ProviderGrant{
		ID:               "grant-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		TokenCiphertext:  `enc:{"refresh_token":"rt-1"}`,
		RefreshSupported: true,
		Status:           providergrant.StatusActive,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	svc := mustNewInvocationService(t, invocationServiceDeps{
		agentStore:         agentStore,
		providerGrantStore: providerGrantStore,
		invokeAdapters: map[provider.Provider]provider.InvocationAdapter{
			provider.OpenAI: &fakeInvocationAdapter{invokeResult: provider.InvokeResult{OutputText: "Hello"}},
		},
		clock: func() time.Time { return now },
	})

	_, err := svc.InvokeAgent(context.Background(), InvokeAgentInput{
		CallerUserID: "user-1",
		AgentID:      "agent-1",
		Input:        "Say hello",
	})
	if got := ErrorKindOf(err); got != ErrKindFailedPrecondition {
		t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindFailedPrecondition)
	}
}

func TestInvocationServiceInvokeAgentSharedAccessUsesTargetedLookupAndWritesAuditEvent(t *testing.T) {
	agentStore := aifakes.NewAgentStore()
	credentialStore := aifakes.NewCredentialStore()
	accessRequestStore := newTrackingAccessRequestStore()
	auditEventStore := aifakes.NewAuditEventStore()
	now := time.Date(2026, 3, 23, 18, 20, 0, 0, time.UTC)
	agentStore.Agents["agent-1"] = agent.Agent{
		ID:            "agent-1",
		OwnerUserID:   "owner-1",
		Label:         "narrator",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: agent.CredentialAuthReference("cred-1"),
		Status:        agent.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	credentialStore.Credentials["cred-1"] = credential.Credential{
		ID:               "cred-1",
		OwnerUserID:      "owner-1",
		Provider:         provider.OpenAI,
		Label:            "Primary",
		SecretCiphertext: "enc:sk-owner",
		Status:           credential.StatusActive,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	accessRequestStore.AccessRequests["request-1"] = accessrequest.AccessRequest{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-1",
		Scope:           accessrequest.ScopeInvoke,
		Status:          accessrequest.StatusApproved,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	invokeAdapter := &fakeInvocationAdapter{invokeResult: provider.InvokeResult{OutputText: "Hello from shared access"}}
	svc := mustNewInvocationService(t, invocationServiceDeps{
		agentStore:         agentStore,
		credentialStore:    credentialStore,
		accessRequestStore: accessRequestStore,
		auditEventStore:    auditEventStore,
		invokeAdapters: map[provider.Provider]provider.InvocationAdapter{
			provider.OpenAI: invokeAdapter,
		},
		clock: func() time.Time { return now },
	})

	result, err := svc.InvokeAgent(context.Background(), InvokeAgentInput{
		CallerUserID: "user-1",
		AgentID:      "agent-1",
		Input:        "hello",
	})
	if err != nil {
		t.Fatalf("InvokeAgent: %v", err)
	}
	if result.OutputText != "Hello from shared access" {
		t.Fatalf("result.OutputText = %q, want %q", result.OutputText, "Hello from shared access")
	}
	if invokeAdapter.lastInput.AuthToken != "sk-owner" {
		t.Fatalf("invokeAdapter.lastInput.AuthToken = %q, want %q", invokeAdapter.lastInput.AuthToken, "sk-owner")
	}
	if accessRequestStore.listByRequesterCalls != 0 {
		t.Fatalf("accessRequestStore.listByRequesterCalls = %d, want 0", accessRequestStore.listByRequesterCalls)
	}
	if accessRequestStore.targetedLookupCalls == 0 {
		t.Fatal("expected targeted approved invoke lookup to be used")
	}
	if len(auditEventStore.AuditEvents) != 1 {
		t.Fatalf("len(auditEventStore.AuditEvents) = %d, want 1", len(auditEventStore.AuditEvents))
	}
	if auditEventStore.AuditEvents[0].EventName != auditevent.NameAgentInvokeShared {
		t.Fatalf("auditEventStore.AuditEvents[0].EventName = %q, want %q", auditEventStore.AuditEvents[0].EventName, auditevent.NameAgentInvokeShared)
	}
	if auditEventStore.AuditEvents[0].AccessRequestID != "request-1" {
		t.Fatalf("auditEventStore.AuditEvents[0].AccessRequestID = %q, want %q", auditEventStore.AuditEvents[0].AccessRequestID, "request-1")
	}
}

func TestInvocationServiceInvokeAgentHiddenWithoutApprovedAccess(t *testing.T) {
	agentStore := aifakes.NewAgentStore()
	now := time.Date(2026, 3, 23, 18, 25, 0, 0, time.UTC)
	agentStore.Agents["agent-1"] = agent.Agent{
		ID:            "agent-1",
		OwnerUserID:   "owner-1",
		Label:         "narrator",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: agent.CredentialAuthReference("cred-1"),
		Status:        agent.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	svc := mustNewInvocationService(t, invocationServiceDeps{
		agentStore:         agentStore,
		credentialStore:    aifakes.NewCredentialStore(),
		accessRequestStore: aifakes.NewAccessRequestStore(),
		invokeAdapters: map[provider.Provider]provider.InvocationAdapter{
			provider.OpenAI: &fakeInvocationAdapter{invokeResult: provider.InvokeResult{OutputText: "Hidden"}},
		},
		clock: func() time.Time { return now },
	})

	_, err := svc.InvokeAgent(context.Background(), InvokeAgentInput{
		CallerUserID: "user-1",
		AgentID:      "agent-1",
		Input:        "Say hello",
	})
	if got := ErrorKindOf(err); got != ErrKindNotFound {
		t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindNotFound)
	}
}

func TestInvocationServiceInvokeAgentMissingAgentIsNotFound(t *testing.T) {
	svc := mustNewInvocationService(t, invocationServiceDeps{
		agentStore: aifakes.NewAgentStore(),
		invokeAdapters: map[provider.Provider]provider.InvocationAdapter{
			provider.OpenAI: &fakeInvocationAdapter{invokeResult: provider.InvokeResult{OutputText: "Hello"}},
		},
	})

	_, err := svc.InvokeAgent(context.Background(), InvokeAgentInput{
		CallerUserID: "user-1",
		AgentID:      "missing",
		Input:        "Say hello",
	})
	if got := ErrorKindOf(err); got != ErrKindNotFound {
		t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindNotFound)
	}
}

func TestInvocationServiceInvokeAgentProviderInvokeFailure(t *testing.T) {
	agentStore := aifakes.NewAgentStore()
	credentialStore := aifakes.NewCredentialStore()
	now := time.Date(2026, 3, 23, 18, 30, 0, 0, time.UTC)
	agentStore.Agents["agent-1"] = agent.Agent{
		ID:            "agent-1",
		OwnerUserID:   "user-1",
		Label:         "narrator",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: agent.CredentialAuthReference("cred-1"),
		Status:        agent.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	credentialStore.Credentials["cred-1"] = credential.Credential{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		Label:            "Main",
		SecretCiphertext: "enc:sk-1",
		Status:           credential.StatusActive,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	svc := mustNewInvocationService(t, invocationServiceDeps{
		agentStore:      agentStore,
		credentialStore: credentialStore,
		invokeAdapters: map[provider.Provider]provider.InvocationAdapter{
			provider.OpenAI: &fakeInvocationAdapter{invokeErr: errors.New("provider unavailable")},
		},
		clock: func() time.Time { return now },
	})

	_, err := svc.InvokeAgent(context.Background(), InvokeAgentInput{
		CallerUserID: "user-1",
		AgentID:      "agent-1",
		Input:        "Say hello",
	})
	if got := ErrorKindOf(err); got != ErrKindInternal {
		t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindInternal)
	}
}

func TestInvocationServiceInvokeAgentRejectsEmptyProviderOutput(t *testing.T) {
	agentStore := aifakes.NewAgentStore()
	credentialStore := aifakes.NewCredentialStore()
	now := time.Date(2026, 3, 23, 18, 35, 0, 0, time.UTC)
	agentStore.Agents["agent-1"] = agent.Agent{
		ID:            "agent-1",
		OwnerUserID:   "user-1",
		Label:         "narrator",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: agent.CredentialAuthReference("cred-1"),
		Status:        agent.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	credentialStore.Credentials["cred-1"] = credential.Credential{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		Label:            "Main",
		SecretCiphertext: "enc:sk-1",
		Status:           credential.StatusActive,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	svc := mustNewInvocationService(t, invocationServiceDeps{
		agentStore:      agentStore,
		credentialStore: credentialStore,
		invokeAdapters: map[provider.Provider]provider.InvocationAdapter{
			provider.OpenAI: &fakeInvocationAdapter{invokeResult: provider.InvokeResult{OutputText: ""}},
		},
		clock: func() time.Time { return now },
	})

	_, err := svc.InvokeAgent(context.Background(), InvokeAgentInput{
		CallerUserID: "user-1",
		AgentID:      "agent-1",
		Input:        "Say hello",
	})
	if got := ErrorKindOf(err); got != ErrKindInternal {
		t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindInternal)
	}
}

func TestInvocationServiceInvokeAgentMissingCredentialFailsPrecondition(t *testing.T) {
	agentStore := aifakes.NewAgentStore()
	now := time.Date(2026, 3, 23, 18, 40, 0, 0, time.UTC)
	agentStore.Agents["agent-1"] = agent.Agent{
		ID:            "agent-1",
		OwnerUserID:   "user-1",
		Label:         "narrator",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: agent.CredentialAuthReference("cred-missing"),
		Status:        agent.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	svc := mustNewInvocationService(t, invocationServiceDeps{
		agentStore:      agentStore,
		credentialStore: aifakes.NewCredentialStore(),
		invokeAdapters: map[provider.Provider]provider.InvocationAdapter{
			provider.OpenAI: &fakeInvocationAdapter{invokeResult: provider.InvokeResult{OutputText: "Hello"}},
		},
		clock: func() time.Time { return now },
	})

	_, err := svc.InvokeAgent(context.Background(), InvokeAgentInput{
		CallerUserID: "user-1",
		AgentID:      "agent-1",
		Input:        "Say hello",
	})
	if got := ErrorKindOf(err); got != ErrKindFailedPrecondition {
		t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindFailedPrecondition)
	}
}

func TestInvocationServiceInvokeAgentAdapterUnavailableFailsPrecondition(t *testing.T) {
	agentStore := aifakes.NewAgentStore()
	credentialStore := aifakes.NewCredentialStore()
	now := time.Date(2026, 3, 23, 18, 45, 0, 0, time.UTC)
	agentStore.Agents["agent-1"] = agent.Agent{
		ID:            "agent-1",
		OwnerUserID:   "user-1",
		Label:         "narrator",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: agent.CredentialAuthReference("cred-1"),
		Status:        agent.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	credentialStore.Credentials["cred-1"] = credential.Credential{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		Label:            "Main",
		SecretCiphertext: "enc:sk-1",
		Status:           credential.StatusActive,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	svc := mustNewInvocationService(t, invocationServiceDeps{
		agentStore:      agentStore,
		credentialStore: credentialStore,
		invokeAdapters:  map[provider.Provider]provider.InvocationAdapter{},
		clock:           func() time.Time { return now },
	})

	_, err := svc.InvokeAgent(context.Background(), InvokeAgentInput{
		CallerUserID: "user-1",
		AgentID:      "agent-1",
		Input:        "Say hello",
	})
	if got := ErrorKindOf(err); got != ErrKindFailedPrecondition {
		t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindFailedPrecondition)
	}
}

func TestInvocationServiceInvokeAgentSecretOpenFailureIsInternal(t *testing.T) {
	agentStore := aifakes.NewAgentStore()
	credentialStore := aifakes.NewCredentialStore()
	now := time.Date(2026, 3, 23, 18, 50, 0, 0, time.UTC)
	agentStore.Agents["agent-1"] = agent.Agent{
		ID:            "agent-1",
		OwnerUserID:   "user-1",
		Label:         "narrator",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: agent.CredentialAuthReference("cred-1"),
		Status:        agent.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	credentialStore.Credentials["cred-1"] = credential.Credential{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		Label:            "Main",
		SecretCiphertext: "enc:sk-1",
		Status:           credential.StatusActive,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	svc := mustNewInvocationService(t, invocationServiceDeps{
		agentStore:      agentStore,
		credentialStore: credentialStore,
		sealer:          &aifakes.Sealer{OpenErr: errors.New("open fail")},
		invokeAdapters: map[provider.Provider]provider.InvocationAdapter{
			provider.OpenAI: &fakeInvocationAdapter{invokeResult: provider.InvokeResult{OutputText: "Hello"}},
		},
		clock: func() time.Time { return now },
	})

	_, err := svc.InvokeAgent(context.Background(), InvokeAgentInput{
		CallerUserID: "user-1",
		AgentID:      "agent-1",
		Input:        "Say hello",
	})
	if got := ErrorKindOf(err); got != ErrKindInternal {
		t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindInternal)
	}
}

type invocationServiceDeps struct {
	agentStore         *aifakes.AgentStore
	credentialStore    *aifakes.CredentialStore
	providerGrantStore *aifakes.ProviderGrantStore
	accessRequestStore storage.AccessRequestStore
	auditEventStore    *aifakes.AuditEventStore
	sealer             *aifakes.Sealer
	oauthAdapters      map[provider.Provider]provideroauth.Adapter
	invokeAdapters     map[provider.Provider]provider.InvocationAdapter
	clock              Clock
}

func mustNewInvocationService(t *testing.T, deps invocationServiceDeps) *InvocationService {
	t.Helper()

	agentStore := deps.agentStore
	if agentStore == nil {
		agentStore = aifakes.NewAgentStore()
	}
	sealer := deps.sealer
	if sealer == nil {
		sealer = &aifakes.Sealer{}
	}
	providerGrantRuntime := NewProviderGrantRuntime(ProviderGrantRuntimeConfig{
		ProviderGrantStore: providerGrantStoreForConfig(deps.providerGrantStore),
		ProviderRegistry:   mustProviderRegistryForTests(t, deps.oauthAdapters, deps.invokeAdapters, nil, nil),
		Sealer:             sealer,
		Clock:              deps.clock,
	})
	authMaterialResolver := NewAuthMaterialResolver(AuthMaterialResolverConfig{
		CredentialStore:      credentialStoreForConfig(deps.credentialStore),
		Sealer:               sealer,
		ProviderGrantRuntime: providerGrantRuntime,
	})
	accessibleAgentResolver := NewAccessibleAgentResolver(agentStore, deps.accessRequestStore)

	svc, err := NewInvocationService(InvocationServiceConfig{
		AgentStore:              agentStore,
		AuditEventStore:         deps.auditEventStore,
		AccessibleAgentResolver: accessibleAgentResolver,
		AuthMaterialResolver:    authMaterialResolver,
		ProviderRegistry:        mustProviderRegistryForTests(t, deps.oauthAdapters, deps.invokeAdapters, nil, nil),
		Clock:                   deps.clock,
	})
	if err != nil {
		t.Fatalf("NewInvocationService: %v", err)
	}
	return svc
}

func credentialStoreForConfig(store *aifakes.CredentialStore) storage.CredentialStore {
	if store == nil {
		return nil
	}
	return store
}

func providerGrantStoreForConfig(store *aifakes.ProviderGrantStore) storage.ProviderGrantStore {
	if store == nil {
		return nil
	}
	return store
}
