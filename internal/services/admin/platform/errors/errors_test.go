package errors

import (
	stderrors "errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestTypedErrorConstructors(t *testing.T) {
	err := E(KindNotFound, "thing missing")
	var appErr Error
	if !stderrors.As(err, &appErr) {
		t.Fatal("E() did not produce Error")
	}
	if appErr.Kind != KindNotFound || appErr.Error() != "thing missing" {
		t.Fatalf("E() = %+v", appErr)
	}

	err = EK(KindInvalidInput, "error.bad_email", "bad email")
	if !stderrors.As(err, &appErr) {
		t.Fatal("EK() did not produce Error")
	}
	if appErr.Key != "error.bad_email" || appErr.Message != "bad email" {
		t.Fatalf("EK() = %+v", appErr)
	}

	if got := (Error{Kind: KindUnknown}).Error(); got != "unknown" {
		t.Fatalf("Error{}.Error() = %q", got)
	}
}

func TestLocalizationKey(t *testing.T) {
	if got := LocalizationKey(nil); got != "" {
		t.Fatalf("LocalizationKey(nil) = %q", got)
	}
	if got := LocalizationKey(stderrors.New("plain")); got != "" {
		t.Fatalf("LocalizationKey(plain) = %q", got)
	}
	err := EK(KindInvalidInput, "error.key", "msg")
	if got := LocalizationKey(err); got != "error.key" {
		t.Fatalf("LocalizationKey() = %q", got)
	}
}

func TestHTTPStatus(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect int
	}{
		{"nil", nil, http.StatusOK},
		{"typed_not_found", E(KindNotFound, "x"), http.StatusNotFound},
		{"typed_invalid_input", E(KindInvalidInput, "x"), http.StatusBadRequest},
		{"typed_unauthorized", E(KindUnauthorized, "x"), http.StatusUnauthorized},
		{"typed_forbidden", E(KindForbidden, "x"), http.StatusForbidden},
		{"typed_conflict", E(KindConflict, "x"), http.StatusConflict},
		{"typed_unavailable", E(KindUnavailable, "x"), http.StatusServiceUnavailable},
		{"typed_unknown", E(KindUnknown, "x"), http.StatusInternalServerError},
		{"grpc_not_found", status.Error(codes.NotFound, "x"), http.StatusNotFound},
		{"grpc_internal", status.Error(codes.Internal, "x"), http.StatusInternalServerError},
		{"plain_error", stderrors.New("boom"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HTTPStatus(tt.err); got != tt.expect {
				t.Errorf("HTTPStatus() = %d, want %d", got, tt.expect)
			}
		})
	}
}

func TestMapGRPCTransportError(t *testing.T) {
	mapping := GRPCStatusMapping{
		FallbackKind:    KindInvalidInput,
		FallbackKey:     "error.fallback",
		FallbackMessage: "fallback msg",
	}

	if err := MapGRPCTransportError(nil, mapping); err != nil {
		t.Fatalf("MapGRPCTransportError(nil) = %v", err)
	}

	// Already typed — pass through.
	typed := E(KindConflict, "already typed")
	if got := MapGRPCTransportError(typed, mapping); got != typed {
		t.Fatalf("MapGRPCTransportError(typed) = %v", got)
	}

	// gRPC NotFound → KindNotFound.
	grpcErr := status.Error(codes.NotFound, "gone")
	mapped := MapGRPCTransportError(grpcErr, mapping)
	var appErr Error
	if !stderrors.As(mapped, &appErr) || appErr.Kind != KindNotFound {
		t.Fatalf("MapGRPCTransportError(NotFound) = %+v", mapped)
	}

	// gRPC InvalidArgument → fallback.
	grpcErr = status.Error(codes.InvalidArgument, "bad")
	mapped = MapGRPCTransportError(grpcErr, mapping)
	if !stderrors.As(mapped, &appErr) || appErr.Kind != KindInvalidInput || appErr.Key != "error.fallback" {
		t.Fatalf("MapGRPCTransportError(InvalidArgument) = %+v", mapped)
	}

	// gRPC Unavailable → KindUnavailable.
	grpcErr = status.Error(codes.Unavailable, "down")
	mapped = MapGRPCTransportError(grpcErr, mapping)
	if !stderrors.As(mapped, &appErr) || appErr.Kind != KindUnavailable {
		t.Fatalf("MapGRPCTransportError(Unavailable) = %+v", mapped)
	}

	// gRPC Aborted → KindConflict with message.
	grpcErr = status.Error(codes.Aborted, "conflict detail")
	mapped = MapGRPCTransportError(grpcErr, mapping)
	if !stderrors.As(mapped, &appErr) || appErr.Kind != KindConflict || appErr.Message != "conflict detail" {
		t.Fatalf("MapGRPCTransportError(Aborted) = %+v", mapped)
	}

	// Non-gRPC error → fallback.
	mapped = MapGRPCTransportError(stderrors.New("random"), mapping)
	if !stderrors.As(mapped, &appErr) || appErr.Kind != KindInvalidInput {
		t.Fatalf("MapGRPCTransportError(plain) = %+v", mapped)
	}

	// Fallback without key.
	noKey := GRPCStatusMapping{FallbackKind: KindUnknown, FallbackMessage: "oops"}
	mapped = MapGRPCTransportError(stderrors.New("x"), noKey)
	if !stderrors.As(mapped, &appErr) || appErr.Key != "" {
		t.Fatalf("MapGRPCTransportError(no key) = %+v", mapped)
	}
}

func TestRequestPrefixIncludesRequestID(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.Header.Set("X-Request-ID", "abc-123")
	prefix := requestPrefix(r)
	if prefix == "" {
		t.Fatal("expected non-empty prefix")
	}
	if !contains(prefix, "request_id=abc-123") {
		t.Errorf("prefix missing request_id: %q", prefix)
	}
	if !contains(prefix, "method=GET") {
		t.Errorf("prefix missing method: %q", prefix)
	}
	if !contains(prefix, "path=/test") {
		t.Errorf("prefix missing path: %q", prefix)
	}
}

func TestRequestPrefixNilRequest(t *testing.T) {
	if got := requestPrefix(nil); got != "" {
		t.Errorf("requestPrefix(nil) = %q, want empty", got)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
