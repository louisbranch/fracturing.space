package interceptors

import (
	"context"
	"strings"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// InternalServiceIdentityConfig configures internal service-id gate behavior.
type InternalServiceIdentityConfig struct {
	// MethodPrefixes scopes interception to full-method prefixes.
	MethodPrefixes []string
	// AllowedServiceIDs lists accepted x-fracturing-space-service-id values.
	AllowedServiceIDs map[string]struct{}
}

// InternalServiceIdentityUnaryInterceptor enforces service-id allowlists for scoped methods.
func InternalServiceIdentityUnaryInterceptor(cfg InternalServiceIdentityConfig) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !internalIdentityMethodScoped(info.FullMethod, cfg.MethodPrefixes) {
			return handler(ctx, req)
		}
		if err := validateInternalServiceIdentity(ctx, cfg.AllowedServiceIDs); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

// InternalServiceIdentityStreamInterceptor enforces service-id allowlists for scoped methods.
func InternalServiceIdentityStreamInterceptor(cfg InternalServiceIdentityConfig) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !internalIdentityMethodScoped(info.FullMethod, cfg.MethodPrefixes) {
			return handler(srv, stream)
		}
		if err := validateInternalServiceIdentity(stream.Context(), cfg.AllowedServiceIDs); err != nil {
			return err
		}
		return handler(srv, stream)
	}
}

func internalIdentityMethodScoped(fullMethod string, prefixes []string) bool {
	fullMethod = strings.TrimSpace(fullMethod)
	if fullMethod == "" || len(prefixes) == 0 {
		return false
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(fullMethod, strings.TrimSpace(prefix)) {
			return true
		}
	}
	return false
}

func validateInternalServiceIdentity(ctx context.Context, allowed map[string]struct{}) error {
	if len(allowed) == 0 {
		return status.Error(codes.Internal, "internal service identity allowlist is not configured")
	}
	serviceID := strings.TrimSpace(serviceIDFromContext(ctx))
	if serviceID == "" {
		return status.Error(codes.PermissionDenied, "internal service identity is required")
	}
	if _, ok := allowed[strings.ToLower(serviceID)]; !ok {
		return status.Error(codes.PermissionDenied, "internal service identity is not allowed")
	}
	return nil
}

func serviceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	return grpcmeta.FirstMetadataValue(md, grpcmeta.ServiceIDHeader)
}
