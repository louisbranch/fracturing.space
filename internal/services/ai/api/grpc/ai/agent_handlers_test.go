package ai

import (
	"context"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/credential"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func TestCreateAgentRequiresUserID(t *testing.T) {
	svc := newAgentHandlersWithStores(t, newFakeStore(), newFakeStore(), &fakeSealer{})
	_, err := svc.CreateAgent(context.Background(), &aiv1.CreateAgentRequest{
		Label:         "narrator",
		Provider:      aiv1.Provider_PROVIDER_OPENAI,
		Model:         "gpt-4o-mini",
		AuthReference: credentialAuthReferenceProto("cred-1"),
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestCreateAgentRejectsInvalidProvider(t *testing.T) {
	svc := newAgentHandlersWithStores(t, newFakeStore(), newFakeStore(), &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.CreateAgent(ctx, &aiv1.CreateAgentRequest{
		Label:         "narrator",
		Provider:      aiv1.Provider(99),
		Model:         "gpt-4o-mini",
		AuthReference: credentialAuthReferenceProto("cred-1"),
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListProviderModelsRejectsInvalidProvider(t *testing.T) {
	svc := newAgentHandlersWithStores(t, newFakeStore(), newFakeStore(), &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.ListProviderModels(ctx, &aiv1.ListProviderModelsRequest{
		Provider:      aiv1.Provider(99),
		AuthReference: credentialAuthReferenceProto("cred-1"),
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListAgentsIncludesAuthStateAndUsage(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
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
	store.Agents["agent-1"] = agent.Agent{
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

	svc := newAgentHandlersWithOpts(t, store, store, &fakeSealer{}, agentTestOpts{
		campaignUsageReader: &fakeCampaignAIAuthStateClient{
			usageByAgent: map[string]int32{"agent-1": 2},
		},
	})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))

	resp, err := svc.ListAgents(ctx, &aiv1.ListAgentsRequest{PageSize: 10})
	if err != nil {
		t.Fatalf("ListAgents: %v", err)
	}
	if len(resp.GetAgents()) != 1 {
		t.Fatalf("len(resp.GetAgents()) = %d, want 1", len(resp.GetAgents()))
	}
	if got := resp.GetAgents()[0].GetAuthState(); got != aiv1.AgentAuthState_AGENT_AUTH_STATE_READY {
		t.Fatalf("resp.GetAgents()[0].GetAuthState() = %v, want ready", got)
	}
	if got := resp.GetAgents()[0].GetAuthReference().GetId(); got != "cred-1" {
		t.Fatalf("resp.GetAgents()[0].GetAuthReference().GetId() = %q, want %q", got, "cred-1")
	}
	if got := resp.GetAgents()[0].GetActiveCampaignCount(); got != 2 {
		t.Fatalf("resp.GetAgents()[0].GetActiveCampaignCount() = %d, want 2", got)
	}
}

func TestListAccessibleAgentsRequiresUserID(t *testing.T) {
	svc := newAgentHandlersWithStores(t, newFakeStore(), newFakeStore(), &fakeSealer{})
	_, err := svc.ListAccessibleAgents(context.Background(), &aiv1.ListAccessibleAgentsRequest{PageSize: 10})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestGetAccessibleAgentRequiresUserID(t *testing.T) {
	svc := newAgentHandlersWithStores(t, newFakeStore(), newFakeStore(), &fakeSealer{})
	_, err := svc.GetAccessibleAgent(context.Background(), &aiv1.GetAccessibleAgentRequest{
		AgentId: "agent-1",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestGetAccessibleAgentRequiresAgentID(t *testing.T) {
	svc := newAgentHandlersWithStores(t, newFakeStore(), newFakeStore(), &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.GetAccessibleAgent(ctx, &aiv1.GetAccessibleAgentRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestValidateCampaignAgentBindingReturnsBoundAgent(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 3, 23, 21, 15, 0, 0, time.UTC)
	store.Credentials["cred-1"] = credential.Credential{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		Label:            "Main",
		Status:           credential.StatusActive,
		SecretCiphertext: "enc:sk-1",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	store.Agents["agent-1"] = agent.Agent{
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

	svc := newTestAgentHandlers(t, store)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.ValidateCampaignAgentBinding(ctx, &aiv1.ValidateCampaignAgentBindingRequest{AgentId: "agent-1"})
	if err != nil {
		t.Fatalf("ValidateCampaignAgentBinding() error = %v", err)
	}
	if resp.GetAgent().GetId() != "agent-1" {
		t.Fatalf("agent id = %q, want %q", resp.GetAgent().GetId(), "agent-1")
	}
}

func TestUpdateAgentTrimsFieldsAndMapsResponse(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 3, 23, 20, 30, 0, 0, time.UTC)
	store.Credentials["cred-1"] = credential.Credential{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		Label:            "primary",
		SecretCiphertext: "enc:sk-1",
		Status:           credential.StatusActive,
		CreatedAt:        now.Add(-2 * time.Hour),
		UpdatedAt:        now.Add(-2 * time.Hour),
	}
	store.Agents["agent-1"] = agent.Agent{
		ID:            "agent-1",
		OwnerUserID:   "user-1",
		Label:         "narrator",
		Instructions:  "Old instructions.",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: agent.CredentialAuthReference("cred-1"),
		Status:        agent.StatusActive,
		CreatedAt:     now.Add(-time.Hour),
		UpdatedAt:     now.Add(-time.Hour),
	}

	adapter := &fakeProviderInvocationAdapter{}
	svc := newAgentHandlersWithOpts(t, store, store, &fakeSealer{}, agentTestOpts{
		modelAdapters: map[provider.Provider]provider.ModelAdapter{
			provider.OpenAI: adapter,
		},
		clock: func() time.Time { return now },
	})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.UpdateAgent(ctx, &aiv1.UpdateAgentRequest{
		AgentId:      " agent-1 ",
		Label:        " lead-narrator ",
		Instructions: " Keep the session moving. ",
	})
	if err != nil {
		t.Fatalf("UpdateAgent: %v", err)
	}
	if resp.GetAgent().GetLabel() != "lead-narrator" {
		t.Fatalf("resp.GetAgent().GetLabel() = %q, want %q", resp.GetAgent().GetLabel(), "lead-narrator")
	}
	if resp.GetAgent().GetInstructions() != "Keep the session moving." {
		t.Fatalf("resp.GetAgent().GetInstructions() = %q, want %q", resp.GetAgent().GetInstructions(), "Keep the session moving.")
	}
}

func TestDeleteAgentRequiresAgentID(t *testing.T) {
	svc := newAgentHandlersWithStores(t, newFakeStore(), newFakeStore(), &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.DeleteAgent(ctx, &aiv1.DeleteAgentRequest{AgentId: " "})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func credentialAuthReferenceProto(id string) *aiv1.AgentAuthReference {
	return &aiv1.AgentAuthReference{
		Type: aiv1.AgentAuthReferenceType_AGENT_AUTH_REFERENCE_TYPE_CREDENTIAL,
		Id:   id,
	}
}
