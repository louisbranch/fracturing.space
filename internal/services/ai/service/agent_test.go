package service

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/accessrequest"
	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/credential"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provideroauth"
	"github.com/louisbranch/fracturing.space/internal/services/ai/secret"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"github.com/louisbranch/fracturing.space/internal/test/mock/aifakes"
)

type fakeModelAdapter struct {
	listModelsErr    error
	listModelsResult []provider.Model
	lastInput        provider.ListModelsInput
}

func (f *fakeModelAdapter) ListModels(_ context.Context, input provider.ListModelsInput) ([]provider.Model, error) {
	f.lastInput = input
	if f.listModelsErr != nil {
		return nil, f.listModelsErr
	}
	return f.listModelsResult, nil
}

type accessibleAgentStore struct {
	*aifakes.AgentStore
	*aifakes.AccessRequestStore
}

func newAccessibleAgentStore() *accessibleAgentStore {
	return &accessibleAgentStore{
		AgentStore:         aifakes.NewAgentStore(),
		AccessRequestStore: aifakes.NewAccessRequestStore(),
	}
}

func (s *accessibleAgentStore) ListAccessibleAgents(_ context.Context, userID string, pageSize int, pageToken string) (agent.Page, error) {
	seen := make(map[string]struct{})
	items := make([]agent.Agent, 0)

	for _, rec := range s.Agents {
		if rec.OwnerUserID == userID {
			items = append(items, rec)
			seen[rec.ID] = struct{}{}
		}
	}

	for _, ar := range s.AccessRequests {
		if ar.RequesterUserID != userID || ar.Scope != accessrequest.ScopeInvoke || ar.Status != accessrequest.StatusApproved {
			continue
		}
		if _, ok := seen[ar.AgentID]; ok {
			continue
		}
		rec, ok := s.Agents[ar.AgentID]
		if !ok || rec.OwnerUserID != ar.OwnerUserID {
			continue
		}
		items = append(items, rec)
		seen[rec.ID] = struct{}{}
	}

	sort.Slice(items, func(i int, j int) bool { return items[i].ID < items[j].ID })

	start := 0
	if pageToken != "" {
		start = len(items)
		for idx, rec := range items {
			if rec.ID > pageToken {
				start = idx
				break
			}
		}
	}
	if start >= len(items) {
		return agent.Page{Agents: []agent.Agent{}}, nil
	}
	items = items[start:]

	if pageSize > 0 && len(items) > pageSize {
		nextToken := items[pageSize-1].ID
		return agent.Page{Agents: items[:pageSize], NextPageToken: nextToken}, nil
	}
	return agent.Page{Agents: items}, nil
}

