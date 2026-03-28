package ai

import (
	"context"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"github.com/louisbranch/fracturing.space/internal/test/grpcassert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func TestNewProviderGrantHandlersRequiresService(t *testing.T) {
	_, err := NewProviderGrantHandlers(ProviderGrantHandlersConfig{})
	if err == nil {
		t.Fatal("expected missing provider grant service error")
	}
}

func TestStartProviderConnectRequiresUserID(t *testing.T) {
	svc := newProviderGrantHandlersWithStores(t, newFakeStore(), newFakeStore(), &fakeSealer{})
	_, err := svc.StartProviderConnect(context.Background(), &aiv1.StartProviderConnectRequest{
		Provider: aiv1.Provider_PROVIDER_OPENAI,
	})
	grpcassert.StatusCode(t, err, codes.PermissionDenied)
}

func TestStartProviderConnectReturnsTransportResponse(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)
	idValues := []string{"session-1", "state-1"}
	handlers := newProviderGrantHandlersWithOpts(t, store, store, &fakeSealer{}, providerGrantTestOpts{
		clock: func() time.Time { return now },
		idGenerator: func() (string, error) {
			value := idValues[0]
			idValues = idValues[1:]
			return value, nil
		},
		codeVerifierGenerator: func() (string, error) {
			return "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~", nil
		},
	})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))

	resp, err := handlers.StartProviderConnect(ctx, &aiv1.StartProviderConnectRequest{
		Provider:        aiv1.Provider_PROVIDER_OPENAI,
		RequestedScopes: []string{"responses.read"},
	})
	if err != nil {
		t.Fatalf("StartProviderConnect() error = %v", err)
	}
	if resp.GetConnectSessionId() != "session-1" {
		t.Fatalf("connect_session_id = %q, want %q", resp.GetConnectSessionId(), "session-1")
	}
	if resp.GetState() != "state-1" {
		t.Fatalf("state = %q, want %q", resp.GetState(), "state-1")
	}
	if resp.GetAuthorizationUrl() == "" {
		t.Fatal("expected authorization url")
	}
	if got := store.ConnectSessions["session-1"].OwnerUserID; got != "user-1" {
		t.Fatalf("stored owner_user_id = %q, want %q", got, "user-1")
	}
}

func TestFinishProviderConnectReturnsProviderGrant(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)
	idValues := []string{"session-1", "state-1", "grant-1"}
	handlers := newProviderGrantHandlersWithOpts(t, store, store, &fakeSealer{}, providerGrantTestOpts{
		clock: func() time.Time { return now },
		idGenerator: func() (string, error) {
			value := idValues[0]
			idValues = idValues[1:]
			return value, nil
		},
		codeVerifierGenerator: func() (string, error) {
			return "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~", nil
		},
	})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))

	started, err := handlers.StartProviderConnect(ctx, &aiv1.StartProviderConnectRequest{
		Provider:        aiv1.Provider_PROVIDER_OPENAI,
		RequestedScopes: []string{"responses.read"},
	})
	if err != nil {
		t.Fatalf("StartProviderConnect() error = %v", err)
	}

	resp, err := handlers.FinishProviderConnect(ctx, &aiv1.FinishProviderConnectRequest{
		ConnectSessionId:  started.GetConnectSessionId(),
		State:             started.GetState(),
		AuthorizationCode: "auth-code-1",
	})
	if err != nil {
		t.Fatalf("FinishProviderConnect() error = %v", err)
	}
	if resp.GetProviderGrant() == nil {
		t.Fatal("expected provider grant response")
	}
	if resp.GetProviderGrant().GetId() != "grant-1" {
		t.Fatalf("provider grant id = %q, want %q", resp.GetProviderGrant().GetId(), "grant-1")
	}
	if got := store.ProviderGrants["grant-1"].Status; got != providergrant.StatusActive {
		t.Fatalf("stored status = %q, want %q", got, providergrant.StatusActive)
	}
}

func TestListProviderGrantsRejectsInvalidProviderFilter(t *testing.T) {
	svc := newProviderGrantHandlersWithStores(t, newFakeStore(), newFakeStore(), &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))

	_, err := svc.ListProviderGrants(ctx, &aiv1.ListProviderGrantsRequest{
		PageSize: 10,
		Provider: aiv1.Provider(99),
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestListProviderGrantsRejectsInvalidStatusFilter(t *testing.T) {
	svc := newProviderGrantHandlersWithStores(t, newFakeStore(), newFakeStore(), &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))

	_, err := svc.ListProviderGrants(ctx, &aiv1.ListProviderGrantsRequest{
		PageSize: 10,
		Status:   aiv1.ProviderGrantStatus(99),
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestListProviderGrantsReturnsFilteredPage(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)
	store.ProviderGrants["grant-1"] = providergrant.ProviderGrant{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        provider.OpenAI,
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: "enc:1",
		Status:          providergrant.StatusActive,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	store.ProviderGrants["grant-2"] = providergrant.ProviderGrant{
		ID:              "grant-2",
		OwnerUserID:     "user-1",
		Provider:        provider.OpenAI,
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: "enc:2",
		Status:          providergrant.StatusRevoked,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	svc := newProviderGrantHandlersWithStores(t, store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))

	resp, err := svc.ListProviderGrants(ctx, &aiv1.ListProviderGrantsRequest{
		PageSize: 10,
		Provider: aiv1.Provider_PROVIDER_OPENAI,
		Status:   aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_ACTIVE,
	})
	if err != nil {
		t.Fatalf("ListProviderGrants() error = %v", err)
	}
	if len(resp.GetProviderGrants()) != 1 {
		t.Fatalf("provider_grants len = %d, want 1", len(resp.GetProviderGrants()))
	}
	if resp.GetProviderGrants()[0].GetId() != "grant-1" {
		t.Fatalf("provider_grants[0].id = %q, want %q", resp.GetProviderGrants()[0].GetId(), "grant-1")
	}
}

func TestRevokeProviderGrantRequiresID(t *testing.T) {
	svc := newProviderGrantHandlersWithStores(t, newFakeStore(), newFakeStore(), &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))

	_, err := svc.RevokeProviderGrant(ctx, &aiv1.RevokeProviderGrantRequest{})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestRevokeProviderGrantReturnsRevokedGrant(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)
	store.ProviderGrants["grant-1"] = providergrant.ProviderGrant{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        provider.OpenAI,
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		Status:          providergrant.StatusActive,
		CreatedAt:       now.Add(-time.Hour),
		UpdatedAt:       now.Add(-time.Hour),
	}
	svc := newProviderGrantHandlersWithOpts(t, store, store, &fakeSealer{}, providerGrantTestOpts{
		clock: func() time.Time { return now },
	})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))

	resp, err := svc.RevokeProviderGrant(ctx, &aiv1.RevokeProviderGrantRequest{ProviderGrantId: "grant-1"})
	if err != nil {
		t.Fatalf("RevokeProviderGrant() error = %v", err)
	}
	if resp.GetProviderGrant() == nil {
		t.Fatal("expected provider grant response")
	}
	if got := resp.GetProviderGrant().GetStatus(); got != aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_REVOKED {
		t.Fatalf("status = %v, want %v", got, aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_REVOKED)
	}
	if got := store.ProviderGrants["grant-1"].Status; got != providergrant.StatusRevoked {
		t.Fatalf("stored status = %q, want %q", got, providergrant.StatusRevoked)
	}
}
