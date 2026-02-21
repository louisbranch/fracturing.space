package grpcauthctx

import (
	"context"
	"strings"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc"
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

// WithAdminOverride returns a context with platform ADMIN override metadata.
func WithAdminOverride(ctx context.Context, reason string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(
		ctx,
		grpcmeta.PlatformRoleHeader, grpcmeta.PlatformRoleAdmin,
		grpcmeta.AuthzOverrideReasonHeader, reason,
	)
}

// AdminOverrideUnaryClientInterceptor appends ADMIN override metadata to unary calls.
func AdminOverrideUnaryClientInterceptor(reason string) grpc.UnaryClientInterceptor {
	reason = strings.TrimSpace(reason)
	return func(
		ctx context.Context,
		method string,
		req any,
		reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		return invoker(WithAdminOverride(ctx, reason), method, req, reply, cc, opts...)
	}
}

// AdminOverrideStreamClientInterceptor appends ADMIN override metadata to stream calls.
func AdminOverrideStreamClientInterceptor(reason string) grpc.StreamClientInterceptor {
	reason = strings.TrimSpace(reason)
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		return streamer(WithAdminOverride(ctx, reason), desc, cc, method, opts...)
	}
}
