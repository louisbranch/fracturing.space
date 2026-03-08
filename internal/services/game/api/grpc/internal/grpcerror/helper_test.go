package grpcerror

import (
	"errors"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type nonRetryableTestError struct {
	err error
}

func (e nonRetryableTestError) Error() string      { return e.err.Error() }
func (e nonRetryableTestError) Unwrap() error      { return e.err }
func (e nonRetryableTestError) NonRetryable() bool { return true }

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

func TestNormalizeDomainWriteOptionsDefaults(t *testing.T) {
	options := domainwrite.Options{}
	NormalizeDomainWriteOptions(&options, NormalizeDomainWriteOptionsConfig{})

	if options.ExecuteErr == nil || options.ApplyErr == nil || options.RejectErr == nil {
		t.Fatal("expected execute/apply/reject handlers to be initialized")
	}

	execErr := options.ExecuteErr(nonRetryableTestError{err: errors.New("checkpoint failed")})
	if status.Code(execErr) != codes.FailedPrecondition {
		t.Fatalf("execute code = %s, want %s", status.Code(execErr), codes.FailedPrecondition)
	}

	applyErr := options.ApplyErr(errors.New("apply failed"))
	if status.Code(applyErr) != codes.Internal {
		t.Fatalf("apply code = %s, want %s", status.Code(applyErr), codes.Internal)
	}

	rejectErr := options.RejectErr("SOME_CODE", "rejected")
	if status.Code(rejectErr) != codes.FailedPrecondition {
		t.Fatalf("reject code = %s, want %s", status.Code(rejectErr), codes.FailedPrecondition)
	}
}

func TestNormalizeDomainWriteOptionsPreservesDomainApplyCode(t *testing.T) {
	options := domainwrite.Options{}
	NormalizeDomainWriteOptions(&options, NormalizeDomainWriteOptionsConfig{
		PreserveDomainCodeOnApply: true,
	})

	domainErr := apperrors.New(apperrors.CodeNotFound, "not found")
	if got := options.ApplyErr(domainErr); got != domainErr {
		t.Fatalf("apply err should preserve domain error instance")
	}
}

func TestApplyErrorWithDomainCodePreserveWrapsUnknown(t *testing.T) {
	handler := ApplyErrorWithDomainCodePreserve("apply event")
	err := handler(errors.New("boom"))
	if status.Code(err) != codes.Internal {
		t.Fatalf("code = %s, want %s", status.Code(err), codes.Internal)
	}
}
