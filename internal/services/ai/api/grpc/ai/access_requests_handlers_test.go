package ai

import (
	"context"
	"testing"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/test/grpcassert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func TestCreateAccessRequestRequiresUserID(t *testing.T) {
	th := newAccessRequestHandlersWithStores(t, newFakeStore(), newFakeStore(), newFakeStore())

	_, err := th.CreateAccessRequest(context.Background(), &aiv1.CreateAccessRequestRequest{
		AgentId: "agent-1",
		Scope:   "invoke",
	})
	grpcassert.StatusCode(t, err, codes.PermissionDenied)
}

func TestListAccessRequestsRejectsMissingRole(t *testing.T) {
	th := newAccessRequestHandlersWithStores(t, newFakeStore(), newFakeStore(), newFakeStore())
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))

	_, err := th.ListAccessRequests(ctx, &aiv1.ListAccessRequestsRequest{})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestReviewAccessRequestRejectsInvalidDecision(t *testing.T) {
	th := newAccessRequestHandlersWithStores(t, newFakeStore(), newFakeStore(), newFakeStore())
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "owner-1"))

	_, err := th.ReviewAccessRequest(ctx, &aiv1.ReviewAccessRequestRequest{
		AccessRequestId: "request-1",
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestRevokeAccessRequestRequiresUserID(t *testing.T) {
	th := newAccessRequestHandlersWithStores(t, newFakeStore(), newFakeStore(), newFakeStore())

	_, err := th.RevokeAccessRequest(context.Background(), &aiv1.RevokeAccessRequestRequest{
		AccessRequestId: "request-1",
	})
	grpcassert.StatusCode(t, err, codes.PermissionDenied)
}
