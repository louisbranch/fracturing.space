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
	if got := HTTPStatus(E(KindConflict, "conflict")); got != http.StatusConflict {
		t.Fatalf("conflict status = %d, want %d", got, http.StatusConflict)
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

func TestMapGRPCTransportErrorMapsGrpcStatusToKinds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		err                 error
		fallbackKind        Kind
		fallbackKey         string
		fallbackMessage     string
		wantKind            Kind
		wantLocalizationKey string
		wantStatus          int
	}{
		{
			name:                "invalid argument falls back to provided policy",
			err:                 status.Error(codes.InvalidArgument, "invalid payload"),
			fallbackKind:        KindInvalidInput,
			fallbackKey:         "public.error.invalid_input",
			fallbackMessage:     "invalid input",
			wantKind:            KindInvalidInput,
			wantLocalizationKey: "public.error.invalid_input",
			wantStatus:          http.StatusBadRequest,
		},
		{
			name:            "unauthenticated maps to unauthorized",
			err:             status.Error(codes.Unauthenticated, "expired token"),
			fallbackKind:    KindInvalidInput,
			fallbackMessage: "fallback invalid",
			wantKind:        KindUnauthorized,
			wantStatus:      http.StatusUnauthorized,
		},
		{
			name:            "permission denied maps to forbidden",
			err:             status.Error(codes.PermissionDenied, "forbidden"),
			fallbackKind:    KindInvalidInput,
			fallbackMessage: "fallback invalid",
			wantKind:        KindForbidden,
			wantStatus:      http.StatusForbidden,
		},
		{
			name:            "not found maps to not found",
			err:             status.Error(codes.NotFound, "missing"),
			fallbackKind:    KindInvalidInput,
			fallbackMessage: "fallback invalid",
			wantKind:        KindNotFound,
			wantStatus:      http.StatusNotFound,
		},
		{
			name:            "unavailable maps to unavailable",
			err:             status.Error(codes.Unavailable, "backend down"),
			fallbackKind:    KindInvalidInput,
			fallbackMessage: "fallback invalid",
			wantKind:        KindUnavailable,
			wantStatus:      http.StatusServiceUnavailable,
		},
		{
			name:            "non-grpc errors fallback to policy",
			err:             errors.New("backend exploded"),
			fallbackKind:    KindUnknown,
			fallbackMessage: "generic fallback",
			wantKind:        KindUnknown,
			wantStatus:      http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := MapGRPCTransportError(tc.err, GRPCStatusMapping{
				FallbackKind:    tc.fallbackKind,
				FallbackKey:     tc.fallbackKey,
				FallbackMessage: tc.fallbackMessage,
			})
			if err == nil {
				t.Fatalf("MapGRPCTransportError() = nil")
			}
			if got := HTTPStatus(err); got != tc.wantStatus {
				t.Fatalf("HTTPStatus(err) = %d, want %d", got, tc.wantStatus)
			}
			var e Error
			if !errors.As(err, &e) {
				t.Fatalf("error type = %T, want %T", err, Error{})
			}
			if e.Kind != tc.wantKind {
				t.Fatalf("Kind = %q, want %q", e.Kind, tc.wantKind)
			}
			if tc.wantLocalizationKey != "" {
				if got := LocalizationKey(err); got != tc.wantLocalizationKey {
					t.Fatalf("LocalizationKey(err) = %q, want %q", got, tc.wantLocalizationKey)
				}
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

func TestMapGRPCTransportErrorPassesThroughAppErrors(t *testing.T) {
	t.Parallel()

	input := E(KindForbidden, "forbidden")
	got := MapGRPCTransportError(input, GRPCStatusMapping{
		FallbackKind: KindInvalidInput,
	})
	if got != input {
		t.Fatalf("MapGRPCTransportError() = %#v, want %#v", got, input)
	}
}
