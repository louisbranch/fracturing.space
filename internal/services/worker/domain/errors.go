package domain

import "errors"

type permanentError struct {
	cause error
}

func (e permanentError) Error() string {
	if e.cause == nil {
		return "permanent error"
	}
	return e.cause.Error()
}

func (e permanentError) Unwrap() error {
	return e.cause
}

// Permanent marks an error as non-retryable.
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return permanentError{cause: err}
}

// IsPermanent reports whether err was explicitly marked as non-retryable.
func IsPermanent(err error) bool {
	var target permanentError
	return errors.As(err, &target)
}
