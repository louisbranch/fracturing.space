package domain

import (
	"errors"
	"testing"
)

func TestPermanent_Nil(t *testing.T) {
	t.Parallel()

	if got := Permanent(nil); got != nil {
		t.Fatalf("Permanent(nil) = %v, want nil", got)
	}
}

func TestPermanentError_DelegatesToCause(t *testing.T) {
	t.Parallel()

	cause := errors.New("boom")
	err := Permanent(cause)
	if err == nil {
		t.Fatal("expected permanent error")
	}
	if got := err.Error(); got != "boom" {
		t.Fatalf("Error() = %q, want %q", got, "boom")
	}
	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is(%v, %v) = false, want true", err, cause)
	}
}

func TestPermanentError_DefaultMessage(t *testing.T) {
	t.Parallel()

	err := permanentError{}
	if got := err.Error(); got != "permanent error" {
		t.Fatalf("Error() = %q, want %q", got, "permanent error")
	}
	if err.Unwrap() != nil {
		t.Fatalf("Unwrap() = %v, want nil", err.Unwrap())
	}
}
