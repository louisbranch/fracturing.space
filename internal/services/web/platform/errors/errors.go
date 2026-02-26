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
	KindUnavailable  Kind = "unavailable"
	KindNotFound     Kind = "not_found"
)

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
