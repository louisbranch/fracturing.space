package grpcerror

import (
	"errors"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestEnsureStatusPreservesExistingGRPCStatus(t *testing.T) {
	err := status.Error(codes.NotFound, "missing")
	got := EnsureStatus(err)
	if status.Code(got) != codes.NotFound {
		t.Fatalf("code = %s, want %s", status.Code(got), codes.NotFound)
	}
}

func TestEnsureStatusMapsDomainErrors(t *testing.T) {
	err := apperrors.New(apperrors.CodeCharacterEmptyName, "character name is required")
	got := EnsureStatus(err)
	if status.Code(got) != codes.InvalidArgument {
		t.Fatalf("code = %s, want %s", status.Code(got), codes.InvalidArgument)
	}
}

func TestEnsureStatusWrapsUnknownAsInternal(t *testing.T) {
	got := EnsureStatus(errors.New("boom"))
	if status.Code(got) != codes.Internal {
		t.Fatalf("code = %s, want %s", status.Code(got), codes.Internal)
	}
}

func TestApplyErrorWithDomainCodePreserveWrapsUnknown(t *testing.T) {
	handler := ApplyErrorWithDomainCodePreserve("apply event")
	err := handler(errors.New("boom"))
	if status.Code(err) != codes.Internal {
		t.Fatalf("code = %s, want %s", status.Code(err), codes.Internal)
	}
}
