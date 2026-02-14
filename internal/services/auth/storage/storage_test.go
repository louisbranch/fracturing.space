package storage

import (
	"errors"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
)

func TestErrNotFound(t *testing.T) {
	if ErrNotFound.Error() != "record not found" {
		t.Fatalf("ErrNotFound.Error() = %q", ErrNotFound.Error())
	}
	if got := apperrors.GetCode(ErrNotFound); got != apperrors.CodeNotFound {
		t.Fatalf("GetCode(ErrNotFound) = %v, want %v", got, apperrors.CodeNotFound)
	}
	if !errors.Is(ErrNotFound, apperrors.New(apperrors.CodeNotFound, "other")) {
		t.Fatal("expected ErrNotFound to match CodeNotFound")
	}
}
