package interceptors

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// ErrorConversionUnaryInterceptor normalizes handler errors at the transport
// boundary so individual handlers never need to convert domain errors to gRPC
// status. Errors that are already gRPC status pass through unchanged; domain
// errors are mapped through the structured error system; anything else becomes
// codes.Internal with a generic message.
func ErrorConversionUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		if err == nil {
			return resp, nil
		}
		if _, ok := status.FromError(err); ok {
			return resp, err
		}
		return nil, grpcerror.HandleDomainError(err)
	}
}

// ErrorConversionStreamInterceptor is the streaming equivalent of
// ErrorConversionUnaryInterceptor.
func ErrorConversionStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		err := handler(srv, stream)
		if err == nil {
			return nil
		}
		if _, ok := status.FromError(err); ok {
			return err
		}
		return grpcerror.HandleDomainError(err)
	}
}
