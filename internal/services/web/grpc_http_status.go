package web

import (
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// grpcErrorHTTPStatus converts transport errors into web-facing status codes.
// This keeps web routing logic simple: handlers can stay HTTP-first while still
// preserving typed failures from gRPC for dashboards and templates.
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
