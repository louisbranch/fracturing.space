package grpcauthctx

import (
	"context"
	"strings"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc/metadata"
)

// WithUserID returns a context with user-id gRPC metadata when userID is non-empty.
func WithUserID(ctx context.Context, userID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, grpcmeta.UserIDHeader, userID)
}

// WithParticipantID returns a context with participant-id gRPC metadata when participantID is non-empty.
func WithParticipantID(ctx context.Context, participantID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	participantID = strings.TrimSpace(participantID)
	if participantID == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, grpcmeta.ParticipantIDHeader, participantID)
}
