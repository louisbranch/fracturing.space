package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/accessrequest"
	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

func TestPutGetAgentRoundTrip(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 22, 45, 0, 0, time.UTC)

	input := agent.Agent{
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
	if err := store.PutAgent(context.Background(), input); err != nil {
		t.Fatalf("put agent: %v", err)
	}

	got, err := store.GetAgent(context.Background(), "agent-1")
	if err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if got.ID != input.ID || got.OwnerUserID != input.OwnerUserID || got.AuthReference.CredentialID() != "cred-1" {
		t.Fatalf("unexpected agent: %+v", got)
	}
}

func TestPutGetAgentRoundTripWithProviderGrant(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 22, 45, 0, 0, time.UTC)

	input := agent.Agent{
		ID:            "agent-grant-1",
		OwnerUserID:   "user-1",
		Label:         "narrator",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: agent.ProviderGrantAuthReference("grant-1"),
		Status:        agent.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := store.PutAgent(context.Background(), input); err != nil {
		t.Fatalf("put agent: %v", err)
	}

	got, err := store.GetAgent(context.Background(), "agent-grant-1")
	if err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if got.AuthReference.ProviderGrantID() != "grant-1" {
		t.Fatalf("provider_grant_id = %q, want %q", got.AuthReference.ProviderGrantID(), "grant-1")
	}
	if got.AuthReference.CredentialID() != "" {
		t.Fatalf("credential_id = %q, want empty", got.AuthReference.CredentialID())
	}
}

func TestPutAgentRequiresExactlyOneAuthReference(t *testing.T) {
	store := openTempStore(t)
	now := time.Now().UTC()

	base := agent.Agent{
		ID:          "agent-1",
		OwnerUserID: "user-1",
		Label:       "narrator",
		Provider:    provider.OpenAI,
		Model:       "gpt-4o-mini",
		Status:      agent.StatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	missing := base
	if err := store.PutAgent(context.Background(), missing); err == nil {
		t.Fatal("expected error for missing auth reference")
	}
}

func TestDeleteAgent(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 22, 45, 0, 0, time.UTC)

	if err := store.PutAgent(context.Background(), agent.Agent{
		ID:            "agent-1",
		OwnerUserID:   "user-1",
		Label:         "narrator",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: agent.CredentialAuthReference("cred-1"),
		Status:        agent.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}); err != nil {
		t.Fatalf("put agent: %v", err)
	}

	if err := store.DeleteAgent(context.Background(), "user-1", "agent-1"); err != nil {
		t.Fatalf("delete agent: %v", err)
	}

	_, err := store.GetAgent(context.Background(), "agent-1")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestPutAgentRejectsDuplicateLabelForOwner(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 1, 15, 0, 0, time.UTC)

	if err := store.PutAgent(context.Background(), agent.Agent{
		ID:            "agent-1",
		OwnerUserID:   "user-1",
		Label:         "narrator",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: agent.CredentialAuthReference("cred-1"),
		Status:        agent.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}); err != nil {
		t.Fatalf("put first agent: %v", err)
	}

	err := store.PutAgent(context.Background(), agent.Agent{
		ID:            "agent-2",
		OwnerUserID:   "user-1",
		Label:         " narrator ",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o",
		AuthReference: agent.CredentialAuthReference("cred-2"),
		Status:        agent.StatusActive,
		CreatedAt:     now.Add(time.Minute),
		UpdatedAt:     now.Add(time.Minute),
	})
	if !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("duplicate label error = %v, want storage.ErrConflict", err)
	}
}

func TestPutAgentAllowsSameLabelAcrossOwners(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 1, 20, 0, 0, time.UTC)

	for _, a := range []agent.Agent{
		{
			ID:            "agent-1",
			OwnerUserID:   "user-1",
			Label:         "narrator",
			Provider:      provider.OpenAI,
			Model:         "gpt-4o-mini",
			AuthReference: agent.CredentialAuthReference("cred-1"),
			Status:        agent.StatusActive,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "agent-2",
			OwnerUserID:   "user-2",
			Label:         " narrator ",
			Provider:      provider.OpenAI,
			Model:         "gpt-4o-mini",
			AuthReference: agent.CredentialAuthReference("cred-2"),
			Status:        agent.StatusActive,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	} {
		if err := store.PutAgent(context.Background(), a); err != nil {
			t.Fatalf("put agent %s: %v", a.ID, err)
		}
	}
}

