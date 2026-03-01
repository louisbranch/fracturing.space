package engine

import (
	"errors"
	"testing"
)

func TestWrapNonRetryable(t *testing.T) {
	if err := wrapNonRetryable(nil); err != nil {
		t.Fatalf("wrapNonRetryable(nil) = %v, want nil", err)
	}

	base := errors.New("boom")
	wrapped := wrapNonRetryable(base)
	if wrapped == nil {
		t.Fatal("expected wrapped error")
	}
	if !errors.Is(wrapped, base) {
		t.Fatalf("expected wrapped error to unwrap to base")
	}
	if !IsNonRetryable(wrapped) {
		t.Fatal("expected wrapped error to be non-retryable")
	}
}
