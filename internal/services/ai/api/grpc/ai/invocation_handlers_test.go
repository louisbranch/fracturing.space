package ai

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func TestInvokeAgentRequiresUserID(t *testing.T) {
	svc := newInvocationHandlersWithStores(newFakeStore(), newFakeStore(), &fakeSealer{})
	_, err := svc.InvokeAgent(context.Background(), &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestInvokeAgentRequiresRequest(t *testing.T) {
	svc := newInvocationHandlersWithStores(newFakeStore(), newFakeStore(), &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestInvokeAgentRequiresAgentID(t *testing.T) {
	svc := newInvocationHandlersWithStores(newFakeStore(), newFakeStore(), &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: " ",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestInvokeAgentRequiresInput(t *testing.T) {
	svc := newInvocationHandlersWithStores(newFakeStore(), newFakeStore(), &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   " ",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestInvokeAgentRequiresActiveOwnedCredential(t *testing.T) {
	store := newFakeStore()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-1",
		Label:        "narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	store.Credentials["cred-1"] = storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		Label:            "Main",
		SecretCiphertext: "enc:sk-1",
		Status:           "revoked",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	svc := newInvocationHandlersWithStores(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestInvokeAgentProviderGrantPathWithoutCredentialStore(t *testing.T) {
	store := newFakeStore()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:              "agent-1",
		OwnerUserID:     "user-1",
		Label:           "narrator",
		Provider:        "openai",
		Model:           "gpt-4o-mini",
		ProviderGrantID: "grant-1",
		Status:          "active",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	store.ProviderGrants["grant-1"] = storage.ProviderGrantRecord{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        "openai",
		TokenCiphertext: `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		Status:          "active",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	svc := newInvocationHandlersWithStores(nil, store, &fakeSealer{})
	svc.providerInvocationAdapters[provider.OpenAI] = &fakeProviderInvocationAdapter{
		invokeResult: provider.InvokeResult{OutputText: "Hello"},
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	if err != nil {
		t.Fatalf("invoke agent: %v", err)
	}
}

func TestInvokeAgentCredentialPathWithoutCredentialStore(t *testing.T) {
	store := newFakeStore()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-1",
		Label:        "narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	svc := newInvocationHandlersWithStores(nil, store, &fakeSealer{})
	svc.providerInvocationAdapters[provider.OpenAI] = &fakeProviderInvocationAdapter{
		invokeResult: provider.InvokeResult{OutputText: "Hello"},
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestInvokeAgentSuccess(t *testing.T) {
	store := newFakeStore()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-1",
		Label:        "narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	store.Credentials["cred-1"] = storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		Label:            "Main",
		SecretCiphertext: "enc:sk-1",
		Status:           "active",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	svc := newInvocationHandlersWithStores(store, store, &fakeSealer{})
	adapter := &fakeProviderInvocationAdapter{
		invokeResult: provider.InvokeResult{
			OutputText: "Hello from AI",
			Usage:      provider.Usage{InputTokens: 12, OutputTokens: 7, ReasoningTokens: 3, TotalTokens: 19},
		},
	}
	svc.providerInvocationAdapters[provider.OpenAI] = adapter

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId:         "agent-1",
		Input:           "Say hello",
		ReasoningEffort: "low",
	})
	if err != nil {
		t.Fatalf("invoke agent: %v", err)
	}
	if adapter.lastInput.CredentialSecret != "sk-1" {
		t.Fatalf("credential secret = %q, want %q", adapter.lastInput.CredentialSecret, "sk-1")
	}
	if adapter.lastInput.Model != "gpt-4o-mini" {
		t.Fatalf("model = %q, want %q", adapter.lastInput.Model, "gpt-4o-mini")
	}
	if adapter.lastInput.Input != "Say hello" {
		t.Fatalf("input = %q, want %q", adapter.lastInput.Input, "Say hello")
	}
	if adapter.lastInput.ReasoningEffort != "low" {
		t.Fatalf("reasoning effort = %q, want %q", adapter.lastInput.ReasoningEffort, "low")
	}
	if resp.GetOutputText() != "Hello from AI" {
		t.Fatalf("output_text = %q, want %q", resp.GetOutputText(), "Hello from AI")
	}
	if resp.GetUsage().GetTotalTokens() != 19 {
		t.Fatalf("usage.total_tokens = %d", resp.GetUsage().GetTotalTokens())
	}
}

func TestInvokeAgentWithProviderGrantRefreshesNearExpiry(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC)
	expiresAt := now.Add(30 * time.Second)
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:              "agent-1",
		OwnerUserID:     "user-1",
		Label:           "narrator",
		Provider:        "openai",
		Model:           "gpt-4o-mini",
		CredentialID:    "",
		ProviderGrantID: "grant-1",
		Status:          "active",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	store.ProviderGrants["grant-1"] = storage.ProviderGrantRecord{
		ID:               "grant-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		GrantedScopes:    []string{"responses.read"},
		TokenCiphertext:  `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		RefreshSupported: true,
		Status:           "active",
		ExpiresAt:        &expiresAt,
		CreatedAt:        now.Add(-time.Hour),
		UpdatedAt:        now.Add(-time.Hour),
	}

	svc := newInvocationHandlersWithStores(store, store, &fakeSealer{})
	svc.clock = func() time.Time { return now }
	oauthAdapter := &fakeProviderOAuthAdapter{
		refreshResult: provider.TokenExchangeResult{
			TokenPlaintext:   `{"access_token":"at-2","refresh_token":"rt-2"}`,
			RefreshSupported: true,
			ExpiresAt:        ptrTime(now.Add(time.Hour)),
		},
	}
	invokeAdapter := &fakeProviderInvocationAdapter{
		invokeResult: provider.InvokeResult{
			OutputText: "Hello from refreshed grant",
		},
	}
	svc.providerOAuthAdapters[provider.OpenAI] = oauthAdapter
	svc.providerInvocationAdapters[provider.OpenAI] = invokeAdapter

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	if err != nil {
		t.Fatalf("invoke agent: %v", err)
	}
	if oauthAdapter.lastRefreshToken != "rt-1" {
		t.Fatalf("refresh token = %q, want %q", oauthAdapter.lastRefreshToken, "rt-1")
	}
	if invokeAdapter.lastInput.CredentialSecret != "at-2" {
		t.Fatalf("invoke auth token = %q, want %q", invokeAdapter.lastInput.CredentialSecret, "at-2")
	}
	if resp.GetOutputText() != "Hello from refreshed grant" {
		t.Fatalf("output_text = %q, want %q", resp.GetOutputText(), "Hello from refreshed grant")
	}
}

func TestInvokeAgentWithRefreshFailedGrantWithoutRefreshSupport(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC)
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:              "agent-1",
		OwnerUserID:     "user-1",
		Label:           "narrator",
		Provider:        "openai",
		Model:           "gpt-4o-mini",
		ProviderGrantID: "grant-1",
		Status:          "active",
		CreatedAt:       now.Add(-time.Hour),
		UpdatedAt:       now.Add(-time.Hour),
	}
	store.ProviderGrants["grant-1"] = storage.ProviderGrantRecord{
		ID:               "grant-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		TokenCiphertext:  `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		RefreshSupported: false,
		Status:           "refresh_failed",
		CreatedAt:        now.Add(-time.Hour),
		UpdatedAt:        now.Add(-time.Hour),
	}

	svc := newInvocationHandlersWithStores(store, store, &fakeSealer{})
	svc.clock = func() time.Time { return now }
	svc.providerInvocationAdapters[provider.OpenAI] = &fakeProviderInvocationAdapter{
		invokeResult: provider.InvokeResult{OutputText: "Hello"},
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestInvokeAgentWithExpiredGrantWithoutRefreshSupport(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC)
	expiresAt := now.Add(-time.Minute)
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:              "agent-1",
		OwnerUserID:     "user-1",
		Label:           "narrator",
		Provider:        "openai",
		Model:           "gpt-4o-mini",
		ProviderGrantID: "grant-1",
		Status:          "active",
		CreatedAt:       now.Add(-time.Hour),
		UpdatedAt:       now.Add(-time.Hour),
	}
	store.ProviderGrants["grant-1"] = storage.ProviderGrantRecord{
		ID:               "grant-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		TokenCiphertext:  `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		RefreshSupported: false,
		Status:           "active",
		ExpiresAt:        &expiresAt,
		CreatedAt:        now.Add(-time.Hour),
		UpdatedAt:        now.Add(-time.Hour),
	}

	svc := newInvocationHandlersWithStores(store, store, &fakeSealer{})
	svc.clock = func() time.Time { return now }
	invokeAdapter := &fakeProviderInvocationAdapter{
		invokeResult: provider.InvokeResult{OutputText: "Hello"},
	}
	svc.providerInvocationAdapters[provider.OpenAI] = invokeAdapter

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
	if got := strings.TrimSpace(invokeAdapter.lastInput.CredentialSecret); got != "" {
		t.Fatalf("invoke adapter should not be called, got credential secret %q", got)
	}
}

func TestInvokeAgentWithRefreshFailedGrantRefreshesAndInvokes(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC)
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:              "agent-1",
		OwnerUserID:     "user-1",
		Label:           "narrator",
		Provider:        "openai",
		Model:           "gpt-4o-mini",
		ProviderGrantID: "grant-1",
		Status:          "active",
		CreatedAt:       now.Add(-time.Hour),
		UpdatedAt:       now.Add(-time.Hour),
	}
	store.ProviderGrants["grant-1"] = storage.ProviderGrantRecord{
		ID:               "grant-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		TokenCiphertext:  `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		RefreshSupported: true,
		Status:           "refresh_failed",
		CreatedAt:        now.Add(-time.Hour),
		UpdatedAt:        now.Add(-time.Hour),
	}

	svc := newInvocationHandlersWithStores(store, store, &fakeSealer{})
	svc.clock = func() time.Time { return now }
	svc.providerOAuthAdapters[provider.OpenAI] = &fakeProviderOAuthAdapter{
		refreshResult: provider.TokenExchangeResult{
			TokenPlaintext:   `{"access_token":"at-2","refresh_token":"rt-2"}`,
			RefreshSupported: true,
			ExpiresAt:        ptrTime(now.Add(time.Hour)),
		},
	}
	invokeAdapter := &fakeProviderInvocationAdapter{
		invokeResult: provider.InvokeResult{OutputText: "Hello"},
	}
	svc.providerInvocationAdapters[provider.OpenAI] = invokeAdapter

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	if _, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	}); err != nil {
		t.Fatalf("invoke agent: %v", err)
	}
	if got := invokeAdapter.lastInput.CredentialSecret; got != "at-2" {
		t.Fatalf("credential secret = %q, want %q", got, "at-2")
	}
}

func TestInvokeAgentWithProviderGrantMissingAccessToken(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:              "agent-1",
		OwnerUserID:     "user-1",
		Label:           "narrator",
		Provider:        "openai",
		Model:           "gpt-4o-mini",
		ProviderGrantID: "grant-1",
		Status:          "active",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	store.ProviderGrants["grant-1"] = storage.ProviderGrantRecord{
		ID:               "grant-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		TokenCiphertext:  `enc:{"refresh_token":"rt-1"}`,
		RefreshSupported: true,
		Status:           "active",
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	svc := newInvocationHandlersWithStores(store, store, &fakeSealer{})
	svc.providerInvocationAdapters[provider.OpenAI] = &fakeProviderInvocationAdapter{
		invokeResult: provider.InvokeResult{OutputText: "Hello"},
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestInvokeAgentMissingAgentIsNotFound(t *testing.T) {
	svc := newInvocationHandlersWithStores(newFakeStore(), newFakeStore(), &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "missing",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestInvokeAgentHiddenForNonOwner(t *testing.T) {
	store := newFakeStore()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-2",
		Label:        "narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	svc := newInvocationHandlersWithStores(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestInvokeAgentApprovedRequesterCanInvokeOwnerAgent(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-owner",
		Label:        "narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.Credentials["cred-1"] = storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-owner",
		Provider:         "openai",
		Label:            "Main",
		SecretCiphertext: "enc:sk-owner",
		Status:           "active",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	store.AccessRequests["request-1"] = storage.AccessRequestRecord{
		ID:              "request-1",
		RequesterUserID: "user-requester",
		OwnerUserID:     "user-owner",
		AgentID:         "agent-1",
		Scope:           "invoke",
		Status:          "approved",
		CreatedAt:       now.Add(-time.Minute),
		UpdatedAt:       now,
	}

	svc := newInvocationHandlersWithStores(store, store, &fakeSealer{})
	invokeAdapter := &fakeProviderInvocationAdapter{
		invokeResult: provider.InvokeResult{OutputText: "Shared response"},
	}
	svc.providerInvocationAdapters[provider.OpenAI] = invokeAdapter

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-requester"))
	resp, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	if err != nil {
		t.Fatalf("invoke agent: %v", err)
	}
	if got := resp.GetOutputText(); got != "Shared response" {
		t.Fatalf("output_text = %q, want %q", got, "Shared response")
	}
	if got := invokeAdapter.lastInput.CredentialSecret; got != "sk-owner" {
		t.Fatalf("credential secret = %q, want %q", got, "sk-owner")
	}
}

func TestInvokeAgentApprovedRequesterWritesAuditEvent(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-owner",
		Label:        "narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.Credentials["cred-1"] = storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-owner",
		Provider:         "openai",
		Label:            "Main",
		SecretCiphertext: "enc:sk-owner",
		Status:           "active",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	store.AccessRequests["request-1"] = storage.AccessRequestRecord{
		ID:              "request-1",
		RequesterUserID: "user-requester",
		OwnerUserID:     "user-owner",
		AgentID:         "agent-1",
		Scope:           "invoke",
		Status:          "approved",
		CreatedAt:       now.Add(-time.Minute),
		UpdatedAt:       now,
	}

	svc := newInvocationHandlersWithStores(store, store, &fakeSealer{})
	svc.providerInvocationAdapters[provider.OpenAI] = &fakeProviderInvocationAdapter{
		invokeResult: provider.InvokeResult{OutputText: "Shared response"},
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-requester"))
	if _, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	}); err != nil {
		t.Fatalf("invoke agent: %v", err)
	}
	if len(store.AuditEventNames) != 1 {
		t.Fatalf("audit events len = %d, want 1", len(store.AuditEventNames))
	}
	if got := store.AuditEventNames[0]; got != "agent.invoke.shared" {
		t.Fatalf("audit event = %q, want %q", got, "agent.invoke.shared")
	}
}

func TestInvokeAgentSharedAccessUsesTargetedLookup(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.ListAccessRequestsByRequesterErr = errors.New("unexpected requester-wide list call")
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "owner-1",
		Label:        "narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.Credentials["cred-1"] = storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "owner-1",
		Provider:         "openai",
		Label:            "Primary",
		SecretCiphertext: "enc:sk-owner",
		Status:           "active",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	store.AccessRequests["request-1"] = storage.AccessRequestRecord{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-1",
		Scope:           "invoke",
		Status:          "approved",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	svc := newInvocationHandlersWithStores(store, store, &fakeSealer{})
	adapter := &fakeProviderInvocationAdapter{
		invokeResult: provider.InvokeResult{OutputText: "Hello from shared access"},
	}
	svc.providerInvocationAdapters[provider.OpenAI] = adapter

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "hello",
	})
	if err != nil {
		t.Fatalf("invoke agent: %v", err)
	}
	if got := resp.GetOutputText(); got != "Hello from shared access" {
		t.Fatalf("output_text = %q, want %q", got, "Hello from shared access")
	}
	if store.ListAccessRequestsByRequesterCalls != 0 {
		t.Fatalf("requester-wide list calls = %d, want 0", store.ListAccessRequestsByRequesterCalls)
	}
	if store.GetApprovedInvokeAccessCalls == 0 {
		t.Fatal("expected targeted approved invoke lookup to be used")
	}
}

func TestInvokeAgentApprovedRequesterDeniedAfterRevocation(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-owner",
		Label:        "narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.Credentials["cred-1"] = storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-owner",
		Provider:         "openai",
		Label:            "Main",
		SecretCiphertext: "enc:sk-owner",
		Status:           "active",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	store.AccessRequests["request-1"] = storage.AccessRequestRecord{
		ID:              "request-1",
		RequesterUserID: "user-requester",
		OwnerUserID:     "user-owner",
		AgentID:         "agent-1",
		Scope:           "invoke",
		Status:          "approved",
		CreatedAt:       now.Add(-time.Minute),
		UpdatedAt:       now,
	}

	svc := newInvocationHandlersWithStores(store, store, &fakeSealer{})
	svc.providerInvocationAdapters[provider.OpenAI] = &fakeProviderInvocationAdapter{
		invokeResult: provider.InvokeResult{OutputText: "Shared response"},
	}
	accessSvc := newAccessRequestHandlersWithStores(store, store, store)
	ownerCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-owner"))
	if _, err := accessSvc.RevokeAccessRequest(ownerCtx, &aiv1.RevokeAccessRequestRequest{
		AccessRequestId: "request-1",
		RevokeNote:      "removed",
	}); err != nil {
		t.Fatalf("revoke access request: %v", err)
	}

	requesterCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-requester"))
	_, err := svc.InvokeAgent(requesterCtx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestInvokeAgentPendingRequesterCannotInvokeOwnerAgent(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-owner",
		Label:        "narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.Credentials["cred-1"] = storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-owner",
		Provider:         "openai",
		Label:            "Main",
		SecretCiphertext: "enc:sk-owner",
		Status:           "active",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	store.AccessRequests["request-1"] = storage.AccessRequestRecord{
		ID:              "request-1",
		RequesterUserID: "user-requester",
		OwnerUserID:     "user-owner",
		AgentID:         "agent-1",
		Scope:           "invoke",
		Status:          "pending",
		CreatedAt:       now.Add(-time.Minute),
		UpdatedAt:       now,
	}

	svc := newInvocationHandlersWithStores(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-requester"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestInvokeAgentDeniedRequesterCannotInvokeOwnerAgent(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-owner",
		Label:        "narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.Credentials["cred-1"] = storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-owner",
		Provider:         "openai",
		Label:            "Main",
		SecretCiphertext: "enc:sk-owner",
		Status:           "active",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	store.AccessRequests["request-1"] = storage.AccessRequestRecord{
		ID:              "request-1",
		RequesterUserID: "user-requester",
		OwnerUserID:     "user-owner",
		AgentID:         "agent-1",
		Scope:           "invoke",
		Status:          "denied",
		CreatedAt:       now.Add(-time.Minute),
		UpdatedAt:       now,
	}

	svc := newInvocationHandlersWithStores(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-requester"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestInvokeAgentMissingCredentialIsFailedPrecondition(t *testing.T) {
	store := newFakeStore()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-1",
		Label:        "narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-missing",
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	svc := newInvocationHandlersWithStores(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestInvokeAgentProviderInvokeError(t *testing.T) {
	store := newFakeStore()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-1",
		Label:        "narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	store.Credentials["cred-1"] = storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		Label:            "Main",
		SecretCiphertext: "enc:sk-1",
		Status:           "active",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	svc := newInvocationHandlersWithStores(store, store, &fakeSealer{})
	svc.providerInvocationAdapters[provider.OpenAI] = &fakeProviderInvocationAdapter{invokeErr: errors.New("provider unavailable")}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestInvokeAgentEmptyProviderOutputIsInternal(t *testing.T) {
	store := newFakeStore()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-1",
		Label:        "narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	store.Credentials["cred-1"] = storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		Label:            "Main",
		SecretCiphertext: "enc:sk-1",
		Status:           "active",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	svc := newInvocationHandlersWithStores(store, store, &fakeSealer{})
	svc.providerInvocationAdapters[provider.OpenAI] = &fakeProviderInvocationAdapter{
		invokeResult: provider.InvokeResult{OutputText: "   "},
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestInvokeAgentAdapterUnavailable(t *testing.T) {
	store := newFakeStore()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-1",
		Label:        "narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	store.Credentials["cred-1"] = storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		Label:            "Main",
		SecretCiphertext: "enc:sk-1",
		Status:           "active",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	svc := newInvocationHandlersWithStores(store, store, &fakeSealer{})
	delete(svc.providerInvocationAdapters, provider.OpenAI)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestInvokeAgentSecretOpenError(t *testing.T) {
	store := newFakeStore()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-1",
		Label:        "narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	store.Credentials["cred-1"] = storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		Label:            "Main",
		SecretCiphertext: "enc:sk-1",
		Status:           "active",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	svc := newInvocationHandlersWithStores(store, store, &fakeSealer{OpenErr: errors.New("open fail")})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.Internal)
}
