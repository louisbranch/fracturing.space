package ai

import (
	"context"
	"errors"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/credential"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func TestCreateCredentialRequiresUserID(t *testing.T) {
	svc := newCredentialHandlersWithStores(t, newFakeStore(), newFakeStore(), &fakeSealer{})
	_, err := svc.CreateCredential(context.Background(), &aiv1.CreateCredentialRequest{
		Provider: aiv1.Provider_PROVIDER_OPENAI,
		Label:    "Main",
		Secret:   "sk-1",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestCreateCredentialEncryptsSecret(t *testing.T) {
	store := newFakeStore()
	h := newCredentialHandlersWithOpts(t, store, store, &fakeSealer{},
		func() time.Time { return time.Date(2026, 2, 15, 22, 50, 0, 0, time.UTC) },
		func() (string, error) { return "cred-1", nil },
		nil,
	)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := h.CreateCredential(ctx, &aiv1.CreateCredentialRequest{
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
	store.Credentials["cred-1"] = credential.Credential{ID: "cred-1", OwnerUserID: "user-1", Provider: provider.OpenAI, Label: "A", Status: credential.StatusActive, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	store.Credentials["cred-2"] = credential.Credential{ID: "cred-2", OwnerUserID: "user-2", Provider: provider.OpenAI, Label: "B", Status: credential.StatusActive, CreatedAt: time.Now(), UpdatedAt: time.Now()}

	h := newCredentialHandlersWithStores(t, store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := h.ListCredentials(ctx, &aiv1.ListCredentialsRequest{PageSize: 10})
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
	store.Credentials["cred-1"] = credential.Credential{ID: "cred-1", OwnerUserID: "user-1", Provider: provider.OpenAI, Label: "A", Status: credential.StatusActive, CreatedAt: time.Now(), UpdatedAt: time.Now()}

	h := newCredentialHandlersWithOpts(t, store, store, &fakeSealer{},
		func() time.Time { return time.Date(2026, 2, 15, 22, 55, 0, 0, time.UTC) },
		nil, nil,
	)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))

	resp, err := h.RevokeCredential(ctx, &aiv1.RevokeCredentialRequest{CredentialId: "cred-1"})
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
	store.Credentials["cred-1"] = credential.Credential{ID: "cred-1", OwnerUserID: "user-1", Provider: provider.OpenAI, Label: "A", Status: credential.StatusActive, CreatedAt: now, UpdatedAt: now}
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

	usageGuard := service.NewUsageGuard(store, &fakeCampaignAIAuthStateClient{
		usageByAgent: map[string]int32{"agent-1": 1},
	})
	h := newCredentialHandlersWithOpts(t, store, store, &fakeSealer{}, nil, nil, usageGuard)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))

	_, err := h.RevokeCredential(ctx, &aiv1.RevokeCredentialRequest{CredentialId: "cred-1"})
	assertStatusCode(t, err, codes.FailedPrecondition)

	if got := store.Credentials["cred-1"].Status; got != credential.StatusActive {
		t.Fatalf("credential status = %q, want active", got)
	}
}

func TestCreateCredentialSealError(t *testing.T) {
	h := newCredentialHandlersWithStores(t, newFakeStore(), newFakeStore(), &fakeSealer{SealErr: errors.New("seal fail")})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := h.CreateCredential(ctx, &aiv1.CreateCredentialRequest{
		Provider: aiv1.Provider_PROVIDER_OPENAI,
		Label:    "Main",
		Secret:   "sk-1",
	})
	assertStatusCode(t, err, codes.Internal)
}
