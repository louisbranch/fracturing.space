// Package errors provides web-specific typed application errors built on top
// of the shared httperrors package.
package errors

import (
	stderrors "errors"
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/shared/httperrors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Re-export shared types so existing web callers compile without import changes.
type (
	Kind              = httperrors.Kind
	Error             = httperrors.Error
	GRPCStatusMapping = httperrors.GRPCStatusMapping
)

const (
	KindUnknown      = httperrors.KindUnknown
	KindInvalidInput = httperrors.KindInvalidInput
	KindUnauthorized = httperrors.KindUnauthorized
	KindForbidden    = httperrors.KindForbidden
	KindConflict     = httperrors.KindConflict
	KindUnavailable  = httperrors.KindUnavailable
	KindNotFound     = httperrors.KindNotFound
)

// E builds a typed Error.
var E = httperrors.E

// EK builds a typed Error with a localization key.
var EK = httperrors.EK

// LocalizationKey returns the structured localization key when available.
var LocalizationKey = httperrors.LocalizationKey

// PublicMessage returns the explicit transport-safe message carried by a typed error.
var PublicMessage = httperrors.PublicMessage

// ResolveRichMessage localizes preserved rich transport details when possible.
var ResolveRichMessage = httperrors.ResolveRichMessage

// MapGRPCTransportError converts gRPC transport errors into typed web errors.
var MapGRPCTransportError = httperrors.MapGRPCTransportError

// HTTPStatus maps an error to an HTTP status code.
// Web uses a slightly different gRPC fallback mapping than the shared default:
// FailedPrecondition maps to Conflict (409) rather than BadRequest (400).
func HTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	// Typed application errors share the same Kind→status mapping.
	var appErr httperrors.Error
	if stderrors.As(err, &appErr) {
		return httperrors.HTTPStatus(err)
	}
	// Raw gRPC errors use web-specific mapping.
	return grpcErrorHTTPStatus(err, http.StatusInternalServerError)
}

// grpcErrorHTTPStatus applies web-specific gRPC status mapping where
// FailedPrecondition maps to Conflict instead of BadRequest.
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
	case codes.Unavailable, codes.DeadlineExceeded:
		return http.StatusServiceUnavailable
	default:
		return fallback
	}
}
