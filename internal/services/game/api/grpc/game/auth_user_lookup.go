package game

import (
	"context"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// authUsername loads the canonical auth-owned username for a linked user.
func authUsername(ctx context.Context, authClient authv1.AuthServiceClient, userID string, notFoundErr error) (string, error) {
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
