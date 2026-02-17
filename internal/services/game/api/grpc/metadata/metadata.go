package metadata

// Package metadata defines the cross-service headers that keep request context
// stable across gRPC boundaries.
//
// IDs are intentionally cheap, stable identifiers used by transport, logs, and
// telemetry stores so behavior remains observable without leaking domain payloads.

import (
	"context"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// RequestIDHeader is the gRPC metadata key for request correlation IDs.
const RequestIDHeader = "x-fracturing-space-request-id"

// InvocationIDHeader is the gRPC metadata key for MCP tool invocation IDs.
const InvocationIDHeader = "x-fracturing-space-invocation-id"

// ParticipantIDHeader is the gRPC metadata key for caller identity hints.
// It allows downstream authz checks and audit logs to attach context to actions.
const ParticipantIDHeader = "x-fracturing-space-participant-id"

// UserIDHeader is the gRPC metadata key for user impersonation hints.
// In web flows this may represent end-user identity for campaign filtering.
const UserIDHeader = "x-fracturing-space-user-id"

// CampaignIDHeader is the gRPC metadata key for campaign routing hints.
// MCP and UI flows can pass this through for consistency and observability.
const CampaignIDHeader = "x-fracturing-space-campaign-id"

// SessionIDHeader is the gRPC metadata key for session routing hints.
// Useful for scoped event and telemetry reads under an active session.
const SessionIDHeader = "x-fracturing-space-session-id"

// contextKey stores metadata values in context.
type contextKey string

const (
	// requestIDContextKey stores the request ID in context.
	requestIDContextKey contextKey = "fracturing-space-request-id"
	// invocationIDContextKey stores the invocation ID in context.
	invocationIDContextKey contextKey = "fracturing-space-invocation-id"
)

// RequestIDFromContext returns the request ID stored in context.
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(requestIDContextKey).(string)
	return value
}

// InvocationIDFromContext returns the invocation ID stored in context.
func InvocationIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(invocationIDContextKey).(string)
	return value
}

// ParticipantIDFromContext returns the participant ID from incoming metadata.
func ParticipantIDFromContext(ctx context.Context) string {
	return metadataValueFromIncomingContext(ctx, ParticipantIDHeader)
}

// UserIDFromContext returns the user ID from incoming metadata.
func UserIDFromContext(ctx context.Context) string {
	return metadataValueFromIncomingContext(ctx, UserIDHeader)
}

// CampaignIDFromContext returns the campaign ID from incoming metadata.
func CampaignIDFromContext(ctx context.Context) string {
	return metadataValueFromIncomingContext(ctx, CampaignIDHeader)
}

// SessionIDFromContext returns the session ID from incoming metadata.
func SessionIDFromContext(ctx context.Context) string {
	return metadataValueFromIncomingContext(ctx, SessionIDHeader)
}

// WithRequestID stores the request ID in context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, requestIDContextKey, requestID)
}

// WithInvocationID stores the invocation ID in context.
func WithInvocationID(ctx context.Context, invocationID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, invocationIDContextKey, invocationID)
}

// IsPrintableASCII reports whether a string contains only printable ASCII characters.
func IsPrintableASCII(value string) bool {
	if value == "" {
		return false
	}
	for i := 0; i < len(value); i++ {
		if value[i] < 0x20 || value[i] > 0x7e {
			return false
		}
	}
	return true
}

// FirstMetadataValue returns the first printable ASCII metadata value for a key.
// Printable filtering drops control characters to keep downstream logs and
// downstream request propagation robust.
func FirstMetadataValue(md metadata.MD, key string) string {
	if len(md) == 0 {
		return ""
	}
	for mdKey, values := range md {
		if !strings.EqualFold(mdKey, key) {
			continue
		}
		for _, value := range values {
			if IsPrintableASCII(value) {
				return value
			}
		}
	}
	return ""
}

