package ai

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestRequireUnaryRequest(t *testing.T) {
	t.Parallel()

	err := requireUnaryRequest[*struct{}](nil, "request is required")
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestRequireCallerUserID(t *testing.T) {
	t.Parallel()

	t.Run("missing user id", func(t *testing.T) {
		t.Parallel()

		_, err := requireCallerUserID(context.Background())
		if status.Code(err) != codes.PermissionDenied {
			t.Fatalf("status code = %v, want %v", status.Code(err), codes.PermissionDenied)
		}
	})

	t.Run("returns trimmed user id", func(t *testing.T) {
		t.Parallel()

		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, " user-1 "))
		userID, err := requireCallerUserID(ctx)
		if err != nil {
			t.Fatalf("requireCallerUserID error = %v", err)
		}
		if userID != "user-1" {
			t.Fatalf("userID = %q, want %q", userID, "user-1")
		}
	})
}

func TestRequireUserScopedUnaryRequest(t *testing.T) {
	t.Parallel()

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))

	userID, err := requireUserScopedUnaryRequest(ctx, &struct{}{}, "request is required")
	if err != nil {
		t.Fatalf("requireUserScopedUnaryRequest error = %v", err)
	}
	if userID != "user-1" {
		t.Fatalf("userID = %q, want %q", userID, "user-1")
	}
}
