package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

func TestPutGetAgentRoundTrip(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 22, 45, 0, 0, time.UTC)

	input := storage.AgentRecord{
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
	if err := store.PutAgent(context.Background(), input); err != nil {
		t.Fatalf("put agent: %v", err)
	}

	got, err := store.GetAgent(context.Background(), "agent-1")
	if err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if got.ID != input.ID || got.OwnerUserID != input.OwnerUserID || got.CredentialID != input.CredentialID {
		t.Fatalf("unexpected agent: %+v", got)
	}
}

func TestPutGetAgentRoundTripWithProviderGrant(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 22, 45, 0, 0, time.UTC)

	input := storage.AgentRecord{
		ID:              "agent-grant-1",
		OwnerUserID:     "user-1",
		Label:           "narrator",
		Provider:        "openai",
		Model:           "gpt-4o-mini",
		CredentialID:    "",
		ProviderGrantID: "grant-1",
		Status:          "active",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := store.PutAgent(context.Background(), input); err != nil {
		t.Fatalf("put agent: %v", err)
	}

	got, err := store.GetAgent(context.Background(), "agent-grant-1")
	if err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if got.ProviderGrantID != "grant-1" {
		t.Fatalf("provider_grant_id = %q, want %q", got.ProviderGrantID, "grant-1")
	}
	if got.CredentialID != "" {
		t.Fatalf("credential_id = %q, want empty", got.CredentialID)
	}
}

func TestPutAgentRequiresExactlyOneAuthReference(t *testing.T) {
	store := openTempStore(t)
	now := time.Now().UTC()

	base := storage.AgentRecord{
		ID:          "agent-1",
		OwnerUserID: "user-1",
		Label:       "narrator",
		Provider:    "openai",
		Model:       "gpt-4o-mini",
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	missing := base
	if err := store.PutAgent(context.Background(), missing); err == nil {
		t.Fatal("expected error for missing auth reference")
	}

	both := base
	both.CredentialID = "cred-1"
	both.ProviderGrantID = "grant-1"
	if err := store.PutAgent(context.Background(), both); err == nil {
		t.Fatal("expected error for multiple auth references")
	}
}

func TestDeleteAgent(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 22, 45, 0, 0, time.UTC)

	if err := store.PutAgent(context.Background(), storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-1",
		Label:        "narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
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

	if err := store.PutAgent(context.Background(), storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-1",
		Label:        "narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}); err != nil {
		t.Fatalf("put first agent: %v", err)
	}

	err := store.PutAgent(context.Background(), storage.AgentRecord{
		ID:           "agent-2",
		OwnerUserID:  "user-1",
		Label:        " narrator ",
		Provider:     "openai",
		Model:        "gpt-4o",
		CredentialID: "cred-2",
		Status:       "active",
		CreatedAt:    now.Add(time.Minute),
		UpdatedAt:    now.Add(time.Minute),
	})
	if !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("duplicate label error = %v, want storage.ErrConflict", err)
	}
}

func TestPutAgentAllowsSameLabelAcrossOwners(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 1, 20, 0, 0, time.UTC)

	for _, rec := range []storage.AgentRecord{
		{
			ID:           "agent-1",
			OwnerUserID:  "user-1",
			Label:        "narrator",
			Provider:     "openai",
			Model:        "gpt-4o-mini",
			CredentialID: "cred-1",
			Status:       "active",
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			ID:           "agent-2",
			OwnerUserID:  "user-2",
			Label:        " narrator ",
			Provider:     "openai",
			Model:        "gpt-4o-mini",
			CredentialID: "cred-2",
			Status:       "active",
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	} {
		if err := store.PutAgent(context.Background(), rec); err != nil {
			t.Fatalf("put agent %s: %v", rec.ID, err)
		}
	}
}
