package grpcerror

import (
	"context"
	"errors"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	errori18n "github.com/louisbranch/fracturing.space/internal/platform/errors/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
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

func TestHandleDomainErrorContextUsesLocaleFromContext(t *testing.T) {
	const locale = "x-test-locale"
	errori18n.RegisterCatalog(locale, errori18n.NewCatalog(locale, map[string]string{
		string(apperrors.CodeCharacterEmptyName): "nome traduzido",
	}))

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		localeHeader, locale,
	))
	err := apperrors.New(apperrors.CodeCharacterEmptyName, "character name is required")

	got := HandleDomainErrorContext(ctx, err)
	st, ok := status.FromError(got)
	if !ok {
		t.Fatalf("expected gRPC status, got %T", got)
	}
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("code = %s, want %s", st.Code(), codes.InvalidArgument)
	}
	for _, detail := range st.Details() {
		localized, ok := detail.(*errdetails.LocalizedMessage)
		if !ok {
			continue
		}
		if localized.Locale != locale {
			t.Fatalf("localized locale = %q, want %q", localized.Locale, locale)
		}
		if localized.Message != "nome traduzido" {
			t.Fatalf("localized message = %q, want translated locale message", localized.Message)
		}
		return
	}
	t.Fatal("expected localized message details")
}

func TestLookupErrorContextOverridesNotFoundMessage(t *testing.T) {
	got := LookupErrorContext(context.Background(), storage.ErrNotFound, "load thing", "thing not found")
	if status.Code(got) != codes.NotFound {
		t.Fatalf("code = %s, want %s", status.Code(got), codes.NotFound)
	}
	if status.Convert(got).Message() != "thing not found" {
		t.Fatalf("message = %q, want %q", status.Convert(got).Message(), "thing not found")
	}
}

func TestLookupErrorContextSanitizesUnknownErrors(t *testing.T) {
	got := LookupErrorContext(context.Background(), errors.New("boom"), "load thing", "thing not found")
	if status.Code(got) != codes.Internal {
		t.Fatalf("code = %s, want %s", status.Code(got), codes.Internal)
	}
	if status.Convert(got).Message() != "load thing" {
		t.Fatalf("message = %q, want %q", status.Convert(got).Message(), "load thing")
	}
}

func TestOptionalLookupErrorContextIgnoresNotFound(t *testing.T) {
	if err := OptionalLookupErrorContext(context.Background(), storage.ErrNotFound, "load thing"); err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
}

func TestOptionalLookupErrorContextPreservesOtherStatus(t *testing.T) {
	err := OptionalLookupErrorContext(context.Background(), status.Error(codes.FailedPrecondition, "blocked"), "load thing")
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("code = %s, want %s", status.Code(err), codes.FailedPrecondition)
	}
}

func TestApplyErrorWithDomainCodePreserveWrapsUnknown(t *testing.T) {
	handler := ApplyErrorWithDomainCodePreserve("apply event")
	err := handler(errors.New("boom"))
	if status.Code(err) != codes.Internal {
		t.Fatalf("code = %s, want %s", status.Code(err), codes.Internal)
	}
}
