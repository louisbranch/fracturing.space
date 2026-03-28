package grpcmeta

import (
	"context"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// RequestIDHeader is the gRPC metadata key for request correlation IDs.
const RequestIDHeader = "x-fracturing-space-request-id"

// InvocationIDHeader is the gRPC metadata key for tool invocation IDs.
const InvocationIDHeader = "x-fracturing-space-invocation-id"

// ParticipantIDHeader is the gRPC metadata key for caller identity hints.
const ParticipantIDHeader = "x-fracturing-space-participant-id"

// UserIDHeader is the gRPC metadata key for user impersonation hints.
const UserIDHeader = "x-fracturing-space-user-id"

// CampaignIDHeader is the gRPC metadata key for campaign routing hints.
const CampaignIDHeader = "x-fracturing-space-campaign-id"

// SessionIDHeader is the gRPC metadata key for session routing hints.
const SessionIDHeader = "x-fracturing-space-session-id"

// PlatformRoleHeader is the gRPC metadata key for platform-level roles.
const PlatformRoleHeader = "x-fracturing-space-platform-role"

// AuthzOverrideReasonHeader is the metadata key for admin override reason text.
const AuthzOverrideReasonHeader = "x-fracturing-space-authz-override-reason"

// ServiceIDHeader is the gRPC metadata key for internal service identity.
const ServiceIDHeader = "x-fracturing-space-service-id"

// LocaleHeader is the gRPC metadata key for the caller's preferred locale.
const LocaleHeader = "x-fracturing-space-locale"

// DefaultLocale is the fallback locale when no locale metadata is present.
const DefaultLocale = "en-US"

// PlatformRoleAdmin identifies platform administrators.
const PlatformRoleAdmin = "ADMIN"

type contextKey string

const (
	requestIDContextKey    contextKey = "fracturing-space-request-id"
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

// ServiceIDFromContext returns the internal service ID from incoming metadata.
func ServiceIDFromContext(ctx context.Context) string {
	return metadataValueFromIncomingContext(ctx, ServiceIDHeader)
}

// LocaleFromContext returns the caller locale from incoming metadata or the
// project default when no locale is supplied.
func LocaleFromContext(ctx context.Context) string {
	if locale := metadataValueFromIncomingContext(ctx, LocaleHeader); locale != "" {
		return locale
	}
	return DefaultLocale
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

// UnaryServerInterceptor ensures inbound unary requests always carry stable
// request metadata, generating request IDs when callers omit them.
func UnaryServerInterceptor(idGenerator func() (string, error)) grpc.UnaryServerInterceptor {
	if idGenerator == nil {
		idGenerator = id.NewID
	}
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		updatedCtx, requestID, invocationID, err := ensureRequestMetadata(ctx, idGenerator)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "ensure request metadata: %v", err)
		}
		if err := grpc.SetHeader(updatedCtx, responseHeaders(requestID, invocationID)); err != nil {
			return nil, status.Errorf(codes.Internal, "set response metadata: %v", err)
		}

		if span := trace.SpanFromContext(updatedCtx); span.IsRecording() {
			span.SetAttributes(
				attribute.String("request.id", requestID),
				attribute.String("request.invocation_id", invocationID),
			)
		}

		return handler(updatedCtx, req)
	}
}

// StreamServerInterceptor ensures inbound streaming requests always carry stable
// request metadata, generating request IDs when callers omit them.
func StreamServerInterceptor(idGenerator func() (string, error)) grpc.StreamServerInterceptor {
	if idGenerator == nil {
		idGenerator = id.NewID
	}
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		updatedCtx, requestID, invocationID, err := ensureRequestMetadata(stream.Context(), idGenerator)
		if err != nil {
			return status.Errorf(codes.Internal, "ensure request metadata: %v", err)
		}
		if err := stream.SetHeader(responseHeaders(requestID, invocationID)); err != nil {
			return status.Errorf(codes.Internal, "set response metadata: %v", err)
		}

		if span := trace.SpanFromContext(updatedCtx); span.IsRecording() {
			span.SetAttributes(
				attribute.String("request.id", requestID),
				attribute.String("request.invocation_id", invocationID),
			)
		}

		return handler(srv, &wrappedServerStream{ServerStream: stream, ctx: updatedCtx})
	}
}

type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

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

func requestIDFromIncomingContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	return FirstMetadataValue(md, RequestIDHeader)
}

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

func responseHeaders(requestID, invocationID string) metadata.MD {
	headers := metadata.Pairs(RequestIDHeader, requestID)
	if invocationID != "" {
		headers.Append(InvocationIDHeader, invocationID)
	}
	return headers
}
