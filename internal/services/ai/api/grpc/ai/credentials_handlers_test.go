package ai

import (
	"context"
	"errors"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func TestCreateCredentialRequiresUserID(t *testing.T) {
	svc := newCredentialHandlersWithStores(newFakeStore(), newFakeStore(), &fakeSealer{})
	_, err := svc.CreateCredential(context.Background(), &aiv1.CreateCredentialRequest{
		Provider: aiv1.Provider_PROVIDER_OPENAI,
		Label:    "Main",
		Secret:   "sk-1",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestCreateCredentialEncryptsSecret(t *testing.T) {
	store := newFakeStore()
	svc := newCredentialHandlersWithStores(store, store, &fakeSealer{})
	svc.clock = func() time.Time { return time.Date(2026, 2, 15, 22, 50, 0, 0, time.UTC) }
	svc.idGenerator = func() (string, error) { return "cred-1", nil }

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.CreateCredential(ctx, &aiv1.CreateCredentialRequest{
		Provider: aiv1.Provider_PROVIDER_OPENAI,
		Label:    "Main",
		Secret:   "sk-1",
	})
	if err != nil {
		t.Fatalf("create credential: %v", err)
	}
	if resp.GetCredential().GetId() != "cred-1" {
		t.Fatalf("credential id = %q, want %q", resp.GetCredential().GetId(), "cred-1")
	}

	stored := store.Credentials["cred-1"]
	if stored.SecretCiphertext != "enc:sk-1" {
		t.Fatalf("stored ciphertext = %q, want %q", stored.SecretCiphertext, "enc:sk-1")
	}
}

func TestListCredentialsReturnsOwnerRecords(t *testing.T) {
	store := newFakeStore()
	store.Credentials["cred-1"] = storage.CredentialRecord{ID: "cred-1", OwnerUserID: "user-1", Provider: "openai", Label: "A", Status: "active", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	store.Credentials["cred-2"] = storage.CredentialRecord{ID: "cred-2", OwnerUserID: "user-2", Provider: "openai", Label: "B", Status: "active", CreatedAt: time.Now(), UpdatedAt: time.Now()}

	svc := newCredentialHandlersWithStores(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.ListCredentials(ctx, &aiv1.ListCredentialsRequest{PageSize: 10})
	if err != nil {
		t.Fatalf("list Credentials: %v", err)
	}
	if len(resp.GetCredentials()) != 1 {
		t.Fatalf("Credentials len = %d, want 1", len(resp.GetCredentials()))
	}
	if resp.GetCredentials()[0].GetId() != "cred-1" {
		t.Fatalf("credential id = %q, want %q", resp.GetCredentials()[0].GetId(), "cred-1")
	}
}

func TestRevokeCredential(t *testing.T) {
	store := newFakeStore()
	store.Credentials["cred-1"] = storage.CredentialRecord{ID: "cred-1", OwnerUserID: "user-1", Provider: "openai", Label: "A", Status: "active", CreatedAt: time.Now(), UpdatedAt: time.Now()}

	svc := newCredentialHandlersWithStores(store, store, &fakeSealer{})
	svc.clock = func() time.Time { return time.Date(2026, 2, 15, 22, 55, 0, 0, time.UTC) }
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))

	resp, err := svc.RevokeCredential(ctx, &aiv1.RevokeCredentialRequest{CredentialId: "cred-1"})
	if err != nil {
		t.Fatalf("revoke credential: %v", err)
	}
	if resp.GetCredential().GetStatus() != aiv1.CredentialStatus_CREDENTIAL_STATUS_REVOKED {
		t.Fatalf("status = %v, want revoked", resp.GetCredential().GetStatus())
	}
}

func TestRevokeCredentialFailsWhenReferencedAgentIsBoundToActiveCampaigns(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.Credentials["cred-1"] = storage.CredentialRecord{ID: "cred-1", OwnerUserID: "user-1", Provider: "openai", Label: "A", Status: "active", CreatedAt: now, UpdatedAt: now}
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

	svc := newCredentialHandlersWithStores(store, store, &fakeSealer{})
	svc.usageGuard.gameCampaignAIClient = &fakeCampaignAIAuthStateClient{
		usageByAgent: map[string]int32{"agent-1": 1},
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))

	_, err := svc.RevokeCredential(ctx, &aiv1.RevokeCredentialRequest{CredentialId: "cred-1"})
	assertStatusCode(t, err, codes.FailedPrecondition)

	if got := store.Credentials["cred-1"].Status; got != "active" {
		t.Fatalf("credential status = %q, want active", got)
	}
}

func TestCreateCredentialSealError(t *testing.T) {
	svc := newCredentialHandlersWithStores(newFakeStore(), newFakeStore(), &fakeSealer{SealErr: errors.New("seal fail")})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.CreateCredential(ctx, &aiv1.CreateCredentialRequest{
		Provider: aiv1.Provider_PROVIDER_OPENAI,
		Label:    "Main",
		Secret:   "sk-1",
	})
	assertStatusCode(t, err, codes.Internal)
}
