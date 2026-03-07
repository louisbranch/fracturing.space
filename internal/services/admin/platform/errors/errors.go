// Package errors provides typed admin application errors with consistent
// logging, gRPC error classification, and HTTP status mapping.
package errors

import (
	stderrors "errors"
	"log"
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

// Error is a typed admin application failure.
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

// GRPCStatusMapping describes how a gRPC transport failure should
// downgrade into admin error classification when a service-specific
// fallback exists.
type GRPCStatusMapping struct {
	FallbackKind    Kind
	FallbackKey     string
	FallbackMessage string
}

// MapGRPCTransportError converts gRPC transport errors into typed admin
// errors with a stable, policy-driven fallback.
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

// mapWithFallback maps values across transport and domain boundaries.
func mapWithFallback(mapping GRPCStatusMapping) error {
	if strings.TrimSpace(mapping.FallbackKey) != "" {
		return EK(mapping.FallbackKind, mapping.FallbackKey, strings.TrimSpace(mapping.FallbackMessage))
	}
	return E(mapping.FallbackKind, strings.TrimSpace(mapping.FallbackMessage))
}

// HTTPStatus maps an error to an HTTP status code. It understands
// both typed Error values and raw gRPC status errors.
func HTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	var appErr Error
	if !stderrors.As(err, &appErr) {
		return grpcHTTPStatus(err, http.StatusInternalServerError)
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

// GRPCHTTPStatus maps a gRPC error to an appropriate HTTP status code.
// Deprecated: use HTTPStatus for typed error support.
func GRPCHTTPStatus(err error) int {
	return grpcHTTPStatus(err, http.StatusInternalServerError)
}

// grpcHTTPStatus maps raw gRPC status codes to HTTP codes.
func grpcHTTPStatus(err error, fallback int) int {
	if err == nil {
		return http.StatusOK
	}
	st, ok := status.FromError(err)
	if !ok {
		return fallback
	}
	switch st.Code() {
	case codes.InvalidArgument, codes.FailedPrecondition:
		return http.StatusBadRequest
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.NotFound:
		return http.StatusNotFound
	case codes.Unavailable, codes.DeadlineExceeded:
		return http.StatusServiceUnavailable
	default:
		return fallback
	}
}

// LogError logs a message with request context (method, path, request ID).
func LogError(r *http.Request, format string, args ...any) {
	prefix := requestPrefix(r)
	log.Printf(prefix+format, args...)
}

// requestPrefix builds a structured log prefix from request context.
func requestPrefix(r *http.Request) string {
	if r == nil {
		return ""
	}
	var b strings.Builder
	if rid := strings.TrimSpace(r.Header.Get("X-Request-ID")); rid != "" {
		b.WriteString("request_id=")
		b.WriteString(rid)
		b.WriteByte(' ')
	}
	b.WriteString("method=")
	b.WriteString(r.Method)
	b.WriteString(" path=")
	b.WriteString(r.URL.Path)
	b.WriteByte(' ')
	return b.String()
}
