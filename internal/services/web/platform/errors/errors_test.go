package errors

import (
	"errors"
	"net/http"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHTTPStatusMapsKnownKinds(t *testing.T) {
	t.Parallel()

	if got := HTTPStatus(E(KindUnauthorized, "unauthorized")); got != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d, want %d", got, http.StatusUnauthorized)
	}
	if got := HTTPStatus(E(KindInvalidInput, "bad")); got != http.StatusBadRequest {
		t.Fatalf("invalid input status = %d, want %d", got, http.StatusBadRequest)
	}
}

func TestHTTPStatusDefaultsToInternalError(t *testing.T) {
	t.Parallel()

	if got := HTTPStatus(errors.New("boom")); got != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", got, http.StatusInternalServerError)
	}
}

func TestErrorStringFallsBackToKindWhenMessageEmpty(t *testing.T) {
	t.Parallel()

	err := Error{Kind: KindForbidden}
	if got := err.Error(); got != string(KindForbidden) {
		t.Fatalf("Error() = %q, want %q", got, string(KindForbidden))
	}
}

func TestHTTPStatusCoversNilAndAdditionalKinds(t *testing.T) {
	t.Parallel()

	if got := HTTPStatus(nil); got != http.StatusOK {
		t.Fatalf("HTTPStatus(nil) = %d, want %d", got, http.StatusOK)
	}
	if got := HTTPStatus(E(KindForbidden, "forbidden")); got != http.StatusForbidden {
		t.Fatalf("forbidden status = %d, want %d", got, http.StatusForbidden)
	}
	if got := HTTPStatus(E(KindUnavailable, "unavailable")); got != http.StatusServiceUnavailable {
		t.Fatalf("unavailable status = %d, want %d", got, http.StatusServiceUnavailable)
	}
	if got := HTTPStatus(E(KindNotFound, "missing")); got != http.StatusNotFound {
		t.Fatalf("not-found status = %d, want %d", got, http.StatusNotFound)
	}
	if got := HTTPStatus(E(KindUnknown, "unknown")); got != http.StatusInternalServerError {
		t.Fatalf("unknown status = %d, want %d", got, http.StatusInternalServerError)
	}
}

func TestHTTPStatusMapsGRPCErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "invalid argument", err: status.Error(codes.InvalidArgument, "invalid"), want: http.StatusBadRequest},
		{name: "unauthenticated", err: status.Error(codes.Unauthenticated, "unauthenticated"), want: http.StatusUnauthorized},
		{name: "permission denied", err: status.Error(codes.PermissionDenied, "forbidden"), want: http.StatusForbidden},
		{name: "not found", err: status.Error(codes.NotFound, "missing"), want: http.StatusNotFound},
		{name: "failed precondition", err: status.Error(codes.FailedPrecondition, "conflict"), want: http.StatusConflict},
		{name: "unavailable", err: status.Error(codes.Unavailable, "unavailable"), want: http.StatusServiceUnavailable},
		{name: "internal falls back", err: status.Error(codes.Internal, "internal"), want: http.StatusInternalServerError},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := HTTPStatus(tc.err); got != tc.want {
				t.Fatalf("HTTPStatus(err) = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestLocalizationKeyReturnsStructuredKey(t *testing.T) {
	t.Parallel()

	err := EK(KindInvalidInput, "web.settings.user_profile.error_username_required", "username must be set")
	if got := LocalizationKey(err); got != "web.settings.user_profile.error_username_required" {
		t.Fatalf("LocalizationKey(err) = %q, want %q", got, "web.settings.user_profile.error_username_required")
	}
}

func TestLocalizationKeyReturnsEmptyForUnstructuredError(t *testing.T) {
	t.Parallel()

	if got := LocalizationKey(errors.New("boom")); got != "" {
		t.Fatalf("LocalizationKey(err) = %q, want empty", got)
	}
}
