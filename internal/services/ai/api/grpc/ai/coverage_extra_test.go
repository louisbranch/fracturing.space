package ai

import (
	"context"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/accessrequest"
	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/credential"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/service"
	"google.golang.org/grpc/metadata"
)

func TestRetrievedContextProtoHelpers(t *testing.T) {
	t.Parallel()

	value := retrievedContextToProto(orchestration.RetrievedContext{
		URI:           "app://story/scene-1",
		RenderedURI:   "Scene 1",
		ContextType:   "story",
		Abstract:      "A storm gathers.",
		MatchReason:   "semantic",
		Score:         0.91,
		ContentSource: "memory",
		ContentError:  "degraded",
	})
	if value.GetUri() != "app://story/scene-1" || value.GetRenderedUri() != "Scene 1" || value.GetScore() != 0.91 {
		t.Fatalf("retrievedContextToProto() = %#v", value)
	}

	if got := retrievedContextsToProto(nil); got != nil {
		t.Fatalf("retrievedContextsToProto(nil) = %#v, want nil", got)
	}
	got := retrievedContextsToProto([]orchestration.RetrievedContext{{URI: "one"}, {URI: "two"}})
	if len(got) != 2 || got[1].GetUri() != "two" {
		t.Fatalf("retrievedContextsToProto() = %#v", got)
	}
}

func TestAgentAuthStateToProtoMappings(t *testing.T) {
	t.Parallel()

	if got := agentAuthStateToProto(service.AgentAuthStateReady); got != aiv1.AgentAuthState_AGENT_AUTH_STATE_READY {
		t.Fatalf("agentAuthStateToProto(ready) = %v", got)
	}
	if got := agentAuthStateToProto(service.AgentAuthStateRevoked); got != aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_REVOKED {
		t.Fatalf("agentAuthStateToProto(revoked) = %v", got)
	}
	if got := agentAuthStateToProto(service.AgentAuthStateUnavailable); got != aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_UNAVAILABLE {
		t.Fatalf("agentAuthStateToProto(unavailable) = %v", got)
	}
	if got := agentAuthStateToProto(service.AgentAuthStateUnknown); got != aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_UNAVAILABLE {
		t.Fatalf("agentAuthStateToProto(default) = %v", got)
	}
}

func TestListProviderModelsAndAccessibleAgentsHappyPaths(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	now := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)
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
	store.Agents["agent-owned"] = agent.Agent{
		ID:            "agent-owned",
		OwnerUserID:   "user-1",
		Label:         "Owned agent",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: agent.CredentialAuthReference("cred-1"),
		Status:        agent.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	store.Agents["agent-shared"] = agent.Agent{
		ID:            "agent-shared",
		OwnerUserID:   "owner-2",
		Label:         "Shared agent",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o",
		AuthReference: agent.CredentialAuthReference("cred-1"),
		Status:        agent.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	store.AccessRequests["request-1"] = accessrequest.AccessRequest{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-2",
		AgentID:         "agent-shared",
		Scope:           accessrequest.ScopeInvoke,
		Status:          accessrequest.StatusApproved,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	modelAdapter := &fakeProviderInvocationAdapter{
		listModelsResult: []provider.Model{{ID: "gpt-4.1-mini"}, {ID: "gpt-4.1"}},
	}
	svc := newAgentHandlersWithOpts(t, store, store, &fakeSealer{}, agentTestOpts{
		modelAdapters: map[provider.Provider]provider.ModelAdapter{
			provider.OpenAI: modelAdapter,
		},
	})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))

	modelsResp, err := svc.ListProviderModels(ctx, &aiv1.ListProviderModelsRequest{
		Provider:      aiv1.Provider_PROVIDER_OPENAI,
		AuthReference: credentialAuthReferenceProto("cred-1"),
	})
	if err != nil {
		t.Fatalf("ListProviderModels() error = %v", err)
	}
	if len(modelsResp.GetModels()) != 2 {
		t.Fatalf("ListProviderModels() = %#v", modelsResp)
	}
	gotModelIDs := []string{modelsResp.GetModels()[0].GetId(), modelsResp.GetModels()[1].GetId()}
	if !((gotModelIDs[0] == "gpt-4.1-mini" && gotModelIDs[1] == "gpt-4.1") || (gotModelIDs[0] == "gpt-4.1" && gotModelIDs[1] == "gpt-4.1-mini")) {
		t.Fatalf("ListProviderModels() = %#v", modelsResp)
	}

	agentsResp, err := svc.ListAccessibleAgents(ctx, &aiv1.ListAccessibleAgentsRequest{PageSize: 10})
	if err != nil {
		t.Fatalf("ListAccessibleAgents() error = %v", err)
	}
	if len(agentsResp.GetAgents()) != 2 {
		t.Fatalf("len(ListAccessibleAgents().Agents) = %d, want 2", len(agentsResp.GetAgents()))
	}
	if agentsResp.GetAgents()[0].GetAuthState() != aiv1.AgentAuthState_AGENT_AUTH_STATE_READY {
		t.Fatalf("auth state = %v, want ready for usable auth reference", agentsResp.GetAgents()[0].GetAuthState())
	}
}
