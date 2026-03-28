package grpcstatus

import (
	"errors"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestInternalReturnsSanitizedInternalStatus(t *testing.T) {
	err := Internal("apply event", errors.New("db exploded"))
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status, got %T", err)
	}
	if st.Code() != codes.Internal {
		t.Fatalf("code = %s, want %s", st.Code(), codes.Internal)
	}
	if st.Message() != "apply event" {
		t.Fatalf("message = %q, want %q", st.Message(), "apply event")
	}
}

func TestApplyErrorWithDomainCodePreservePreservesStructuredDomainErrors(t *testing.T) {
	handler := ApplyErrorWithDomainCodePreserve("apply event")
	domainErr := apperrors.New(apperrors.CodeNotFound, "missing")

	if got := handler(domainErr); got != domainErr {
		t.Fatalf("expected preserved domain error instance")
	}
}

func TestApplyErrorWithDomainCodePreserveSanitizesUnknownErrors(t *testing.T) {
	handler := ApplyErrorWithDomainCodePreserve("apply event")

	err := handler(errors.New("boom"))
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status, got %T", err)
	}
	if st.Code() != codes.Internal {
		t.Fatalf("code = %s, want %s", st.Code(), codes.Internal)
	}
	if st.Message() != "apply event" {
		t.Fatalf("message = %q, want %q", st.Message(), "apply event")
	}
}
