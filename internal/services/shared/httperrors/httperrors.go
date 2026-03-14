// Package httperrors provides typed HTTP application errors with gRPC error
// classification and consistent status mapping shared across services.
package httperrors

import (
	stderrors "errors"
	"net/http"
	"strings"

	platformerrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	platformerrori18n "github.com/louisbranch/fracturing.space/internal/platform/errors/i18n"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
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

// Error is a typed application failure.
type Error struct {
	Kind Kind
	Key  string
	// Message carries the local error text already present at this web/admin
	// boundary. It is not treated as user-safe by default.
	Message string
	// PublicMessage carries transport-safe copy extracted from rich gRPC
	// details or an explicit fallback chosen by the caller.
	PublicMessage string
	// PublicMessageLocale preserves the locale attached to the rich gRPC
	// LocalizedMessage detail when one was provided.
	PublicMessageLocale string
	// ReasonDomain, Reason, and ReasonMetadata preserve gRPC ErrorInfo so the
	// web layer can localize platform domain errors in the active request
	// locale instead of relying on backend-emitted copy.
	ReasonDomain   string
	Reason         string
	ReasonMetadata map[string]string
}

// Error renders the human-readable message.
func (e Error) Error() string {
	if msg := strings.TrimSpace(e.Message); msg != "" {
		return msg
	}
	if msg := strings.TrimSpace(e.PublicMessage); msg != "" {
		return msg
	}
	return string(e.Kind)
}

// E builds a typed Error.
func E(kind Kind, message string) error {
	return Error{Kind: kind, Message: strings.TrimSpace(message)}
}

// EK builds a typed Error with a localization key.
func EK(kind Kind, key string, message string) error {
	return Error{
		Kind:    kind,
		Key:     strings.TrimSpace(key),
		Message: strings.TrimSpace(message),
	}
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

// PublicMessage returns the explicit transport-safe message when one was
// attached to the typed error.
func PublicMessage(err error) string {
	if err == nil {
		return ""
	}
	var appErr Error
	if !stderrors.As(err, &appErr) {
		return ""
	}
	return strings.TrimSpace(appErr.PublicMessage)
}

// ResolveRichMessage localizes preserved transport-safe error details when
// possible and otherwise returns the stored public fallback text.
func ResolveRichMessage(err error, locale string) string {
	if err == nil {
		return ""
	}
	var appErr Error
	if !stderrors.As(err, &appErr) {
		return ""
	}
	if msg := resolvePlatformReasonMessage(appErr, locale); msg != "" {
		return msg
	}
	return strings.TrimSpace(appErr.PublicMessage)
}

// GRPCStatusMapping describes how a gRPC transport failure should
// downgrade into error classification when a service-specific fallback exists.
type GRPCStatusMapping struct {
	FallbackKind    Kind
	FallbackKey     string
	FallbackMessage string
}

// MapGRPCTransportError converts gRPC transport errors into typed errors with
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
	details := extractStatusDetails(st)
	switch st.Code() {
	case codes.InvalidArgument, codes.OutOfRange:
		return mapTransportError(KindInvalidInput, mapping, details)
	case codes.FailedPrecondition, codes.AlreadyExists, codes.Aborted:
		return mapTransportError(KindConflict, mapping, details)
	case codes.Unauthenticated:
		return mapTransportError(KindUnauthorized, GRPCStatusMapping{FallbackMessage: "authentication required"}, details)
	case codes.PermissionDenied:
		return mapTransportError(KindForbidden, GRPCStatusMapping{FallbackMessage: "access denied"}, details)
	case codes.NotFound:
		return mapTransportError(KindNotFound, GRPCStatusMapping{FallbackMessage: "resource not found"}, details)
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted, codes.Canceled:
		return mapTransportError(KindUnavailable, GRPCStatusMapping{FallbackMessage: "dependency is temporarily unavailable"}, details)
	default:
		return mapTransportError(mapping.FallbackKind, mapping, details)
	}
}

// mapWithFallback maps values across transport and domain boundaries.
func mapWithFallback(mapping GRPCStatusMapping) error {
	if strings.TrimSpace(mapping.FallbackKey) != "" {
		return Error{
			Kind:          mapping.FallbackKind,
			Key:           strings.TrimSpace(mapping.FallbackKey),
			Message:       strings.TrimSpace(mapping.FallbackMessage),
			PublicMessage: strings.TrimSpace(mapping.FallbackMessage),
		}
	}
	return Error{
		Kind:          mapping.FallbackKind,
		Message:       strings.TrimSpace(mapping.FallbackMessage),
		PublicMessage: strings.TrimSpace(mapping.FallbackMessage),
	}
}

// HTTPStatus maps an error to an HTTP status code. It understands
// both typed Error values and raw gRPC status errors.
func HTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	var appErr Error
	if !stderrors.As(err, &appErr) {
		return GRPCErrorHTTPStatus(err, http.StatusInternalServerError)
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

// GRPCErrorHTTPStatus maps raw gRPC status codes to HTTP codes with a
// configurable fallback for unmapped codes.
func GRPCErrorHTTPStatus(err error, fallback int) int {
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

// grpcStatusDetails captures user-safe information preserved from rich gRPC
// status details so renderers can make locale-aware choices later.
type grpcStatusDetails struct {
	publicMessage       string
	publicMessageLocale string
	reasonDomain        string
	reason              string
	reasonMetadata      map[string]string
}

// suppressesFallbackKey reports whether renderers can already prefer richer
// transport detail over a caller's generic web localization key.
func (d grpcStatusDetails) suppressesFallbackKey() bool {
	if strings.TrimSpace(d.publicMessage) != "" {
		return true
	}
	return strings.TrimSpace(d.reason) != "" &&
		strings.TrimSpace(d.reasonDomain) == platformerrors.Domain
}

// extractStatusDetails preserves the first user-safe rich-detail values from a
// gRPC status so downstream renderers can localize or display them.
func extractStatusDetails(st *status.Status) grpcStatusDetails {
	if st == nil {
		return grpcStatusDetails{}
	}
	result := grpcStatusDetails{}
	for _, detail := range st.Details() {
		switch typed := detail.(type) {
		case *errdetails.LocalizedMessage:
			if strings.TrimSpace(result.publicMessage) == "" {
				result.publicMessage = strings.TrimSpace(typed.GetMessage())
				result.publicMessageLocale = strings.TrimSpace(typed.GetLocale())
			}
		case *errdetails.ErrorInfo:
			if strings.TrimSpace(result.reason) == "" {
				result.reasonDomain = strings.TrimSpace(typed.GetDomain())
				result.reason = strings.TrimSpace(typed.GetReason())
				result.reasonMetadata = cloneMetadata(typed.GetMetadata())
			}
		}
	}
	return result
}

// mapTransportError centralizes how typed transport errors retain both caller
// fallback policy and richer user-safe detail from gRPC statuses.
func mapTransportError(kind Kind, mapping GRPCStatusMapping, details grpcStatusDetails) error {
	publicMessage := strings.TrimSpace(details.publicMessage)
	if publicMessage == "" {
		publicMessage = strings.TrimSpace(mapping.FallbackMessage)
	}
	key := strings.TrimSpace(mapping.FallbackKey)
	if details.suppressesFallbackKey() {
		key = ""
	}
	return Error{
		Kind:                kind,
		Key:                 key,
		Message:             publicMessage,
		PublicMessage:       publicMessage,
		PublicMessageLocale: strings.TrimSpace(details.publicMessageLocale),
		ReasonDomain:        strings.TrimSpace(details.reasonDomain),
		Reason:              strings.TrimSpace(details.reason),
		ReasonMetadata:      cloneMetadata(details.reasonMetadata),
	}
}

// resolvePlatformReasonMessage re-renders shared platform domain errors in the
// active locale when ErrorInfo provides a machine-readable reason + metadata.
func resolvePlatformReasonMessage(err Error, locale string) string {
	reason := strings.TrimSpace(err.Reason)
	if reason == "" || strings.TrimSpace(err.ReasonDomain) != platformerrors.Domain {
		return ""
	}
	catalog := platformerrori18n.GetCatalog(locale)
	msg := strings.TrimSpace(catalog.Format(reason, cloneMetadata(err.ReasonMetadata)))
	if msg == "" || msg == reason {
		return ""
	}
	return msg
}

// cloneMetadata copies metadata maps so typed errors do not share mutable
// transport detail across callers or tests.
func cloneMetadata(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
