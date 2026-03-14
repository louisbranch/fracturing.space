package domain

import (
	"context"
	"fmt"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type socialDirectoryClient interface {
	SyncDirectoryUser(ctx context.Context, in *socialv1.SyncDirectoryUserRequest, opts ...grpc.CallOption) (*socialv1.SyncDirectoryUserResponse, error)
}

// SignupSocialDirectoryHandler ensures signup events update the social username directory.
type SignupSocialDirectoryHandler struct {
	social socialDirectoryClient
}

// NewSignupSocialDirectoryHandler creates the signup directory sync handler.
func NewSignupSocialDirectoryHandler(social socialDirectoryClient) *SignupSocialDirectoryHandler {
	return &SignupSocialDirectoryHandler{social: social}
}

// Handle idempotently syncs auth username identity into the social directory.
func (h *SignupSocialDirectoryHandler) Handle(ctx context.Context, event *authv1.IntegrationOutboxEvent) error {
	if h == nil || h.social == nil {
		return Permanent(fmt.Errorf("social directory client is not configured"))
	}

	payload, err := decodeSignupCompletedPayload(event)
	if err != nil {
		return Permanent(err)
	}
	if payload.Username == "" {
		return Permanent(fmt.Errorf("username is required in signup payload"))
	}

	_, err = h.social.SyncDirectoryUser(ctx, &socialv1.SyncDirectoryUserRequest{
		UserId:   payload.UserID,
		Username: payload.Username,
	})
	if err == nil {
		return nil
	}
	if isPermanentSocialDirectoryError(err) {
		return Permanent(err)
	}
	return err
}

func isPermanentSocialDirectoryError(err error) bool {
	switch status.Code(err) {
	case codes.InvalidArgument, codes.PermissionDenied, codes.NotFound, codes.FailedPrecondition, codes.Unauthenticated:
		return true
	default:
		return false
	}
}
