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

type socialProfileClient interface {
	GetUserProfile(ctx context.Context, in *socialv1.GetUserProfileRequest, opts ...grpc.CallOption) (*socialv1.GetUserProfileResponse, error)
	SetUserProfile(ctx context.Context, in *socialv1.SetUserProfileRequest, opts ...grpc.CallOption) (*socialv1.SetUserProfileResponse, error)
}

// SignupSocialProfileHandler ensures signup events have a corresponding social
// profile so web flows can immediately render profile completion prompts.
type SignupSocialProfileHandler struct {
	social socialProfileClient
}

// NewSignupSocialProfileHandler creates the social profile bootstrap handler.
func NewSignupSocialProfileHandler(social socialProfileClient) *SignupSocialProfileHandler {
	return &SignupSocialProfileHandler{social: social}
}

// Handle idempotently creates a social profile record for signup events.
func (h *SignupSocialProfileHandler) Handle(ctx context.Context, event *authv1.IntegrationOutboxEvent) error {
	if h == nil || h.social == nil {
		return Permanent(fmt.Errorf("social profile client is not configured"))
	}

	payload, err := decodeSignupCompletedPayload(event)
	if err != nil {
		return Permanent(err)
	}

	if _, err := h.social.GetUserProfile(ctx, &socialv1.GetUserProfileRequest{UserId: payload.UserID}); err == nil {
		return nil
	} else if status.Code(err) != codes.NotFound {
		if isPermanentSocialProfileError(err) {
			return Permanent(err)
		}
		return err
	}

	_, err = h.social.SetUserProfile(ctx, &socialv1.SetUserProfileRequest{UserId: payload.UserID})
	if err == nil {
		return nil
	}
	if isPermanentSocialProfileError(err) {
		return Permanent(err)
	}
	return err
}

func isPermanentSocialProfileError(err error) bool {
	switch status.Code(err) {
	case codes.InvalidArgument, codes.PermissionDenied, codes.NotFound, codes.FailedPrecondition, codes.Unauthenticated:
		return true
	default:
		return false
	}
}
