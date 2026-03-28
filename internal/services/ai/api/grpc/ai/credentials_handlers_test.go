package ai

import (
	"context"
	"errors"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/credential"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/test/grpcassert"
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
	grpcassert.StatusCode(t, err, codes.PermissionDenied)
}

func TestCreateCredentialSealError(t *testing.T) {
	h := newCredentialHandlersWithStores(t, newFakeStore(), newFakeStore(), &fakeSealer{SealErr: errors.New("seal fail")})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := h.CreateCredential(ctx, &aiv1.CreateCredentialRequest{
		Provider: aiv1.Provider_PROVIDER_OPENAI,
		Label:    "Main",
		Secret:   "sk-1",
	})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestCreateCredentialRejectsUnavailableProvider(t *testing.T) {
	h := newCredentialHandlersWithStores(t, newFakeStore(), newFakeStore(), &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := h.CreateCredential(ctx, &aiv1.CreateCredentialRequest{
		Provider: aiv1.Provider_PROVIDER_ANTHROPIC,
		Label:    "Main",
		Secret:   "sk-1",
	})
	grpcassert.StatusCode(t, err, codes.FailedPrecondition)
}

func TestListCredentialsReturnsOwnedPage(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 3, 23, 21, 0, 0, 0, time.UTC)
	store.Credentials["cred-1"] = credential.Credential{
		ID:          "cred-1",
		OwnerUserID: "user-1",
		Provider:    provider.OpenAI,
		Label:       "Main",
		Status:      credential.StatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	store.Credentials["cred-2"] = credential.Credential{
		ID:          "cred-2",
		OwnerUserID: "user-2",
		Provider:    provider.Anthropic,
		Label:       "Claude",
		Status:      credential.StatusRevoked,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	h := newCredentialHandlersWithStores(t, store, newFakeStore(), &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := h.ListCredentials(ctx, &aiv1.ListCredentialsRequest{PageSize: 10})
	if err != nil {
		t.Fatalf("ListCredentials() error = %v", err)
	}
	if len(resp.GetCredentials()) != 1 || resp.GetCredentials()[0].GetId() != "cred-1" {
		t.Fatalf("credentials = %+v, want cred-1 only", resp.GetCredentials())
	}
}

func TestRevokeCredentialRequiresIDAndReturnsUpdatedRecord(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 3, 23, 21, 5, 0, 0, time.UTC)
	store.Credentials["cred-1"] = credential.Credential{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		Label:            "Main",
		Status:           credential.StatusActive,
		SecretCiphertext: "enc:sk-1",
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	h := newCredentialHandlersWithStores(t, store, newFakeStore(), &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))

	_, err := h.RevokeCredential(ctx, &aiv1.RevokeCredentialRequest{})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)

	resp, err := h.RevokeCredential(ctx, &aiv1.RevokeCredentialRequest{CredentialId: "cred-1"})
	if err != nil {
		t.Fatalf("RevokeCredential() error = %v", err)
	}
	if resp.GetCredential().GetStatus() != aiv1.CredentialStatus_CREDENTIAL_STATUS_REVOKED {
		t.Fatalf("status = %v, want revoked", resp.GetCredential().GetStatus())
	}
}
