package grpcstatus

import (
	"log/slog"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Internal logs the full server-side error and returns a sanitized gRPC
// Internal status for client-facing transport boundaries.
func Internal(message string, err error) error {
	slog.Error(message, "error", err)
	return status.Error(codes.Internal, message)
}

// ApplyErrorWithDomainCodePreserve preserves structured domain errors and
// wraps unknown apply failures in a sanitized gRPC Internal status.
func ApplyErrorWithDomainCodePreserve(message string) func(error) error {
	return func(err error) error {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return err
		}
		return Internal(message, err)
	}
}
