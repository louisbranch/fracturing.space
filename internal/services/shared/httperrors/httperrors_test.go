package httperrors

import (
	"errors"
	"net/http"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestLocalizationKey(t *testing.T) {
	if key := LocalizationKey(EK(KindInvalidInput, "err.key", "invalid")); key != "err.key" {
		t.Fatalf("expected localization key err.key, got %q", key)
	}
	if key := LocalizationKey(errors.New("plain")); key != "" {
		t.Fatalf("expected empty key for plain error, got %q", key)
	}
}

func TestMapGRPCTransportError(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		mapping GRPCStatusMapping
		want    Kind
	}{
		{
			name: "permission denied maps to forbidden",
			err:  status.Error(codes.PermissionDenied, "denied"),
			mapping: GRPCStatusMapping{
				FallbackKind:    KindUnavailable,
				FallbackMessage: "fallback",
			},
			want: KindForbidden,
		},
		{
			name: "invalid argument falls back",
			err:  status.Error(codes.InvalidArgument, "invalid"),
			mapping: GRPCStatusMapping{
				FallbackKind:    KindInvalidInput,
				FallbackMessage: "fallback invalid",
			},
			want: KindInvalidInput,
		},
		{
			name: "unknown status falls back",
			err:  status.Error(codes.Unknown, "unknown"),
			mapping: GRPCStatusMapping{
				FallbackKind:    KindUnavailable,
				FallbackMessage: "fallback unavailable",
			},
			want: KindUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapped := MapGRPCTransportError(tt.err, tt.mapping)
			var appErr Error
			if !errors.As(mapped, &appErr) {
				t.Fatalf("expected typed app error, got %T", mapped)
			}
			if appErr.Kind != tt.want {
				t.Fatalf("expected kind %q, got %q", tt.want, appErr.Kind)
			}
		})
	}
}

func TestHTTPStatus(t *testing.T) {
	if got := HTTPStatus(E(KindUnauthorized, "auth required")); got != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", got)
	}
	if got := HTTPStatus(E(KindConflict, "conflict")); got != http.StatusConflict {
		t.Fatalf("expected 409, got %d", got)
	}
	if got := HTTPStatus(status.Error(codes.NotFound, "missing")); got != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", got)
	}
}

func TestGRPCErrorHTTPStatus(t *testing.T) {
	if got := GRPCErrorHTTPStatus(status.Error(codes.Unavailable, "down"), http.StatusInternalServerError); got != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", got)
	}
	if got := GRPCErrorHTTPStatus(errors.New("plain"), http.StatusTeapot); got != http.StatusTeapot {
		t.Fatalf("expected fallback 418, got %d", got)
	}
}