func TestAgentServiceCreateWithCredentialSuccess(t *testing.T) {
	credentialStore := aifakes.NewCredentialStore()
	agentStore := newAccessibleAgentStore()
	now := time.Date(2026, 3, 23, 19, 10, 0, 0, time.UTC)
	credentialStore.Credentials["cred-1"] = credential.Credential{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		Label:            "main",
		SecretCiphertext: "enc:sk-1",
		Status:           credential.StatusActive,
		CreatedAt:        now.Add(-time.Hour),
		UpdatedAt:        now.Add(-time.Hour),
	}

	modelAdapter := &fakeModelAdapter{
		listModelsResult: []provider.Model{{ID: "gpt-4o-mini"}},
	}
	svc := mustNewAgentService(t, agentServiceDeps{
		credentialStore: credentialStore,
		agentStore:      agentStore,
		modelAdapters: map[provider.Provider]provider.ModelAdapter{
			provider.OpenAI: modelAdapter,
		},
		clock:       func() time.Time { return now },
		idGenerator: func() (string, error) { return "agent-1", nil },
	})

	record, err := svc.Create(context.Background(), CreateAgentInput{
		OwnerUserID:   "user-1",
		Label:         "narrator",
		Instructions:  "Keep the scene moving.",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: agent.CredentialAuthReference("cred-1"),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if record.ID != "agent-1" {
		t.Fatalf("record.ID = %q, want %q", record.ID, "agent-1")
	}
	if record.Instructions != "Keep the scene moving." {
		t.Fatalf("record.Instructions = %q, want %q", record.Instructions, "Keep the scene moving.")
	}
	if record.AuthReference.CredentialID() != "cred-1" {
		t.Fatalf("record.AuthReference.CredentialID() = %q, want %q", record.AuthReference.CredentialID(), "cred-1")
	}
	if modelAdapter.lastInput.AuthToken != "sk-1" {
		t.Fatalf("modelAdapter.lastInput.AuthToken = %q, want %q", modelAdapter.lastInput.AuthToken, "sk-1")
	}
}

func TestAgentServiceCreateWithProviderGrantSuccess(t *testing.T) {
	agentStore := newAccessibleAgentStore()
	providerGrantStore := aifakes.NewProviderGrantStore()
	now := time.Date(2026, 3, 23, 19, 15, 0, 0, time.UTC)
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

	modelAdapter := &fakeModelAdapter{
		listModelsResult: []provider.Model{{ID: "gpt-4o-mini"}},
	}
	svc := mustNewAgentService(t, agentServiceDeps{
		agentStore:         agentStore,
		providerGrantStore: providerGrantStore,
		modelAdapters: map[provider.Provider]provider.ModelAdapter{
			provider.OpenAI: modelAdapter,
		},
		clock:       func() time.Time { return now },
		idGenerator: func() (string, error) { return "agent-1", nil },
	})

	record, err := svc.Create(context.Background(), CreateAgentInput{
		OwnerUserID:   "user-1",
		Label:         "narrator",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: agent.ProviderGrantAuthReference("grant-1"),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if record.AuthReference.ProviderGrantID() != "grant-1" {
		t.Fatalf("record.AuthReference.ProviderGrantID() = %q, want %q", record.AuthReference.ProviderGrantID(), "grant-1")
	}
	if record.AuthReference.CredentialID() != "" {
		t.Fatalf("record.AuthReference.CredentialID() = %q, want empty", record.AuthReference.CredentialID())
	}
	if modelAdapter.lastInput.AuthToken != "at-1" {
		t.Fatalf("modelAdapter.lastInput.AuthToken = %q, want %q", modelAdapter.lastInput.AuthToken, "at-1")
	}
}

func TestAgentServiceCreateRejectsInvalidAuthReference(t *testing.T) {
	credentialStore := aifakes.NewCredentialStore()
	providerGrantStore := aifakes.NewProviderGrantStore()
	now := time.Date(2026, 3, 23, 19, 20, 0, 0, time.UTC)
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
	providerGrantStore.ProviderGrants["grant-1"] = providergrant.ProviderGrant{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        provider.OpenAI,
		TokenCiphertext: `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		Status:          providergrant.StatusActive,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	svc := mustNewAgentService(t, agentServiceDeps{
		credentialStore:    credentialStore,
		providerGrantStore: providerGrantStore,
		agentStore:         newAccessibleAgentStore(),
	})

	_, err := svc.Create(context.Background(), CreateAgentInput{
		OwnerUserID:   "user-1",
		Label:         "narrator",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: agent.AuthReference{Kind: "other", ID: "cred-1"},
	})
	if got := ErrorKindOf(err); got != ErrKindInvalidArgument {
		t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindInvalidArgument)
	}
}

func TestAgentServiceListByOwner(t *testing.T) {
	agentStore := newAccessibleAgentStore()
	now := time.Date(2026, 3, 23, 19, 22, 0, 0, time.UTC)
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
	agentStore.Agents["agent-2"] = agent.Agent{
		ID:            "agent-2",
		OwnerUserID:   "user-2",
		Label:         "oracle",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o",
		AuthReference: agent.CredentialAuthReference("cred-2"),
		Status:        agent.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	svc := mustNewAgentService(t, agentServiceDeps{
		agentStore: agentStore,
	})

	page, err := svc.List(context.Background(), "user-1", 10, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Agents) != 1 || page.Agents[0].ID != "agent-1" {
		t.Fatalf("page.Agents = %+v, want agent-1 only", page.Agents)
	}
}

func TestAgentServiceCreateRejectsRevokedCredential(t *testing.T) {
	credentialStore := aifakes.NewCredentialStore()
	now := time.Date(2026, 3, 23, 19, 25, 0, 0, time.UTC)
	credentialStore.Credentials["cred-1"] = credential.Credential{
		ID:          "cred-1",
		OwnerUserID: "user-1",
		Provider:    provider.OpenAI,
		Label:       "main",
		Status:      credential.StatusRevoked,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	svc := mustNewAgentService(t, agentServiceDeps{
		credentialStore: credentialStore,
		agentStore:      newAccessibleAgentStore(),
	})

	_, err := svc.Create(context.Background(), CreateAgentInput{
		OwnerUserID:   "user-1",
		Label:         "narrator",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: agent.CredentialAuthReference("cred-1"),
	})
	if got := ErrorKindOf(err); got != ErrKindFailedPrecondition {
		t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindFailedPrecondition)
	}
}

func TestAgentServiceListProviderModelsReturnsAlphabeticalIDs(t *testing.T) {
	credentialStore := aifakes.NewCredentialStore()
	now := time.Date(2026, 3, 23, 19, 30, 0, 0, time.UTC)
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
			{ID: "alpha"},
			{ID: "zeta"},
			{ID: "beta"},
			{ID: ""},
		},
	}
	svc := mustNewAgentService(t, agentServiceDeps{
		credentialStore: credentialStore,
		agentStore:      newAccessibleAgentStore(),
		modelAdapters: map[provider.Provider]provider.ModelAdapter{
			provider.OpenAI: modelAdapter,
		},
		clock: func() time.Time { return now },
	})

	models, err := svc.ListProviderModels(context.Background(), ListProviderModelsInput{
		OwnerUserID:   "user-1",
		Provider:      provider.OpenAI,
		AuthReference: agent.CredentialAuthReference("cred-1"),
	})
	if err != nil {
		t.Fatalf("ListProviderModels: %v", err)
	}
	if modelAdapter.lastInput.AuthToken != "sk-1" {
		t.Fatalf("modelAdapter.lastInput.AuthToken = %q, want %q", modelAdapter.lastInput.AuthToken, "sk-1")
	}
	if len(models) != 3 {
		t.Fatalf("len(models) = %d, want 3", len(models))
	}
	if models[0].ID != "alpha" || models[1].ID != "beta" || models[2].ID != "zeta" {
		t.Fatalf("models order = %#v, want alpha, beta, zeta", models)
	}
}

func TestAgentServiceListAccessibleIncludesOwnedAndApprovedShared(t *testing.T) {
	agentStore := newAccessibleAgentStore()
	now := time.Date(2026, 3, 23, 19, 35, 0, 0, time.UTC)
	agentStore.Agents["agent-own-1"] = agent.Agent{
		ID:            "agent-own-1",
		OwnerUserID:   "user-1",
		Label:         "owner-agent",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: agent.CredentialAuthReference("cred-own-1"),
		Status:        agent.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	agentStore.Agents["agent-shared-1"] = agent.Agent{
		ID:            "agent-shared-1",
		OwnerUserID:   "user-owner",
		Label:         "shared-agent",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: agent.CredentialAuthReference("cred-shared-1"),
		Status:        agent.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	agentStore.AccessRequests["request-approved"] = accessrequest.AccessRequest{
		ID:              "request-approved",
		RequesterUserID: "user-1",
		OwnerUserID:     "user-owner",
		AgentID:         "agent-shared-1",
		Scope:           accessrequest.ScopeInvoke,
		Status:          accessrequest.StatusApproved,
		CreatedAt:       now.Add(-time.Minute),
		UpdatedAt:       now,
	}

	svc := mustNewAgentService(t, agentServiceDeps{
		agentStore:         agentStore,
		accessRequestStore: agentStore,
	})

	page, err := svc.ListAccessible(context.Background(), "user-1", 10, "")
	if err != nil {
		t.Fatalf("ListAccessible: %v", err)
	}
	if len(page.Agents) != 2 {
		t.Fatalf("len(page.Agents) = %d, want 2", len(page.Agents))
	}
	if page.Agents[0].ID != "agent-own-1" || page.Agents[1].ID != "agent-shared-1" {
		t.Fatalf("page.Agents = %#v, want agent-own-1 then agent-shared-1", page.Agents)
	}
}

func TestAgentServiceListAccessibleExcludesPendingDeniedAndStale(t *testing.T) {
	agentStore := newAccessibleAgentStore()
	now := time.Date(2026, 3, 23, 19, 40, 0, 0, time.UTC)
	agentStore.Agents["agent-approved"] = agent.Agent{
		ID:            "agent-approved",
		OwnerUserID:   "owner-1",
		Label:         "approved-agent",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: agent.CredentialAuthReference("cred-1"),
		Status:        agent.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	agentStore.Agents["agent-pending"] = agent.Agent{
		ID:            "agent-pending",
		OwnerUserID:   "owner-1",
		Label:         "pending-agent",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: agent.CredentialAuthReference("cred-2"),
		Status:        agent.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	agentStore.Agents["agent-denied"] = agent.Agent{
		ID:            "agent-denied",
		OwnerUserID:   "owner-1",
		Label:         "denied-agent",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: agent.CredentialAuthReference("cred-3"),
		Status:        agent.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	agentStore.AccessRequests["request-approved"] = accessrequest.AccessRequest{
		ID:              "request-approved",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-approved",
		Scope:           accessrequest.ScopeInvoke,
		Status:          accessrequest.StatusApproved,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	agentStore.AccessRequests["request-pending"] = accessrequest.AccessRequest{
		ID:              "request-pending",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-pending",
		Scope:           accessrequest.ScopeInvoke,
		Status:          accessrequest.StatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	agentStore.AccessRequests["request-denied"] = accessrequest.AccessRequest{
		ID:              "request-denied",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-denied",
		Scope:           accessrequest.ScopeInvoke,
		Status:          accessrequest.StatusDenied,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	agentStore.AccessRequests["request-stale-agent"] = accessrequest.AccessRequest{
		ID:              "request-stale-agent",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-missing",
		Scope:           accessrequest.ScopeInvoke,
		Status:          accessrequest.StatusApproved,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	svc := mustNewAgentService(t, agentServiceDeps{
		agentStore:         agentStore,
		accessRequestStore: agentStore,
	})

	page, err := svc.ListAccessible(context.Background(), "user-1", 10, "")
	if err != nil {
		t.Fatalf("ListAccessible: %v", err)
	}
	if len(page.Agents) != 1 {
		t.Fatalf("len(page.Agents) = %d, want 1", len(page.Agents))
	}
	if page.Agents[0].ID != "agent-approved" {
		t.Fatalf("page.Agents[0].ID = %q, want %q", page.Agents[0].ID, "agent-approved")
	}
}

func TestAgentServiceListAccessiblePaginatesByAgentID(t *testing.T) {
	agentStore := newAccessibleAgentStore()
	now := time.Date(2026, 3, 23, 19, 45, 0, 0, time.UTC)
	agentStore.Agents["agent-a"] = agent.Agent{ID: "agent-a", OwnerUserID: "user-1", Label: "agent-a", Provider: provider.OpenAI, Model: "gpt-4o-mini", AuthReference: agent.CredentialAuthReference("cred-a"), Status: agent.StatusActive, CreatedAt: now, UpdatedAt: now}
	agentStore.Agents["agent-b"] = agent.Agent{ID: "agent-b", OwnerUserID: "owner-1", Label: "agent-b", Provider: provider.OpenAI, Model: "gpt-4o-mini", AuthReference: agent.CredentialAuthReference("cred-b"), Status: agent.StatusActive, CreatedAt: now, UpdatedAt: now}
	agentStore.Agents["agent-c"] = agent.Agent{ID: "agent-c", OwnerUserID: "owner-2", Label: "agent-c", Provider: provider.OpenAI, Model: "gpt-4o-mini", AuthReference: agent.CredentialAuthReference("cred-c"), Status: agent.StatusActive, CreatedAt: now, UpdatedAt: now}
	agentStore.AccessRequests["request-b"] = accessrequest.AccessRequest{ID: "request-b", RequesterUserID: "user-1", OwnerUserID: "owner-1", AgentID: "agent-b", Scope: accessrequest.ScopeInvoke, Status: accessrequest.StatusApproved, CreatedAt: now, UpdatedAt: now}
	agentStore.AccessRequests["request-c"] = accessrequest.AccessRequest{ID: "request-c", RequesterUserID: "user-1", OwnerUserID: "owner-2", AgentID: "agent-c", Scope: accessrequest.ScopeInvoke, Status: accessrequest.StatusApproved, CreatedAt: now, UpdatedAt: now}

	svc := mustNewAgentService(t, agentServiceDeps{
		agentStore:         agentStore,
		accessRequestStore: agentStore,
	})

	first, err := svc.ListAccessible(context.Background(), "user-1", 2, "")
	if err != nil {
		t.Fatalf("ListAccessible first: %v", err)
	}
	if len(first.Agents) != 2 {
		t.Fatalf("len(first.Agents) = %d, want 2", len(first.Agents))
	}
	if first.Agents[0].ID != "agent-a" || first.Agents[1].ID != "agent-b" {
		t.Fatalf("first.Agents = %#v, want agent-a then agent-b", first.Agents)
	}
	if first.NextPageToken != "agent-b" {
		t.Fatalf("first.NextPageToken = %q, want %q", first.NextPageToken, "agent-b")
	}

	second, err := svc.ListAccessible(context.Background(), "user-1", 2, first.NextPageToken)
	if err != nil {
		t.Fatalf("ListAccessible second: %v", err)
	}
	if len(second.Agents) != 1 {
		t.Fatalf("len(second.Agents) = %d, want 1", len(second.Agents))
	}
	if second.Agents[0].ID != "agent-c" {
		t.Fatalf("second.Agents[0].ID = %q, want %q", second.Agents[0].ID, "agent-c")
	}
	if second.NextPageToken != "" {
		t.Fatalf("second.NextPageToken = %q, want empty", second.NextPageToken)
	}
}

func TestAgentServiceGetAccessibleOwnerAndApprovedRequester(t *testing.T) {
	now := time.Date(2026, 3, 23, 19, 50, 0, 0, time.UTC)

	t.Run("owner", func(t *testing.T) {
		agentStore := newAccessibleAgentStore()
		agentStore.Agents["agent-1"] = agent.Agent{
			ID:            "agent-1",
			OwnerUserID:   "user-1",
			Label:         "owner-agent",
			Provider:      provider.OpenAI,
			Model:         "gpt-4o-mini",
			AuthReference: agent.CredentialAuthReference("cred-1"),
			Status:        agent.StatusActive,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		svc := mustNewAgentService(t, agentServiceDeps{
			agentStore:         agentStore,
			accessRequestStore: agentStore,
		})

		record, err := svc.GetAccessible(context.Background(), "user-1", "agent-1")
		if err != nil {
			t.Fatalf("GetAccessible: %v", err)
		}
		if record.ID != "agent-1" {
			t.Fatalf("record.ID = %q, want %q", record.ID, "agent-1")
		}
	})

	t.Run("approved requester", func(t *testing.T) {
		agentStore := newAccessibleAgentStore()
		agentStore.Agents["agent-shared"] = agent.Agent{
			ID:            "agent-shared",
			OwnerUserID:   "owner-1",
			Label:         "shared-agent",
			Provider:      provider.OpenAI,
			Model:         "gpt-4o-mini",
			AuthReference: agent.CredentialAuthReference("cred-owner"),
			Status:        agent.StatusActive,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		agentStore.AccessRequests["request-1"] = accessrequest.AccessRequest{
			ID:              "request-1",
			RequesterUserID: "user-1",
			OwnerUserID:     "owner-1",
			AgentID:         "agent-shared",
			Scope:           accessrequest.ScopeInvoke,
			Status:          accessrequest.StatusApproved,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		svc := mustNewAgentService(t, agentServiceDeps{
			agentStore:         agentStore,
			accessRequestStore: agentStore,
		})

		record, err := svc.GetAccessible(context.Background(), "user-1", "agent-shared")
		if err != nil {
			t.Fatalf("GetAccessible: %v", err)
		}
		if record.ID != "agent-shared" {
			t.Fatalf("record.ID = %q, want %q", record.ID, "agent-shared")
		}
	})
}

func TestAgentServiceGetAccessibleHiddenWithoutApprovedAccess(t *testing.T) {
	agentStore := newAccessibleAgentStore()
	now := time.Date(2026, 3, 23, 19, 55, 0, 0, time.UTC)
	agentStore.Agents["agent-shared"] = agent.Agent{
		ID:            "agent-shared",
		OwnerUserID:   "owner-1",
		Label:         "shared-agent",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: agent.CredentialAuthReference("cred-owner"),
		Status:        agent.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	agentStore.AccessRequests["request-1"] = accessrequest.AccessRequest{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-shared",
		Scope:           accessrequest.ScopeInvoke,
		Status:          accessrequest.StatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	svc := mustNewAgentService(t, agentServiceDeps{
		agentStore:         agentStore,
		accessRequestStore: agentStore,
	})

	_, err := svc.GetAccessible(context.Background(), "user-1", "agent-shared")
	if got := ErrorKindOf(err); got != ErrKindNotFound {
		t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindNotFound)
	}
}

func TestAgentServiceValidateCampaignAgentBindingRejectsRevokedCredentialBackedAgent(t *testing.T) {
	credentialStore := aifakes.NewCredentialStore()
	agentStore := newAccessibleAgentStore()
	now := time.Date(2026, 3, 23, 20, 0, 0, 0, time.UTC)
	credentialStore.Credentials["cred-1"] = credential.Credential{
		ID:          "cred-1",
		OwnerUserID: "user-1",
		Provider:    provider.OpenAI,
		Label:       "main",
		Status:      credential.StatusRevoked,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
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

	svc := mustNewAgentService(t, agentServiceDeps{
		credentialStore: credentialStore,
		agentStore:      agentStore,
	})

	_, err := svc.ValidateCampaignAgentBinding(context.Background(), "user-1", "agent-1")
	if got := ErrorKindOf(err); got != ErrKindFailedPrecondition {
		t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindFailedPrecondition)
	}
}

func TestAgentServiceUpdateSwitchesCredentialToProviderGrant(t *testing.T) {
	agentStore := newAccessibleAgentStore()
	providerGrantStore := aifakes.NewProviderGrantStore()
	now := time.Date(2026, 3, 23, 20, 5, 0, 0, time.UTC)
	agentStore.Agents["agent-1"] = agent.Agent{
		ID:            "agent-1",
		OwnerUserID:   "user-1",
		Label:         "narrator",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: agent.CredentialAuthReference("cred-1"),
		Status:        agent.StatusActive,
		CreatedAt:     now.Add(-time.Hour),
		UpdatedAt:     now.Add(-time.Hour),
	}
	providerGrantStore.ProviderGrants["grant-1"] = providergrant.ProviderGrant{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        provider.OpenAI,
		TokenCiphertext: `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		Status:          providergrant.StatusActive,
		CreatedAt:       now.Add(-time.Hour),
		UpdatedAt:       now.Add(-time.Hour),
	}

	modelAdapter := &fakeModelAdapter{
		listModelsResult: []provider.Model{{ID: "gpt-4o"}},
	}
	svc := mustNewAgentService(t, agentServiceDeps{
		agentStore:         agentStore,
		providerGrantStore: providerGrantStore,
		modelAdapters: map[provider.Provider]provider.ModelAdapter{
			provider.OpenAI: modelAdapter,
		},
		clock: func() time.Time { return now },
	})

	record, err := svc.Update(context.Background(), UpdateAgentInput{
		OwnerUserID:   "user-1",
		AgentID:       "agent-1",
		AuthReference: agent.ProviderGrantAuthReference("grant-1"),
		Model:         "gpt-4o",
		Instructions:  "Answer as the GM.",
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if record.AuthReference.ProviderGrantID() != "grant-1" {
		t.Fatalf("record.AuthReference.ProviderGrantID() = %q, want %q", record.AuthReference.ProviderGrantID(), "grant-1")
	}
	if record.AuthReference.CredentialID() != "" {
		t.Fatalf("record.AuthReference.CredentialID() = %q, want empty", record.AuthReference.CredentialID())
	}
	if record.Model != "gpt-4o" {
		t.Fatalf("record.Model = %q, want %q", record.Model, "gpt-4o")
	}
	if record.Instructions != "Answer as the GM." {
		t.Fatalf("record.Instructions = %q, want %q", record.Instructions, "Answer as the GM.")
	}
	if modelAdapter.lastInput.AuthToken != "at-1" {
		t.Fatalf("modelAdapter.lastInput.AuthToken = %q, want %q", modelAdapter.lastInput.AuthToken, "at-1")
	}
}

func TestAgentServiceUpdateMetadataEditDoesNotRequireLiveModelListing(t *testing.T) {
	credentialStore := aifakes.NewCredentialStore()
	agentStore := newAccessibleAgentStore()
	now := time.Date(2026, 3, 23, 20, 10, 0, 0, time.UTC)
	credentialStore.Credentials["cred-1"] = credential.Credential{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		Label:            "primary",
		SecretCiphertext: "enc:sk-1",
		Status:           credential.StatusActive,
		CreatedAt:        now.Add(-2 * time.Hour),
		UpdatedAt:        now.Add(-2 * time.Hour),
	}
	agentStore.Agents["agent-1"] = agent.Agent{
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

	modelAdapter := &fakeModelAdapter{listModelsErr: errors.New("provider unavailable")}
	svc := mustNewAgentService(t, agentServiceDeps{
		credentialStore: credentialStore,
		agentStore:      agentStore,
		modelAdapters: map[provider.Provider]provider.ModelAdapter{
			provider.OpenAI: modelAdapter,
		},
		clock: func() time.Time { return now },
	})

	record, err := svc.Update(context.Background(), UpdateAgentInput{
		OwnerUserID:  "user-1",
		AgentID:      "agent-1",
		Label:        "lead-narrator",
		Instructions: "Keep the session moving.",
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if record.Label != "lead-narrator" {
		t.Fatalf("record.Label = %q, want %q", record.Label, "lead-narrator")
	}
	if record.Instructions != "Keep the session moving." {
		t.Fatalf("record.Instructions = %q, want %q", record.Instructions, "Keep the session moving.")
	}
	if modelAdapter.lastInput.AuthToken != "" {
		t.Fatalf("modelAdapter.lastInput.AuthToken = %q, want empty", modelAdapter.lastInput.AuthToken)
	}
}

func TestAgentServiceDeleteRemovesOwnedRecord(t *testing.T) {
	agentStore := newAccessibleAgentStore()
	now := time.Date(2026, 3, 23, 20, 15, 0, 0, time.UTC)
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

	svc := mustNewAgentService(t, agentServiceDeps{
		agentStore: agentStore,
	})

	if err := svc.Delete(context.Background(), "user-1", "agent-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, ok := agentStore.Agents["agent-1"]; ok {
		t.Fatal("agent should be deleted")
	}
}

func TestAgentServiceGetAuthStateAndActiveCampaignCount(t *testing.T) {
	credentialStore := aifakes.NewCredentialStore()
	providerGrantStore := aifakes.NewProviderGrantStore()
	agentStore := newAccessibleAgentStore()
	now := time.Date(2026, 3, 23, 20, 20, 0, 0, time.UTC)
	credentialStore.Credentials["cred-ready"] = credential.Credential{
		ID:               "cred-ready",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		Label:            "ready",
		SecretCiphertext: "enc:sk-ready",
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

	svc := mustNewAgentService(t, agentServiceDeps{
		credentialStore:    credentialStore,
		providerGrantStore: providerGrantStore,
		agentStore:         agentStore,
		campaignUsageReader: &fakeCampaignUsageReader{
			usageByAgent: map[string]int32{"agent-ready": 2},
		},
	})

	if got := svc.GetAuthState(context.Background(), agent.Agent{
		ID:            "agent-ready",
		OwnerUserID:   "user-1",
		Provider:      provider.OpenAI,
		AuthReference: agent.CredentialAuthReference("cred-ready"),
	}); got != AgentAuthStateReady {
		t.Fatalf("GetAuthState(ready) = %v, want %v", got, AgentAuthStateReady)
	}
	if got := svc.GetAuthState(context.Background(), agent.Agent{
		ID:            "agent-revoked",
		OwnerUserID:   "user-1",
		Provider:      provider.OpenAI,
		AuthReference: agent.CredentialAuthReference("cred-revoked"),
	}); got != AgentAuthStateRevoked {
		t.Fatalf("GetAuthState(revoked credential) = %v, want %v", got, AgentAuthStateRevoked)
	}
	if got := svc.GetAuthState(context.Background(), agent.Agent{
		ID:            "agent-grant-revoked",
		OwnerUserID:   "user-1",
		Provider:      provider.OpenAI,
		AuthReference: agent.ProviderGrantAuthReference("grant-revoked"),
	}); got != AgentAuthStateRevoked {
		t.Fatalf("GetAuthState(revoked grant) = %v, want %v", got, AgentAuthStateRevoked)
	}
	if got := svc.GetAuthState(context.Background(), agent.Agent{
		ID:            "agent-missing",
		OwnerUserID:   "user-1",
		Provider:      provider.OpenAI,
		AuthReference: agent.CredentialAuthReference("cred-missing"),
	}); got != AgentAuthStateUnavailable {
		t.Fatalf("GetAuthState(missing) = %v, want %v", got, AgentAuthStateUnavailable)
	}

	count, err := svc.GetActiveCampaignCount(context.Background(), "agent-ready")
	if err != nil {
		t.Fatalf("GetActiveCampaignCount: %v", err)
	}
	if count != 2 {
		t.Fatalf("count = %d, want %d", count, 2)
	}
}

type agentServiceDeps struct {
	credentialStore     storage.CredentialStore
	agentStore          storage.AgentStore
	providerGrantStore  storage.ProviderGrantStore
	accessRequestStore  storage.AccessRequestStore
	modelAdapters       map[provider.Provider]provider.ModelAdapter
	oauthAdapters       map[provider.Provider]provideroauth.Adapter
	campaignUsageReader CampaignUsageReader
	sealer              secret.Sealer
	clock               Clock
	idGenerator         IDGenerator
}

func mustNewAgentService(t *testing.T, deps agentServiceDeps) *AgentService {
	t.Helper()

	agentStore := deps.agentStore
	if agentStore == nil {
		agentStore = newAccessibleAgentStore()
	}
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
	authMaterialResolver := NewAuthMaterialResolver(AuthMaterialResolverConfig{
		CredentialStore:      deps.credentialStore,
		Sealer:               sealer,
		ProviderGrantRuntime: providerGrantRuntime,
	})
	authReferencePolicy, err := NewAuthReferencePolicy(AuthReferencePolicyConfig{
		CredentialStore:      deps.credentialStore,
		ProviderGrantStore:   deps.providerGrantStore,
		ProviderRegistry:     mustProviderRegistryForTests(t, deps.oauthAdapters, nil, deps.modelAdapters, nil),
		AuthMaterialResolver: authMaterialResolver,
	})
	if err != nil {
		t.Fatalf("NewAuthReferencePolicy: %v", err)
	}
	accessibleAgentResolver := NewAccessibleAgentResolver(agentStore, deps.accessRequestStore)
	agentBindingUsageReader := NewAgentBindingUsageReader(deps.campaignUsageReader)
	authReferenceUsageReader := NewAuthReferenceUsageReader(agentStore, agentBindingUsageReader)
	usagePolicy := NewUsagePolicy(UsagePolicyConfig{
		AgentBindingUsageReader:  agentBindingUsageReader,
		AuthReferenceUsageReader: authReferenceUsageReader,
	})

	svc, err := NewAgentService(AgentServiceConfig{
		AgentStore:              agentStore,
		AuthReferencePolicy:     authReferencePolicy,
		AccessibleAgentResolver: accessibleAgentResolver,
		UsagePolicy:             usagePolicy,
		AgentBindingUsageReader: agentBindingUsageReader,
		Clock:                   deps.clock,
		IDGenerator:             deps.idGenerator,
	})
	if err != nil {
		t.Fatalf("NewAgentService: %v", err)
	}
	return svc
}
