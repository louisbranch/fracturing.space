// Package service contains the application service layer for the AI service.
// Each service encapsulates business logic for a domain boundary, accepting
// domain inputs and returning domain outputs. Transport-layer handlers become
// thin wrappers that parse proto requests, call service methods, and convert
// results to proto responses.
package service

import (
	"errors"
	"fmt"
)

// ErrorKind classifies service errors for transport-layer mapping.
type ErrorKind int

const (
	// ErrKindInvalidArgument signals validation failures or bad input.
	ErrKindInvalidArgument ErrorKind = iota
	// ErrKindNotFound signals a missing resource.
	ErrKindNotFound
	// ErrKindAlreadyExists signals a uniqueness conflict.
	ErrKindAlreadyExists
	// ErrKindPermissionDenied signals an authorization failure.
	ErrKindPermissionDenied
	// ErrKindFailedPrecondition signals the resource state doesn't allow the operation.
	ErrKindFailedPrecondition
	// ErrKindInternal signals an unexpected infrastructure failure.
	ErrKindInternal
)

// Error is a typed service-layer error that transport handlers map to
// protocol-specific error representations (e.g., gRPC status codes).
type Error struct {
	Kind    ErrorKind
	Message string
	Cause   error
}

func (e *Error) Error() string { return e.Message }

func (e *Error) Unwrap() error { return e.Cause }

// Errorf creates a service error with the given kind and formatted message.
func Errorf(kind ErrorKind, format string, args ...any) *Error {
	return &Error{Kind: kind, Message: fmt.Sprintf(format, args...)}
}

// Wrapf creates a service error wrapping a cause.
func Wrapf(kind ErrorKind, cause error, format string, args ...any) *Error {
	return &Error{Kind: kind, Message: fmt.Sprintf(format, args...), Cause: cause}
}

// ErrorKindOf extracts the ErrorKind from an error chain, returning
// ErrKindInternal if the error is not a service Error.
func ErrorKindOf(err error) ErrorKind {
	var svcErr *Error
	if errors.As(err, &svcErr) {
		return svcErr.Kind
	}
	return ErrKindInternal
}
