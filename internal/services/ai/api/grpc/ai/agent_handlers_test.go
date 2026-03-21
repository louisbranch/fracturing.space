package ai

import (
	"context"
	"errors"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func TestCreateAgentRequiresActiveOwnedCredential(t *testing.T) {
	store := newFakeStore()
	store.Credentials["cred-1"] = storage.CredentialRecord{ID: "cred-1", OwnerUserID: "user-1", Provider: "openai", Label: "A", Status: "revoked", CreatedAt: time.Now(), UpdatedAt: time.Now()}

	svc := newTestAgentHandlers(store)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.CreateAgent(ctx, &aiv1.CreateAgentRequest{
		Label:        "narrator",
		Provider:     aiv1.Provider_PROVIDER_OPENAI,
		Model:        "gpt-4o-mini",
		CredentialId: "cred-1",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestCreateAgentSuccess(t *testing.T) {
	store := newFakeStore()
	store.Credentials["cred-1"] = storage.CredentialRecord{ID: "cred-1", OwnerUserID: "user-1", Provider: "openai", Label: "A", Status: "active", CreatedAt: time.Now(), UpdatedAt: time.Now()}

	svc := newTestAgentHandlers(store)
	svc.clock = func() time.Time { return time.Date(2026, 2, 15, 22, 57, 0, 0, time.UTC) }
	svc.idGenerator = func() (string, error) { return "agent-1", nil }

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.CreateAgent(ctx, &aiv1.CreateAgentRequest{
		Label:        "narrator",
		Provider:     aiv1.Provider_PROVIDER_OPENAI,
		Model:        "gpt-4o-mini",
		CredentialId: "cred-1",
		Instructions: "Keep the scene moving.",
	})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if resp.GetAgent().GetId() != "agent-1" {
		t.Fatalf("agent id = %q, want %q", resp.GetAgent().GetId(), "agent-1")
	}
	if got := resp.GetAgent().GetInstructions(); got != "Keep the scene moving." {
		t.Fatalf("instructions = %q, want %q", got, "Keep the scene moving.")
	}
}

func TestListProviderModelsReturnsNewestFirst(t *testing.T) {
	store := newFakeStore()
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

	adapter := &fakeProviderInvocationAdapter{
		listModelsResult: []provider.Model{
			{ID: "alpha", OwnedBy: "openai", Created: 100},
			{ID: "zeta", OwnedBy: "openai", Created: 200},
			{ID: "beta", OwnedBy: "openai", Created: 200},
			{ID: "", OwnedBy: "openai", Created: 300},
		},
	}
	svc := newAgentHandlersWithStores(store, store, &fakeSealer{})
	svc.providerModelAdapters[provider.OpenAI] = adapter

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.ListProviderModels(ctx, &aiv1.ListProviderModelsRequest{
		Provider:     aiv1.Provider_PROVIDER_OPENAI,
		CredentialId: "cred-1",
	})
	if err != nil {
		t.Fatalf("list provider models: %v", err)
	}
	if got := adapter.lastListModelsInput.CredentialSecret; got != "sk-1" {
		t.Fatalf("credential secret = %q, want %q", got, "sk-1")
	}
	if len(resp.GetModels()) != 3 {
		t.Fatalf("models len = %d, want 3", len(resp.GetModels()))
	}
	if resp.GetModels()[0].GetId() != "zeta" || resp.GetModels()[1].GetId() != "beta" || resp.GetModels()[2].GetId() != "alpha" {
		t.Fatalf("model order = %#v, want zeta, beta, alpha", resp.GetModels())
	}
}

func TestCreateAgentWithProviderGrantSuccess(t *testing.T) {
	store := newFakeStore()
	store.ProviderGrants["grant-1"] = storage.ProviderGrantRecord{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        "openai",
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		Status:          "active",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	svc := newTestAgentHandlers(store)
	svc.clock = func() time.Time { return time.Date(2026, 2, 15, 22, 57, 0, 0, time.UTC) }
	svc.idGenerator = func() (string, error) { return "agent-1", nil }

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.CreateAgent(ctx, &aiv1.CreateAgentRequest{
		Label:           "narrator",
		Provider:        aiv1.Provider_PROVIDER_OPENAI,
		Model:           "gpt-4o-mini",
		ProviderGrantId: "grant-1",
	})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if resp.GetAgent().GetId() != "agent-1" {
		t.Fatalf("agent id = %q, want %q", resp.GetAgent().GetId(), "agent-1")
	}
	stored := store.Agents["agent-1"]
	if stored.ProviderGrantID != "grant-1" {
		t.Fatalf("provider_grant_id = %q, want %q", stored.ProviderGrantID, "grant-1")
	}
	if stored.CredentialID != "" {
		t.Fatalf("credential_id = %q, want empty", stored.CredentialID)
	}
}

func TestCreateAgentRejectsMultipleAuthReferences(t *testing.T) {
	store := newFakeStore()
	store.Credentials["cred-1"] = storage.CredentialRecord{ID: "cred-1", OwnerUserID: "user-1", Provider: "openai", Label: "A", Status: "active", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	store.ProviderGrants["grant-1"] = storage.ProviderGrantRecord{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        "openai",
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		Status:          "active",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	svc := newTestAgentHandlers(store)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.CreateAgent(ctx, &aiv1.CreateAgentRequest{
		Label:           "narrator",
		Provider:        aiv1.Provider_PROVIDER_OPENAI,
		Model:           "gpt-4o-mini",
		CredentialId:    "cred-1",
		ProviderGrantId: "grant-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListAgentsReturnsOwnerRecords(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:              "agent-1",
		OwnerUserID:     "user-1",
		Label:           "narrator",
		Provider:        "openai",
		Model:           "gpt-4o-mini",
		CredentialID:    "cred-1",
		ProviderGrantID: "",
		Status:          "active",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	store.Agents["agent-2"] = storage.AgentRecord{
		ID:              "agent-2",
		OwnerUserID:     "user-2",
		Label:           "planner",
		Provider:        "openai",
		Model:           "gpt-4o-mini",
		CredentialID:    "cred-2",
		ProviderGrantID: "",
		Status:          "active",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	svc := newAgentHandlersWithStores(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.ListAgents(ctx, &aiv1.ListAgentsRequest{PageSize: 10})
	if err != nil {
		t.Fatalf("list Agents: %v", err)
	}
	if len(resp.GetAgents()) != 1 {
		t.Fatalf("Agents len = %d, want 1", len(resp.GetAgents()))
	}
	if got := resp.GetAgents()[0].GetId(); got != "agent-1" {
		t.Fatalf("agent id = %q, want %q", got, "agent-1")
	}
}

func TestListAgentsIncludesAuthStateAndUsage(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.Credentials["cred-1"] = storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		Label:            "Main",
		SecretCiphertext: "enc:sk-1",
		Status:           "active",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-1",
		Label:        "narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	cfg := newAgentHandlersConfigWithStores(store, store, &fakeSealer{})
	cfg.GameCampaignAIClient = &fakeCampaignAIAuthStateClient{
		usageByAgent: map[string]int32{"agent-1": 2},
	}
	svc := NewAgentHandlers(cfg)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))

	resp, err := svc.ListAgents(ctx, &aiv1.ListAgentsRequest{PageSize: 10})
	if err != nil {
		t.Fatalf("list agents: %v", err)
	}
	if len(resp.GetAgents()) != 1 {
		t.Fatalf("agents len = %d, want 1", len(resp.GetAgents()))
	}
	if got := resp.GetAgents()[0].GetAuthState(); got != aiv1.AgentAuthState_AGENT_AUTH_STATE_READY {
		t.Fatalf("auth state = %v, want ready", got)
	}
	if got := resp.GetAgents()[0].GetActiveCampaignCount(); got != 2 {
		t.Fatalf("active campaign count = %d, want 2", got)
	}
}

func TestValidateCampaignAgentBindingRejectsRevokedCredentialBackedAgent(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.Credentials["cred-1"] = storage.CredentialRecord{
		ID:          "cred-1",
		OwnerUserID: "user-1",
		Provider:    "openai",
		Label:       "Main",
		Status:      "revoked",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-1",
		Label:        "narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	svc := newAgentHandlersWithStores(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))

	_, err := svc.ValidateCampaignAgentBinding(ctx, &aiv1.ValidateCampaignAgentBindingRequest{
		AgentId:    "agent-1",
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestListAccessibleAgentsIncludesOwnedAndApprovedShared(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.Agents["agent-own-1"] = storage.AgentRecord{
		ID:           "agent-own-1",
		OwnerUserID:  "user-1",
		Label:        "owner-agent",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-own-1",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.Agents["agent-shared-1"] = storage.AgentRecord{
		ID:           "agent-shared-1",
		OwnerUserID:  "user-owner",
		Label:        "shared-agent",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-shared-1",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.AccessRequests["request-approved"] = storage.AccessRequestRecord{
		ID:              "request-approved",
		RequesterUserID: "user-1",
		OwnerUserID:     "user-owner",
		AgentID:         "agent-shared-1",
		Scope:           "invoke",
		Status:          "approved",
		CreatedAt:       now.Add(-time.Minute),
		UpdatedAt:       now,
	}

	svc := newAgentHandlersWithStores(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.ListAccessibleAgents(ctx, &aiv1.ListAccessibleAgentsRequest{PageSize: 10})
	if err != nil {
		t.Fatalf("list accessible Agents: %v", err)
	}
	if len(resp.GetAgents()) != 2 {
		t.Fatalf("Agents len = %d, want 2", len(resp.GetAgents()))
	}
	got := []string{resp.GetAgents()[0].GetId(), resp.GetAgents()[1].GetId()}
	want := []string{"agent-own-1", "agent-shared-1"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("agent[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestListAccessibleAgentsExcludesPendingDeniedAndStale(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.Agents["agent-approved"] = storage.AgentRecord{
		ID:           "agent-approved",
		OwnerUserID:  "owner-1",
		Label:        "approved-agent",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.Agents["agent-pending"] = storage.AgentRecord{
		ID:           "agent-pending",
		OwnerUserID:  "owner-1",
		Label:        "pending-agent",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-2",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.Agents["agent-denied"] = storage.AgentRecord{
		ID:           "agent-denied",
		OwnerUserID:  "owner-1",
		Label:        "denied-agent",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-3",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.AccessRequests["request-approved"] = storage.AccessRequestRecord{
		ID:              "request-approved",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-approved",
		Scope:           "invoke",
		Status:          "approved",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	store.AccessRequests["request-pending"] = storage.AccessRequestRecord{
		ID:              "request-pending",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-pending",
		Scope:           "invoke",
		Status:          "pending",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	store.AccessRequests["request-denied"] = storage.AccessRequestRecord{
		ID:              "request-denied",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-denied",
		Scope:           "invoke",
		Status:          "denied",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	store.AccessRequests["request-stale-agent"] = storage.AccessRequestRecord{
		ID:              "request-stale-agent",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-missing",
		Scope:           "invoke",
		Status:          "approved",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	store.AccessRequests["request-wrong-owner"] = storage.AccessRequestRecord{
		ID:              "request-wrong-owner",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-other",
		AgentID:         "agent-approved",
		Scope:           "invoke",
		Status:          "approved",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	svc := newAgentHandlersWithStores(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.ListAccessibleAgents(ctx, &aiv1.ListAccessibleAgentsRequest{PageSize: 10})
	if err != nil {
		t.Fatalf("list accessible Agents: %v", err)
	}
	if len(resp.GetAgents()) != 1 {
		t.Fatalf("Agents len = %d, want 1", len(resp.GetAgents()))
	}
	if got := resp.GetAgents()[0].GetId(); got != "agent-approved" {
		t.Fatalf("agent id = %q, want %q", got, "agent-approved")
	}
}

func TestListAccessibleAgentsRequiresUserID(t *testing.T) {
	svc := newAgentHandlersWithStores(newFakeStore(), newFakeStore(), &fakeSealer{})
	_, err := svc.ListAccessibleAgents(context.Background(), &aiv1.ListAccessibleAgentsRequest{PageSize: 10})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestListAccessibleAgentsPaginatesByAgentID(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.Agents["agent-a"] = storage.AgentRecord{
		ID:           "agent-a",
		OwnerUserID:  "user-1",
		Label:        "agent-a",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-a",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.Agents["agent-b"] = storage.AgentRecord{
		ID:           "agent-b",
		OwnerUserID:  "owner-1",
		Label:        "agent-b",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-b",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.Agents["agent-c"] = storage.AgentRecord{
		ID:           "agent-c",
		OwnerUserID:  "owner-2",
		Label:        "agent-c",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-c",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.AccessRequests["request-b"] = storage.AccessRequestRecord{
		ID:              "request-b",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-b",
		Scope:           "invoke",
		Status:          "approved",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	store.AccessRequests["request-c"] = storage.AccessRequestRecord{
		ID:              "request-c",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-2",
		AgentID:         "agent-c",
		Scope:           "invoke",
		Status:          "approved",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	svc := newAgentHandlersWithStores(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))

	first, err := svc.ListAccessibleAgents(ctx, &aiv1.ListAccessibleAgentsRequest{PageSize: 2})
	if err != nil {
		t.Fatalf("list accessible Agents first page: %v", err)
	}
	if len(first.GetAgents()) != 2 {
		t.Fatalf("first page Agents len = %d, want 2", len(first.GetAgents()))
	}
	if got := first.GetAgents()[0].GetId(); got != "agent-a" {
		t.Fatalf("first page agent[0] = %q, want %q", got, "agent-a")
	}
	if got := first.GetAgents()[1].GetId(); got != "agent-b" {
		t.Fatalf("first page agent[1] = %q, want %q", got, "agent-b")
	}
	if got := first.GetNextPageToken(); got != "agent-b" {
		t.Fatalf("first page next token = %q, want %q", got, "agent-b")
	}

	second, err := svc.ListAccessibleAgents(ctx, &aiv1.ListAccessibleAgentsRequest{
		PageSize:  2,
		PageToken: first.GetNextPageToken(),
	})
	if err != nil {
		t.Fatalf("list accessible Agents second page: %v", err)
	}
	if len(second.GetAgents()) != 1 {
		t.Fatalf("second page Agents len = %d, want 1", len(second.GetAgents()))
	}
	if got := second.GetAgents()[0].GetId(); got != "agent-c" {
		t.Fatalf("second page agent[0] = %q, want %q", got, "agent-c")
	}
	if got := second.GetNextPageToken(); got != "" {
		t.Fatalf("second page next token = %q, want empty", got)
	}
}

func TestGetAccessibleAgentRequiresUserID(t *testing.T) {
	svc := newAgentHandlersWithStores(newFakeStore(), newFakeStore(), &fakeSealer{})
	_, err := svc.GetAccessibleAgent(context.Background(), &aiv1.GetAccessibleAgentRequest{
		AgentId: "agent-1",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestGetAccessibleAgentRequiresAgentID(t *testing.T) {
	svc := newAgentHandlersWithStores(newFakeStore(), newFakeStore(), &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.GetAccessibleAgent(ctx, &aiv1.GetAccessibleAgentRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetAccessibleAgentMissingAgent(t *testing.T) {
	svc := newAgentHandlersWithStores(newFakeStore(), newFakeStore(), &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.GetAccessibleAgent(ctx, &aiv1.GetAccessibleAgentRequest{AgentId: "agent-missing"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetAccessibleAgentOwner(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 2, 30, 0, 0, time.UTC)
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-1",
		Label:        "owner-agent",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	svc := newAgentHandlersWithStores(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.GetAccessibleAgent(ctx, &aiv1.GetAccessibleAgentRequest{AgentId: "agent-1"})
	if err != nil {
		t.Fatalf("get accessible agent: %v", err)
	}
	if got := resp.GetAgent().GetId(); got != "agent-1" {
		t.Fatalf("agent id = %q, want %q", got, "agent-1")
	}
}

func TestGetAccessibleAgentApprovedRequester(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 2, 30, 0, 0, time.UTC)
	store.Agents["agent-shared"] = storage.AgentRecord{
		ID:           "agent-shared",
		OwnerUserID:  "owner-1",
		Label:        "shared-agent",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-owner",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.AccessRequests["request-1"] = storage.AccessRequestRecord{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-shared",
		Scope:           "invoke",
		Status:          "approved",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	svc := newAgentHandlersWithStores(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.GetAccessibleAgent(ctx, &aiv1.GetAccessibleAgentRequest{AgentId: "agent-shared"})
	if err != nil {
		t.Fatalf("get accessible agent: %v", err)
	}
	if got := resp.GetAgent().GetId(); got != "agent-shared" {
		t.Fatalf("agent id = %q, want %q", got, "agent-shared")
	}
}

func TestGetAccessibleAgentPendingRequesterHidden(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 2, 30, 0, 0, time.UTC)
	store.Agents["agent-shared"] = storage.AgentRecord{
		ID:           "agent-shared",
		OwnerUserID:  "owner-1",
		Label:        "shared-agent",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-owner",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.AccessRequests["request-1"] = storage.AccessRequestRecord{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-shared",
		Scope:           "invoke",
		Status:          "pending",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	svc := newAgentHandlersWithStores(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.GetAccessibleAgent(ctx, &aiv1.GetAccessibleAgentRequest{AgentId: "agent-shared"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestUpdateAgentSwitchesCredentialToProviderGrant(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 0, 10, 0, 0, time.UTC)
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:              "agent-1",
		OwnerUserID:     "user-1",
		Label:           "narrator",
		Provider:        "openai",
		Model:           "gpt-4o-mini",
		CredentialID:    "cred-1",
		ProviderGrantID: "",
		Status:          "active",
		CreatedAt:       now.Add(-time.Hour),
		UpdatedAt:       now.Add(-time.Hour),
	}
	store.ProviderGrants["grant-1"] = storage.ProviderGrantRecord{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        "openai",
		TokenCiphertext: `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		Status:          "active",
		CreatedAt:       now.Add(-time.Hour),
		UpdatedAt:       now.Add(-time.Hour),
	}

	svc := newTestAgentHandlers(store)
	svc.clock = func() time.Time { return now }
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.UpdateAgent(ctx, &aiv1.UpdateAgentRequest{
		AgentId:         "agent-1",
		ProviderGrantId: "grant-1",
		Model:           "gpt-4o",
		Instructions:    "Answer as the GM.",
	})
	if err != nil {
		t.Fatalf("update agent: %v", err)
	}

	if got := resp.GetAgent().GetCredentialId(); got != "" {
		t.Fatalf("credential_id = %q, want empty", got)
	}
	if got := resp.GetAgent().GetProviderGrantId(); got != "grant-1" {
		t.Fatalf("provider_grant_id = %q, want %q", got, "grant-1")
	}
	if got := resp.GetAgent().GetModel(); got != "gpt-4o" {
		t.Fatalf("model = %q, want %q", got, "gpt-4o")
	}
	if got := resp.GetAgent().GetInstructions(); got != "Answer as the GM." {
		t.Fatalf("instructions = %q, want %q", got, "Answer as the GM.")
	}
}

func TestUpdateAgentMetadataEditDoesNotRequireLiveModelListing(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 15, 22, 57, 0, 0, time.UTC)
	store.Credentials["cred-1"] = storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		Label:            "Primary",
		SecretCiphertext: "enc:sk-1",
		Status:           "active",
		CreatedAt:        now.Add(-2 * time.Hour),
		UpdatedAt:        now.Add(-2 * time.Hour),
	}
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-1",
		Label:        "narrator",
		Instructions: "Old instructions.",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    now.Add(-time.Hour),
		UpdatedAt:    now.Add(-time.Hour),
	}

	adapter := &fakeProviderInvocationAdapter{listModelsErr: errors.New("provider unavailable")}
	svc := newAgentHandlersWithStores(store, store, &fakeSealer{})
	svc.providerModelAdapters[provider.OpenAI] = adapter
	svc.clock = func() time.Time { return now }

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.UpdateAgent(ctx, &aiv1.UpdateAgentRequest{
		AgentId:      "agent-1",
		Label:        "lead-narrator",
		Instructions: "Keep the session moving.",
	})
	if err != nil {
		t.Fatalf("update agent: %v", err)
	}
	if got := resp.GetAgent().GetLabel(); got != "lead-narrator" {
		t.Fatalf("label = %q, want %q", got, "lead-narrator")
	}
	if got := resp.GetAgent().GetInstructions(); got != "Keep the session moving." {
		t.Fatalf("instructions = %q, want %q", got, "Keep the session moving.")
	}
	if got := adapter.lastListModelsInput.CredentialSecret; got != "" {
		t.Fatalf("model listing should not run for metadata-only edit, got credential secret %q", got)
	}
}

func TestDeleteAgentRemovesOwnedRecord(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:              "agent-1",
		OwnerUserID:     "user-1",
		Label:           "narrator",
		Provider:        "openai",
		Model:           "gpt-4o-mini",
		CredentialID:    "cred-1",
		ProviderGrantID: "",
		Status:          "active",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	svc := newAgentHandlersWithStores(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	if _, err := svc.DeleteAgent(ctx, &aiv1.DeleteAgentRequest{AgentId: "agent-1"}); err != nil {
		t.Fatalf("delete agent: %v", err)
	}
	if _, ok := store.Agents["agent-1"]; ok {
		t.Fatal("agent should be deleted")
	}
}
