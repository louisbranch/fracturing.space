// Package errors provides admin-specific error helpers built on top of the
// shared httperrors package.
package errors

import (
	"log"
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/shared/httperrors"
)

// Re-export shared types so existing admin callers compile without import changes.
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

// MapGRPCTransportError converts gRPC transport errors into typed errors.
var MapGRPCTransportError = httperrors.MapGRPCTransportError

// HTTPStatus maps an error to an HTTP status code.
var HTTPStatus = httperrors.HTTPStatus

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
