// Package errors defines web typed application errors.
package errors

import (
	stderrors "errors"
	"net/http"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Kind classifies application failures for consistent HTTP mapping.
type Kind string

const (
	KindUnknown      Kind = "unknown"
	KindInvalidInput Kind = "invalid_input"
	KindUnauthorized Kind = "unauthorized"
	KindForbidden    Kind = "forbidden"
	KindConflict     Kind = "conflict"
	KindUnavailable  Kind = "unavailable"
	KindNotFound     Kind = "not_found"
)

// GRPCStatusMapping describes how a gRPC transport failure should
// downgrade into web error classification when a service-specific fallback exists.
type GRPCStatusMapping struct {
	FallbackKind    Kind
	FallbackKey     string
	FallbackMessage string
}

// MapGRPCTransportError converts gRPC transport errors into typed web errors with
// a stable, policy-driven fallback.
func MapGRPCTransportError(err error, mapping GRPCStatusMapping) error {
	if err == nil {
		return nil
	}
	var appErr Error
	if stderrors.As(err, &appErr) {
		return appErr
	}

	st, ok := status.FromError(err)
	if !ok {
		return mapWithFallback(mapping)
	}
	switch st.Code() {
	case codes.InvalidArgument, codes.OutOfRange, codes.FailedPrecondition, codes.AlreadyExists:
		return mapWithFallback(mapping)
	case codes.Unauthenticated:
		return E(KindUnauthorized, "authentication required")
	case codes.PermissionDenied:
		return E(KindForbidden, "access denied")
	case codes.NotFound:
		return E(KindNotFound, "resource not found")
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted, codes.Canceled:
		return E(KindUnavailable, "dependency is temporarily unavailable")
	case codes.Aborted:
		return E(KindConflict, st.Message())
	default:
		return mapWithFallback(mapping)
	}
}

func mapWithFallback(mapping GRPCStatusMapping) error {
	if strings.TrimSpace(mapping.FallbackKey) != "" {
		return EK(mapping.FallbackKind, mapping.FallbackKey, strings.TrimSpace(mapping.FallbackMessage))
	}
	return E(mapping.FallbackKind, strings.TrimSpace(mapping.FallbackMessage))
}

// Error is a typed web application failure.
type Error struct {
	Kind    Kind
	Key     string
	Message string
}

// Error renders the human-readable message.
func (e Error) Error() string {
	if e.Message == "" {
		return string(e.Kind)
	}
	return e.Message
}

// E builds a typed Error.
func E(kind Kind, message string) error {
	return Error{Kind: kind, Message: message}
}

// EK builds a typed Error with a localization key.
func EK(kind Kind, key string, message string) error {
	return Error{Kind: kind, Key: strings.TrimSpace(key), Message: message}
}

// LocalizationKey returns the structured localization key when available.
func LocalizationKey(err error) string {
	if err == nil {
		return ""
	}
	var appErr Error
	if !stderrors.As(err, &appErr) {
		return ""
	}
	return strings.TrimSpace(appErr.Key)
}

// HTTPStatus maps an error to an HTTP status code.
func HTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	var appErr Error
	if !stderrors.As(err, &appErr) {
		return grpcErrorHTTPStatus(err, http.StatusInternalServerError)
	}
	switch appErr.Kind {
	case KindInvalidInput:
		return http.StatusBadRequest
	case KindConflict:
		return http.StatusConflict
	case KindUnauthorized:
		return http.StatusUnauthorized
	case KindForbidden:
		return http.StatusForbidden
	case KindUnavailable:
		return http.StatusServiceUnavailable
	case KindNotFound:
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

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
