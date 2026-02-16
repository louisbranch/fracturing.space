package web

import (
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// grpcErrorHTTPStatus maps common gRPC status codes to HTTP status codes.
// It returns fallback when err is not a gRPC status or is unmapped.
func grpcErrorHTTPStatus(err error, fallback int) int {
	st, ok := status.FromError(err)
	if !ok {
		return fallback
	}
	switch st.Code() {
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.NotFound:
		return http.StatusNotFound
	case codes.FailedPrecondition:
		return http.StatusConflict
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	default:
		return fallback
	}
}
