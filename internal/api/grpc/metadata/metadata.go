package metadata

import (
	"context"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/id"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// RequestIDHeader is the gRPC metadata key for request correlation IDs.
const RequestIDHeader = "x-duality-request-id"

// InvocationIDHeader is the gRPC metadata key for MCP tool invocation IDs.
const InvocationIDHeader = "x-duality-invocation-id"

// ParticipantIDHeader is the gRPC metadata key for caller identity hints.
const ParticipantIDHeader = "x-duality-participant-id"

// CampaignIDHeader is the gRPC metadata key for campaign routing hints.
const CampaignIDHeader = "x-duality-campaign-id"

// SessionIDHeader is the gRPC metadata key for session routing hints.
const SessionIDHeader = "x-duality-session-id"

// contextKey stores metadata values in context.
type contextKey string

const (
	// requestIDContextKey stores the request ID in context.
	requestIDContextKey contextKey = "duality-request-id"
	// invocationIDContextKey stores the invocation ID in context.
	invocationIDContextKey contextKey = "duality-invocation-id"
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

// UnaryServerInterceptor enforces Fracturing.Space request metadata on unary calls.
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

		// TODO: Add request_id and invocation_id to OpenTelemetry span attributes once tracing is added.

		return handler(updatedCtx, req)
	}
}

// StreamServerInterceptor enforces Fracturing.Space request metadata on streaming calls.
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

		// TODO: Add request_id and invocation_id to OpenTelemetry span attributes once tracing is added.

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
