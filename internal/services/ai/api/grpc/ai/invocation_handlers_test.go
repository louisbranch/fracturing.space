package ai

import (
	"context"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/credential"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/test/grpcassert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func TestInvokeAgentRequiresUserID(t *testing.T) {
	svc := newInvocationHandlersWithStores(t, newFakeStore(), newFakeStore(), &fakeSealer{})
	_, err := svc.InvokeAgent(context.Background(), &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	grpcassert.StatusCode(t, err, codes.PermissionDenied)
}

func TestInvokeAgentRequiresRequest(t *testing.T) {
	svc := newInvocationHandlersWithStores(t, newFakeStore(), newFakeStore(), &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, nil)
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestInvokeAgentRequiresAgentID(t *testing.T) {
	svc := newInvocationHandlersWithStores(t, newFakeStore(), newFakeStore(), &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: " ",
		Input:   "Say hello",
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestInvokeAgentRequiresInput(t *testing.T) {
	svc := newInvocationHandlersWithStores(t, newFakeStore(), newFakeStore(), &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   " ",
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestInvokeAgentTrimsRequestAndMapsResponse(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 3, 23, 19, 0, 0, 0, time.UTC)
	store.Agents["agent-1"] = agent.Agent{
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
	store.Credentials["cred-1"] = credential.Credential{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		Label:            "Main",
		SecretCiphertext: "enc:sk-1",
		Status:           credential.StatusActive,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	invokeAdapter := &fakeProviderInvocationAdapter{
		invokeResult: provider.InvokeResult{
			OutputText: "Hello from AI",
			Usage:      provider.Usage{InputTokens: 12, OutputTokens: 7, ReasoningTokens: 3, TotalTokens: 19},
		},
	}
	svc := newInvocationHandlersWithOpts(t, store, store, &fakeSealer{}, invocationTestOpts{
		invocationAdapters: map[provider.Provider]provider.InvocationAdapter{
			provider.OpenAI: invokeAdapter,
		},
	})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId:         " agent-1 ",
		Input:           " Say hello ",
		ReasoningEffort: " low ",
	})
	if err != nil {
		t.Fatalf("InvokeAgent: %v", err)
	}
	if invokeAdapter.lastInput.Input != "Say hello" {
		t.Fatalf("invokeAdapter.lastInput.Input = %q, want %q", invokeAdapter.lastInput.Input, "Say hello")
	}
	if invokeAdapter.lastInput.ReasoningEffort != "low" {
		t.Fatalf("invokeAdapter.lastInput.ReasoningEffort = %q, want %q", invokeAdapter.lastInput.ReasoningEffort, "low")
	}
	if resp.GetOutputText() != "Hello from AI" {
		t.Fatalf("resp.GetOutputText() = %q, want %q", resp.GetOutputText(), "Hello from AI")
	}
	if resp.GetProvider() != aiv1.Provider_PROVIDER_OPENAI {
		t.Fatalf("resp.GetProvider() = %v, want %v", resp.GetProvider(), aiv1.Provider_PROVIDER_OPENAI)
	}
	if resp.GetModel() != "gpt-4o-mini" {
		t.Fatalf("resp.GetModel() = %q, want %q", resp.GetModel(), "gpt-4o-mini")
	}
	if resp.GetUsage().GetTotalTokens() != 19 {
		t.Fatalf("resp.GetUsage().GetTotalTokens() = %d, want %d", resp.GetUsage().GetTotalTokens(), 19)
	}
}
