package engine

import "errors"

// nonRetryableError wraps an error to signal that retrying the operation
// would be harmful (e.g. duplicate event creation after a post-persist
// fold failure). Transport middleware should use IsNonRetryable to detect
// this condition and return a permanent failure instead of a retry hint.
type nonRetryableError struct {
	err error
}

func (e *nonRetryableError) Error() string { return e.err.Error() }
func (e *nonRetryableError) Unwrap() error { return e.err }

// NonRetryable returns true from IsNonRetryable checks.
func (e *nonRetryableError) NonRetryable() bool { return true }

// wrapNonRetryable marks an error as non-retryable.
func wrapNonRetryable(err error) error {
	if err == nil {
		return nil
	}
	return &nonRetryableError{err: err}
}

// IsNonRetryable returns true when the error (or any error in its chain)
// signals that the operation must not be retried. Use this in transport
// middleware to prevent accidental duplicate event creation after
// post-persist fold failures.
func IsNonRetryable(err error) bool {
	var target interface{ NonRetryable() bool }
	if errors.As(err, &target) {
		return target.NonRetryable()
	}
	return false
}
