package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providerconnect"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

func TestPutGetProviderConnectSessionRoundTrip(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 23, 40, 0, 0, time.UTC)
	expiresAt := now.Add(10 * time.Minute)

	input := providerconnect.Session{
		ID:                     "session-1",
		OwnerUserID:            "user-1",
		Provider:               provider.OpenAI,
		RequestedScopes:        []string{"responses.read"},
		StateHash:              "hash:state",
		CodeVerifierCiphertext: "enc:verifier",
		Status:                 providerconnect.StatusPending,
		CreatedAt:              now,
		UpdatedAt:              now,
		ExpiresAt:              expiresAt,
	}
	if err := store.PutProviderConnectSession(context.Background(), input); err != nil {
		t.Fatalf("put provider connect session: %v", err)
	}

	got, err := store.GetProviderConnectSession(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("get provider connect session: %v", err)
	}
	if got.ID != input.ID || got.OwnerUserID != input.OwnerUserID || got.StateHash != input.StateHash {
		t.Fatalf("unexpected provider connect session: %+v", got)
	}
}

func TestCompleteProviderConnectSession(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 23, 40, 0, 0, time.UTC)
	expiresAt := now.Add(10 * time.Minute)

	if err := store.PutProviderConnectSession(context.Background(), providerconnect.Session{
		ID:                     "session-1",
		OwnerUserID:            "user-1",
		Provider:               provider.OpenAI,
		RequestedScopes:        []string{"responses.read"},
		StateHash:              "hash:state",
		CodeVerifierCiphertext: "enc:verifier",
		Status:                 providerconnect.StatusPending,
		CreatedAt:              now,
		UpdatedAt:              now,
		ExpiresAt:              expiresAt,
	}); err != nil {
		t.Fatalf("put provider connect session: %v", err)
	}

	completedAt := now.Add(time.Minute)
	session, err := store.GetProviderConnectSession(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("get provider connect session: %v", err)
	}
	completed, err := providerconnect.Complete(session, completedAt)
	if err != nil {
		t.Fatalf("complete provider connect session model: %v", err)
	}
	if err := store.CompleteProviderConnectSession(context.Background(), completed); err != nil {
		t.Fatalf("complete provider connect session: %v", err)
	}

	got, err := store.GetProviderConnectSession(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("get provider connect session: %v", err)
	}
	if got.Status != providerconnect.StatusCompleted {
		t.Fatalf("status = %q, want %q", got.Status, providerconnect.StatusCompleted)
	}
	if got.CompletedAt == nil || !got.CompletedAt.Equal(completedAt) {
		t.Fatalf("completed_at = %v, want %v", got.CompletedAt, completedAt)
	}
}

func TestFinishProviderConnectStoresGrantAndCompletesSessionAtomically(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 23, 40, 0, 0, time.UTC)
	expiresAt := now.Add(10 * time.Minute)

	session := providerconnect.Session{
		ID:                     "session-1",
		OwnerUserID:            "user-1",
		Provider:               provider.OpenAI,
		RequestedScopes:        []string{"responses.read"},
		StateHash:              "hash:state",
		CodeVerifierCiphertext: "enc:verifier",
		Status:                 providerconnect.StatusPending,
		CreatedAt:              now,
		UpdatedAt:              now,
		ExpiresAt:              expiresAt,
	}
	if err := store.PutProviderConnectSession(context.Background(), session); err != nil {
		t.Fatalf("put provider connect session: %v", err)
	}

	completed, err := providerconnect.Complete(session, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("complete provider connect session model: %v", err)
	}
	grant := providergrant.ProviderGrant{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        provider.OpenAI,
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: "enc:token",
		Status:          providergrant.StatusActive,
		CreatedAt:       now.Add(time.Minute),
		UpdatedAt:       now.Add(time.Minute),
	}

	if err := store.FinishProviderConnect(context.Background(), grant, completed); err != nil {
		t.Fatalf("finish provider connect: %v", err)
	}

	storedGrant, err := store.GetProviderGrant(context.Background(), grant.ID)
	if err != nil {
		t.Fatalf("get provider grant: %v", err)
	}
	if storedGrant.ID != grant.ID {
		t.Fatalf("stored grant id = %q, want %q", storedGrant.ID, grant.ID)
	}

	storedSession, err := store.GetProviderConnectSession(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("get provider connect session: %v", err)
	}
	if storedSession.Status != providerconnect.StatusCompleted {
		t.Fatalf("stored session status = %q, want %q", storedSession.Status, providerconnect.StatusCompleted)
	}
}

func TestFinishProviderConnectRollsBackGrantWhenSessionCompletionFails(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 23, 40, 0, 0, time.UTC)

	grant := providergrant.ProviderGrant{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        provider.OpenAI,
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: "enc:token",
		Status:          providergrant.StatusActive,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	completed := providerconnect.Session{
		ID:          "missing-session",
		OwnerUserID: "user-1",
		Provider:    provider.OpenAI,
		Status:      providerconnect.StatusCompleted,
		UpdatedAt:   now,
		CompletedAt: ptrTime(now),
	}

	err := store.FinishProviderConnect(context.Background(), grant, completed)
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("errors.Is(err, storage.ErrNotFound) = false, err = %v", err)
	}

	_, err = store.GetProviderGrant(context.Background(), grant.ID)
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("grant persisted after failed finish: err = %v", err)
	}
}