// UnaryServerInterceptor enforces project request metadata on unary calls.
// The interceptor guarantees every inbound call gets correlation identifiers, so
// downstream logs and storage can correlate activity even when clients omit headers.
func UnaryServerInterceptor(idGenerator func() (string, error)) grpc.UnaryServerInterceptor {
	if idGenerator == nil {
		idGenerator = id.NewID
	}
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		updatedCtx, requestID, invocationID, err := ensureRequestMetadata(ctx, idGenerator)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "ensure request metadata: %v", err)
		}
		headerErr := grpc.SetHeader(updatedCtx, responseHeaders(requestID, invocationID))
		if headerErr != nil {
			return nil, status.Errorf(codes.Internal, "set response metadata: %v", headerErr)
		}

		// TODO: Add request_id and invocation_id to OpenTelemetry span attributes for unary calls once tracing is added.
		// Metadata headers are propagated today as the transport-agnostic audit rail.
		// This keeps correlation consistent across CLI, MCP, and web callers even before
		// tracing is introduced. Spans can be added behind the same metadata keys later
		// without changing this contract.
		// Metadata headers are propagated today for auditability; tracing is intentionally
		// deferred to avoid blocking this layer on observability plumbing choices.

		return handler(updatedCtx, req)
	}
}

// StreamServerInterceptor enforces project request metadata on streaming calls.
// Streaming calls also receive stable request/invocation IDs so cross-stream
// diagnostics can continue to stitch events together.
func StreamServerInterceptor(idGenerator func() (string, error)) grpc.StreamServerInterceptor {
	if idGenerator == nil {
		idGenerator = id.NewID
	}
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		updatedCtx, requestID, invocationID, err := ensureRequestMetadata(stream.Context(), idGenerator)
		if err != nil {
			return status.Errorf(codes.Internal, "ensure request metadata: %v", err)
		}
		headerErr := stream.SetHeader(responseHeaders(requestID, invocationID))
		if headerErr != nil {
			return status.Errorf(codes.Internal, "set response metadata: %v", headerErr)
		}

		// TODO: Add request_id and invocation_id to OpenTelemetry span attributes for stream calls once tracing is added.
		// Metadata headers are propagated today as the transport-agnostic audit rail.
		// This keeps correlation consistent across CLI, MCP, and web callers even before
		// tracing is introduced. Spans can be added behind the same metadata keys later
		// without changing this contract.
		// Metadata headers are propagated today for auditability; tracing is intentionally
		// deferred to avoid blocking this layer on observability plumbing choices.

		return handler(srv, &wrappedServerStream{ServerStream: stream, ctx: updatedCtx})
	}
}

// wrappedServerStream overrides the context for a gRPC stream.
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the updated stream context.
func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// ensureRequestMetadata ensures the request ID exists and returns updated context.
// This guarantees correlation identifiers are always attached to downstream handlers,
// even when callers are not yet emitting them.
func ensureRequestMetadata(ctx context.Context, idGenerator func() (string, error)) (context.Context, string, string, error) {
	requestID := requestIDFromIncomingContext(ctx)
	invocationID := invocationIDFromIncomingContext(ctx)
	if requestID == "" {
		generatedID, err := idGenerator()
		if err != nil {
			return nil, "", "", err
		}
		requestID = generatedID
	}

	updatedCtx := WithRequestID(ctx, requestID)
	if invocationID != "" {
		updatedCtx = WithInvocationID(updatedCtx, invocationID)
	}
	return updatedCtx, requestID, invocationID, nil
}

// requestIDFromIncomingContext returns the request ID from incoming metadata.
func requestIDFromIncomingContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	return FirstMetadataValue(md, RequestIDHeader)
}

// invocationIDFromIncomingContext returns the invocation ID from incoming metadata.
func invocationIDFromIncomingContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	return FirstMetadataValue(md, InvocationIDHeader)
}

func metadataValueFromIncomingContext(ctx context.Context, header string) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	return FirstMetadataValue(md, header)
}

// responseHeaders builds response metadata headers from IDs.
func responseHeaders(requestID, invocationID string) metadata.MD {
	headers := metadata.Pairs(RequestIDHeader, requestID)
	if invocationID != "" {
		headers.Append(InvocationIDHeader, invocationID)
	}
	return headers
}
