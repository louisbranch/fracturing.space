package providerconnect

import (
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
)

func TestCreatePendingNormalizesAndInitializesSession(t *testing.T) {
	now := time.Date(2026, 3, 23, 16, 0, 0, 0, time.UTC)

	session, err := CreatePending(CreateInput{
		ID:                     "session-1",
		OwnerUserID:            " user-1 ",
		Provider:               provider.Provider("openai"),
		RequestedScopes:        []string{"responses.read", "responses.read", " "},
		StateHash:              "hash:state",
		CodeVerifierCiphertext: "enc:verifier",
		CreatedAt:              now,
		ExpiresAt:              now.Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("CreatePending: %v", err)
	}
	if session.Status != StatusPending {
		t.Fatalf("session.Status = %q, want %q", session.Status, StatusPending)
	}
	if session.Provider != provider.OpenAI {
		t.Fatalf("session.Provider = %q, want %q", session.Provider, provider.OpenAI)
	}
	if len(session.RequestedScopes) != 1 || session.RequestedScopes[0] != "responses.read" {
		t.Fatalf("session.RequestedScopes = %#v", session.RequestedScopes)
	}
	if !session.CreatedAt.Equal(now) || !session.UpdatedAt.Equal(now) {
		t.Fatalf("session timestamps = %v / %v, want %v", session.CreatedAt, session.UpdatedAt, now)
	}
}

func TestCompleteMarksPendingSessionCompleted(t *testing.T) {
	now := time.Date(2026, 3, 23, 16, 5, 0, 0, time.UTC)
	session := Session{
		ID:          "session-1",
		OwnerUserID: "user-1",
		Provider:    provider.OpenAI,
		Status:      StatusPending,
		CreatedAt:   now.Add(-time.Minute),
		UpdatedAt:   now.Add(-time.Minute),
		ExpiresAt:   now.Add(9 * time.Minute),
	}

	completed, err := Complete(session, now)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if completed.Status != StatusCompleted {
		t.Fatalf("completed.Status = %q, want %q", completed.Status, StatusCompleted)
	}
	if completed.CompletedAt == nil || !completed.CompletedAt.Equal(now) {
		t.Fatalf("completed.CompletedAt = %v, want %v", completed.CompletedAt, now)
	}
	if !completed.UpdatedAt.Equal(now) {
		t.Fatalf("completed.UpdatedAt = %v, want %v", completed.UpdatedAt, now)
	}
}
