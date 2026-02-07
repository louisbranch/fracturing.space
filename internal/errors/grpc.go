package errors

import (
	"errors"

	"github.com/louisbranch/fracturing.space/internal/errors/i18n"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DefaultLocale is the default locale for error messages.
const DefaultLocale = "en-US"

// HandleError converts domain errors to gRPC status for client responses.
// It formats the user-facing message using the i18n catalog for the given locale,
// defaulting to en-US if the locale is empty.
func HandleError(err error, locale string) error {
	if err == nil {
		return nil
	}

	if locale == "" {
		locale = DefaultLocale
	}

	var appErr *Error
	if errors.As(err, &appErr) {
		catalog := i18n.GetCatalog(locale)
		userMsg := catalog.Format(string(appErr.Code), appErr.Metadata)
		return appErr.ToGRPCStatus(catalog.Locale(), userMsg)
	}

	// Unknown error - return internal with generic message
	return status.Error(codes.Internal, "an unexpected error occurred")
}

// GetCode extracts the error code from any error.
// Returns CodeUnknown if the error is not a domain error.
func GetCode(err error) Code {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return CodeUnknown
}

// IsCode checks if the error has the specified code.
func IsCode(err error, code Code) bool {
	return GetCode(err) == code
}

// GetMetadata extracts metadata from an error if present.
// Returns nil if the error is not a domain error or has no metadata.
func GetMetadata(err error) map[string]string {
	var e *Error
	if errors.As(err, &e) {
		return e.Metadata
	}
	return nil
}
