package grpcerror

import (
	"context"
	"errors"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	grpcstatus "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcstatus"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	grpcmetadata "google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	localeHeader  = "x-fracturing-space-locale"
	defaultLocale = "en-US"
)

// Internal logs the full error server-side and returns a sanitized gRPC
// Internal status that does not expose implementation details to clients.
func Internal(msg string, err error) error {
	return grpcstatus.Internal(msg, err)
}

// HandleDomainError maps domain errors through the structured app error system
// using the default locale. Prefer HandleDomainErrorContext when a request
// context is available so user-facing messages follow the caller's locale.
func HandleDomainError(err error) error {
	return apperrors.HandleError(err, apperrors.DefaultLocale)
}

// HandleDomainErrorContext maps domain errors through the structured app error
// system using the caller locale extracted from the request context.
func HandleDomainErrorContext(ctx context.Context, err error) error {
	return HandleDomainErrorLocale(err, localeFromContext(ctx))
}

// HandleDomainErrorLocale maps domain errors through the structured app error
// system using the provided locale string. Callers with a request context
// should extract the locale via grpcmeta.LocaleFromContext before calling.
func HandleDomainErrorLocale(err error, locale string) error {
	return apperrors.HandleError(err, locale)
}

// LookupError maps storage/domain lookup failures to stable gRPC statuses.
// A caller-supplied not-found message overrides the generic storage message
// while preserving codes.NotFound; other structured domain errors preserve
// their semantic gRPC code, and unknown failures are sanitized as Internal.
func LookupError(err error, internalMessage, notFoundMessage string) error {
	return lookupError(nil, err, internalMessage, notFoundMessage)
}

// LookupErrorContext is the request-aware version of LookupError and should be
// preferred when a handler context is available so structured domain errors
// keep the caller locale for localized details.
func LookupErrorContext(ctx context.Context, err error, internalMessage, notFoundMessage string) error {
	return lookupError(ctx, err, internalMessage, notFoundMessage)
}

// OptionalLookupError treats not-found lookups as absent data and returns nil.
// Other structured domain errors preserve their semantic gRPC code, and
// unknown failures are sanitized as Internal.
func OptionalLookupError(err error, internalMessage string) error {
	return optionalLookupError(nil, err, internalMessage)
}

// OptionalLookupErrorContext is the request-aware version of
// OptionalLookupError and should be preferred when a handler context is
// available so structured domain errors keep the caller locale.
func OptionalLookupErrorContext(ctx context.Context, err error, internalMessage string) error {
	return optionalLookupError(ctx, err, internalMessage)
}

func localeFromContext(ctx context.Context) string {
	if ctx == nil {
		return defaultLocale
	}
	md, ok := grpcmetadata.FromIncomingContext(ctx)
	if !ok {
		return defaultLocale
	}
	for key, values := range md {
		if !strings.EqualFold(key, localeHeader) {
			continue
		}
		for _, value := range values {
			trimmed := strings.TrimSpace(value)
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return defaultLocale
}

func lookupError(ctx context.Context, err error, internalMessage, notFoundMessage string) error {
	if err == nil {
		return nil
	}
	if _, ok := status.FromError(err); ok {
		return err
	}
	if errors.Is(err, storage.ErrNotFound) && strings.TrimSpace(notFoundMessage) != "" {
		return status.Error(codes.NotFound, notFoundMessage)
	}
	if apperrors.GetCode(err) != apperrors.CodeUnknown {
		if ctx != nil {
			return HandleDomainErrorContext(ctx, err)
		}
		return HandleDomainError(err)
	}
	if strings.TrimSpace(internalMessage) == "" {
		internalMessage = "lookup failed"
	}
	return Internal(internalMessage, err)
}

func optionalLookupError(ctx context.Context, err error, internalMessage string) error {
	if err == nil {
		return nil
	}
	if st, ok := status.FromError(err); ok {
		if st.Code() == codes.NotFound {
			return nil
		}
		return err
	}
	if errors.Is(err, storage.ErrNotFound) || apperrors.GetCode(err) == apperrors.CodeNotFound {
		return nil
	}
	if apperrors.GetCode(err) != apperrors.CodeUnknown {
		if ctx != nil {
			return HandleDomainErrorContext(ctx, err)
		}
		return HandleDomainError(err)
	}
	if strings.TrimSpace(internalMessage) == "" {
		internalMessage = "lookup failed"
	}
	return Internal(internalMessage, err)
}

// EnsureStatus guarantees transport boundaries always return gRPC status
// errors, preserving structured domain code mappings when present.
func EnsureStatus(err error) error {
	if _, ok := status.FromError(err); ok {
		return err
	}
	if apperrors.GetCode(err) != apperrors.CodeUnknown {
		return HandleDomainError(err)
	}
	return status.Error(codes.Internal, err.Error())
}

// ApplyErrorWithDomainCodePreserve preserves structured domain errors and wraps
// unknown errors in a gRPC internal status for apply callback flows.
func ApplyErrorWithDomainCodePreserve(message string) func(error) error {
	return grpcstatus.ApplyErrorWithDomainCodePreserve(message)
}
