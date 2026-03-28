package testclients

import (
	"context"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler/social"
	"google.golang.org/grpc"
)

// FakeAuthClient is a test double for auth RPC dependencies.
type FakeAuthClient struct {
	User               *authv1.User
	GetUserErr         error
	LastGetUserRequest *authv1.GetUserRequest
}

// FakeSocialClient is a test double for social RPC dependencies.
type FakeSocialClient struct {
	Profile               *socialv1.UserProfile
	GetUserProfileErr     error
	LastGetUserProfileReq *socialv1.GetUserProfileRequest
	GetUserProfileCalls   int
}

var (
	_ handler.AuthUserClient = (*FakeAuthClient)(nil)
	_ social.ProfileClient   = (*FakeSocialClient)(nil)
)

func (f *FakeAuthClient) GetUser(ctx context.Context, req *authv1.GetUserRequest, opts ...grpc.CallOption) (*authv1.GetUserResponse, error) {
	f.LastGetUserRequest = req
	if f.GetUserErr != nil {
		return nil, f.GetUserErr
	}
	return &authv1.GetUserResponse{User: f.User}, nil
}

func (f *FakeSocialClient) GetUserProfile(_ context.Context, req *socialv1.GetUserProfileRequest, _ ...grpc.CallOption) (*socialv1.GetUserProfileResponse, error) {
	f.GetUserProfileCalls++
	f.LastGetUserProfileReq = req
	if f.GetUserProfileErr != nil {
		return nil, f.GetUserProfileErr
	}
	if f.Profile == nil {
		return &socialv1.GetUserProfileResponse{}, nil
	}
	return &socialv1.GetUserProfileResponse{UserProfile: f.Profile}, nil
}