func TestListAgentsByOwner(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 1, 30, 0, 0, time.UTC)

	records := []agent.Agent{
		{
			ID:            "agent-1",
			OwnerUserID:   "user-1",
			Label:         "narrator",
			Provider:      provider.OpenAI,
			Model:         "gpt-4o-mini",
			AuthReference: agent.CredentialAuthReference("cred-1"),
			Status:        agent.StatusActive,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "agent-2",
			OwnerUserID:   "user-1",
			Label:         "oracle",
			Provider:      provider.OpenAI,
			Model:         "gpt-4o",
			AuthReference: agent.CredentialAuthReference("cred-2"),
			Status:        agent.StatusActive,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "agent-3",
			OwnerUserID:   "user-2",
			Label:         "other",
			Provider:      provider.OpenAI,
			Model:         "gpt-4o",
			AuthReference: agent.CredentialAuthReference("cred-3"),
			Status:        agent.StatusActive,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}
	for _, record := range records {
		if err := store.PutAgent(context.Background(), record); err != nil {
			t.Fatalf("put agent %s: %v", record.ID, err)
		}
	}

	page, err := store.ListAgentsByOwner(context.Background(), "user-1", 1, "")
	if err != nil {
		t.Fatalf("ListAgentsByOwner(first page) error = %v", err)
	}
	if len(page.Agents) != 1 || page.Agents[0].ID != "agent-1" {
		t.Fatalf("first page = %+v, want agent-1 only", page.Agents)
	}
	if page.NextPageToken != "agent-1" {
		t.Fatalf("NextPageToken = %q, want %q", page.NextPageToken, "agent-1")
	}

	page, err = store.ListAgentsByOwner(context.Background(), "user-1", 2, page.NextPageToken)
	if err != nil {
		t.Fatalf("ListAgentsByOwner(second page) error = %v", err)
	}
	if len(page.Agents) != 1 || page.Agents[0].ID != "agent-2" {
		t.Fatalf("second page = %+v, want agent-2 only", page.Agents)
	}
	if page.NextPageToken != "" {
		t.Fatalf("NextPageToken = %q, want empty", page.NextPageToken)
	}
}

func TestListAccessibleAgents(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 1, 40, 0, 0, time.UTC)

	agents := []agent.Agent{
		{
			ID:            "agent-1",
			OwnerUserID:   "user-1",
			Label:         "owned",
			Provider:      provider.OpenAI,
			Model:         "gpt-4o-mini",
			AuthReference: agent.CredentialAuthReference("cred-1"),
			Status:        agent.StatusActive,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "agent-2",
			OwnerUserID:   "owner-2",
			Label:         "shared",
			Provider:      provider.OpenAI,
			Model:         "gpt-4o",
			AuthReference: agent.CredentialAuthReference("cred-2"),
			Status:        agent.StatusActive,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}
	for _, record := range agents {
		if err := store.PutAgent(context.Background(), record); err != nil {
			t.Fatalf("put agent %s: %v", record.ID, err)
		}
	}

	if err := store.PutAccessRequest(context.Background(), accessrequest.AccessRequest{
		ID:              "ar-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-2",
		AgentID:         "agent-2",
		Scope:           accessrequest.ScopeInvoke,
		Status:          accessrequest.StatusApproved,
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("put access request: %v", err)
	}

	page, err := store.ListAccessibleAgents(context.Background(), "user-1", 10, "")
	if err != nil {
		t.Fatalf("ListAccessibleAgents() error = %v", err)
	}
	if len(page.Agents) != 2 {
		t.Fatalf("len(page.Agents) = %d, want 2", len(page.Agents))
	}
	if page.Agents[0].ID != "agent-1" || page.Agents[1].ID != "agent-2" {
		t.Fatalf("accessible agents = %+v, want agent-1 then agent-2", page.Agents)
	}
}
