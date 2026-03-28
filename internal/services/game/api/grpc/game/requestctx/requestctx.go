package requestctx

import (
	"context"
	"strings"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"google.golang.org/grpc/metadata"
)

// WithParticipantID injects a participant ID into incoming gRPC metadata.
func WithParticipantID(ctx context.Context, participantID string) context.Context {
	if participantID == "" {
		return ctx
	}
	md := metadata.Pairs(grpcmeta.ParticipantIDHeader, participantID)
	return metadata.NewIncomingContext(ctx, md)
}

// WithUserID injects a user ID into incoming gRPC metadata.
func WithUserID(ctx context.Context, userID string) context.Context {
	if userID == "" {
		return ctx
	}
	md := metadata.Pairs(grpcmeta.UserIDHeader, userID)
	return metadata.NewIncomingContext(ctx, md)
}

// WithAdminOverride injects admin override metadata for transport tests.
func WithAdminOverride(ctx context.Context, reason string) context.Context {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "test-override"
	}
	md := metadata.Pairs(
		grpcmeta.PlatformRoleHeader, grpcmeta.PlatformRoleAdmin,
		grpcmeta.AuthzOverrideReasonHeader, reason,
		grpcmeta.UserIDHeader, "user-admin-test",
	)
	return metadata.NewIncomingContext(ctx, md)
}
