package handler

import (
	"context"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthUserClient is the narrow auth dependency needed by game transport.
type AuthUserClient interface {
	GetUser(ctx context.Context, req *authv1.GetUserRequest, opts ...grpc.CallOption) (*authv1.GetUserResponse, error)
}

// AuthUsername loads the canonical auth-owned username for a linked user.
func AuthUsername(ctx context.Context, authClient AuthUserClient, userID string, notFoundErr error) (string, error) {
	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return "", nil
	}
	if authClient == nil {
		return "", status.Error(codes.Internal, "auth client is not configured")
	}

	userResponse, err := authClient.GetUser(ctx, &authv1.GetUserRequest{UserId: trimmedUserID})
	if err != nil {
		if status.Code(err) == codes.NotFound && notFoundErr != nil {
			return "", notFoundErr
		}
		return "", status.Errorf(codes.Internal, "get auth user: %v", err)
	}
	if userResponse == nil || userResponse.GetUser() == nil {
		return "", status.Error(codes.Internal, "auth user response is missing")
	}

	username := strings.TrimSpace(userResponse.GetUser().GetUsername())
	if username == "" {
		return "", status.Error(codes.Internal, "auth user username is missing")
	}
	return username, nil
}
