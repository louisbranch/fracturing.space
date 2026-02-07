package errors

import (
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/status"
)

// Domain is the error domain for Fracturing.Space errors.
const Domain = "github.com/louisbranch/fracturing.space"

// Error is the domain error type with structured metadata.
type Error struct {
	Code     Code              // Machine-readable error code
	Message  string            // Internal message (for logs/telemetry)
	Metadata map[string]string // Additional context for templating
	Cause    error             // Wrapped underlying error
}

// Error implements the error interface.
func (e *Error) Error() string {
	return e.Message
}

// Unwrap returns the underlying cause for error chain traversal.
func (e *Error) Unwrap() error {
	return e.Cause
}

// Is reports whether target matches this error by code.
func (e *Error) Is(target error) bool {
	if t, ok := target.(*Error); ok {
		return e.Code == t.Code
	}
	return false
}

// New creates a simple domain error with a code and message.
func New(code Code, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// WithMetadata creates a domain error with metadata for i18n templating.
func WithMetadata(code Code, message string, metadata map[string]string) *Error {
	return &Error{
		Code:     code,
		Message:  message,
		Metadata: metadata,
	}
}

// Wrap creates a domain error that wraps an underlying cause.
func Wrap(code Code, message string, cause error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// WrapWithMetadata creates a domain error with both metadata and a cause.
func WrapWithMetadata(code Code, message string, metadata map[string]string, cause error) *Error {
	return &Error{
		Code:     code,
		Message:  message,
		Metadata: metadata,
		Cause:    cause,
	}
}

// ToGRPCStatus converts the error to a gRPC status with errdetails.
// The status message contains the internal message for logging.
// The LocalizedMessage contains the user-facing translated message.
func (e *Error) ToGRPCStatus(locale string, userMessage string) error {
	grpcCode := e.Code.GRPCCode()
	st := status.New(grpcCode, e.Message)

	// Attach structured error details
	st, err := st.WithDetails(
		&errdetails.ErrorInfo{
			Reason:   string(e.Code),
			Domain:   Domain,
			Metadata: e.Metadata,
		},
		&errdetails.LocalizedMessage{
			Locale:  locale,
			Message: userMessage,
		},
	)
	if err != nil {
		// If we can't attach details, return the basic status
		return status.New(grpcCode, e.Message).Err()
	}
	return st.Err()
}
