package requestctx

import (
	"context"
	"strings"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc/metadata"
)

// WithParticipantID injects a participant ID into incoming gRPC metadata.
func WithParticipantID(participantID string) context.Context {
	if participantID == "" {
		return context.Background()
	}
	md := metadata.Pairs(grpcmeta.ParticipantIDHeader, participantID)
	return metadata.NewIncomingContext(context.Background(), md)
}

// WithUserID injects a user ID into incoming gRPC metadata.
func WithUserID(userID string) context.Context {
	if userID == "" {
		return context.Background()
	}
	md := metadata.Pairs(grpcmeta.UserIDHeader, userID)
	return metadata.NewIncomingContext(context.Background(), md)
}

// WithAdminOverride injects admin override metadata for transport tests.
func WithAdminOverride(reason string) context.Context {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "test-override"
	}
	md := metadata.Pairs(
		grpcmeta.PlatformRoleHeader, grpcmeta.PlatformRoleAdmin,
		grpcmeta.AuthzOverrideReasonHeader, reason,
		grpcmeta.UserIDHeader, "user-admin-test",
	)
	return metadata.NewIncomingContext(context.Background(), md)
}
