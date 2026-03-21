package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

func TestPutGetProviderConnectSessionRoundTrip(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 23, 40, 0, 0, time.UTC)
	expiresAt := now.Add(10 * time.Minute)

	input := storage.ProviderConnectSessionRecord{
		ID:                     "session-1",
		OwnerUserID:            "user-1",
		Provider:               "openai",
		RequestedScopes:        []string{"responses.read"},
		StateHash:              "hash:state",
		CodeVerifierCiphertext: "enc:verifier",
		Status:                 "pending",
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

	if err := store.PutProviderConnectSession(context.Background(), storage.ProviderConnectSessionRecord{
		ID:                     "session-1",
		OwnerUserID:            "user-1",
		Provider:               "openai",
		RequestedScopes:        []string{"responses.read"},
		StateHash:              "hash:state",
		CodeVerifierCiphertext: "enc:verifier",
		Status:                 "pending",
		CreatedAt:              now,
		UpdatedAt:              now,
		ExpiresAt:              expiresAt,
	}); err != nil {
		t.Fatalf("put provider connect session: %v", err)
	}

	completedAt := now.Add(time.Minute)
	if err := store.CompleteProviderConnectSession(context.Background(), "user-1", "session-1", completedAt); err != nil {
		t.Fatalf("complete provider connect session: %v", err)
	}

	got, err := store.GetProviderConnectSession(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("get provider connect session: %v", err)
	}
	if got.Status != "completed" {
		t.Fatalf("status = %q, want %q", got.Status, "completed")
	}
	if got.CompletedAt == nil || !got.CompletedAt.Equal(completedAt) {
		t.Fatalf("completed_at = %v, want %v", got.CompletedAt, completedAt)
	}
}
