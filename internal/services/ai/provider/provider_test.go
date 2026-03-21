package provider

import (
	"errors"
	"testing"
)

func TestNormalize(t *testing.T) {
	got, err := Normalize(" OpenAI ")
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if got != OpenAI {
		t.Fatalf("Normalize() = %q, want %q", got, OpenAI)
	}

	_, err = Normalize("other")
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("Normalize(other) error = %v, want %v", err, ErrInvalid)
	}
}
