package ai

import (
	"context"
	"testing"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func TestStartProviderConnectRequiresUserID(t *testing.T) {
	svc := newProviderGrantHandlersWithStores(t, newFakeStore(), newFakeStore(), &fakeSealer{})
	_, err := svc.StartProviderConnect(context.Background(), &aiv1.StartProviderConnectRequest{
		Provider: aiv1.Provider_PROVIDER_OPENAI,
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestListProviderGrantsRejectsInvalidProviderFilter(t *testing.T) {
	svc := newProviderGrantHandlersWithStores(t, newFakeStore(), newFakeStore(), &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))

	_, err := svc.ListProviderGrants(ctx, &aiv1.ListProviderGrantsRequest{
		PageSize: 10,
		Provider: aiv1.Provider(99),
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListProviderGrantsRejectsInvalidStatusFilter(t *testing.T) {
	svc := newProviderGrantHandlersWithStores(t, newFakeStore(), newFakeStore(), &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))

	_, err := svc.ListProviderGrants(ctx, &aiv1.ListProviderGrantsRequest{
		PageSize: 10,
		Status:   aiv1.ProviderGrantStatus(99),
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestRevokeProviderGrantRequiresID(t *testing.T) {
	svc := newProviderGrantHandlersWithStores(t, newFakeStore(), newFakeStore(), &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))

	_, err := svc.RevokeProviderGrant(ctx, &aiv1.RevokeProviderGrantRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}
