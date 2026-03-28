package server

import (
	"context"
	"strings"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// validateIncomingServiceIdentity rejects unrecognized service IDs while still
// allowing user-scoped calls that do not carry internal identity metadata.
func validateIncomingServiceIdentity(allowed map[string]struct{}) func(context.Context) error {
	return func(ctx context.Context) error {
		serviceID := strings.ToLower(strings.TrimSpace(serviceIDFromIncomingContext(ctx)))
		if serviceID == "" {
			return nil
		}
		if len(allowed) == 0 {
			return status.Error(codes.Internal, "internal service identity allowlist is not configured")
		}
		if _, ok := allowed[serviceID]; !ok {
			return status.Error(codes.PermissionDenied, "internal service identity is not allowed")
		}
		return nil
	}
}

func serviceIdentityValidationUnaryInterceptor(allowed map[string]struct{}) grpc.UnaryServerInterceptor {
	validate := validateIncomingServiceIdentity(allowed)
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if err := validate(ctx); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func serviceIdentityValidationStreamInterceptor(allowed map[string]struct{}) grpc.StreamServerInterceptor {
	validate := validateIncomingServiceIdentity(allowed)
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if err := validate(stream.Context()); err != nil {
			return err
		}
		return handler(srv, stream)
	}
}

func serviceIDFromIncomingContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	return grpcmeta.FirstMetadataValue(md, grpcmeta.ServiceIDHeader)
}
