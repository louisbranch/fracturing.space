package httperrors

import (
	"errors"
	"net/http"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestErrorFormattingAndFallbackHelpers(t *testing.T) {
	t.Parallel()

	if got := (Error{Kind: KindConflict, Message: " local detail "}).Error(); got != "local detail" {
		t.Fatalf("Error.Error(message) = %q, want %q", got, "local detail")
	}
	if got := (Error{Kind: KindConflict, PublicMessage: " public detail "}).Error(); got != "public detail" {
		t.Fatalf("Error.Error(public) = %q, want %q", got, "public detail")
	}
	if got := (Error{Kind: KindForbidden}).Error(); got != string(KindForbidden) {
		t.Fatalf("Error.Error(kind) = %q, want %q", got, KindForbidden)
	}

	withKey := mapWithFallback(GRPCStatusMapping{
		FallbackKind:    KindInvalidInput,
		FallbackKey:     "error.key",
		FallbackMessage: "fallback detail",
	})
	var appErr Error
	if !errors.As(withKey, &appErr) {
		t.Fatalf("mapWithFallback(with key) type = %T, want Error", withKey)
	}
	if appErr.Key != "error.key" || appErr.PublicMessage != "fallback detail" {
		t.Fatalf("mapWithFallback(with key) = %#v", appErr)
	}

	withoutKey := mapWithFallback(GRPCStatusMapping{
		FallbackKind:    KindUnavailable,
		FallbackMessage: "downstream unavailable",
	})
	if !errors.As(withoutKey, &appErr) {
		t.Fatalf("mapWithFallback(no key) type = %T, want Error", withoutKey)
	}
	if appErr.Key != "" || appErr.Kind != KindUnavailable {
		t.Fatalf("mapWithFallback(no key) = %#v", appErr)
	}

	if got := ResolveRichMessage(E(KindInvalidInput, "unsafe"), "en-US"); got != "" {
		t.Fatalf("ResolveRichMessage(plain) = %q, want empty", got)
	}
	if got := ResolveRichMessage(Error{Kind: KindConflict, PublicMessage: "safe fallback"}, "en-US"); got != "safe fallback" {
		t.Fatalf("ResolveRichMessage(fallback public) = %q, want %q", got, "safe fallback")
	}
}

func TestTransportStatusMappingBranches(t *testing.T) {
	t.Parallel()

	mapped := MapGRPCTransportError(errors.New("plain"), GRPCStatusMapping{
		FallbackKind:    KindUnknown,
		FallbackKey:     "error.transport",
		FallbackMessage: "transport failed",
	})
	var appErr Error
	if !errors.As(mapped, &appErr) {
		t.Fatalf("MapGRPCTransportError(plain) type = %T, want Error", mapped)
	}
	if appErr.Kind != KindUnknown || appErr.Key != "error.transport" {
		t.Fatalf("MapGRPCTransportError(plain) = %#v", appErr)
	}

	preMapped := Error{Kind: KindForbidden, Message: "already typed"}
	got := MapGRPCTransportError(preMapped, GRPCStatusMapping{FallbackKind: KindUnknown})
	var preserved Error
	if !errors.As(got, &preserved) {
		t.Fatalf("MapGRPCTransportError(pretyped) type = %T, want Error", got)
	}
	if preserved.Kind != preMapped.Kind || preserved.Message != preMapped.Message || preserved.Key != "" || preserved.PublicMessage != "" {
		t.Fatalf("MapGRPCTransportError(pretyped) = %#v, want preserved typed error", preserved)
	}

	for _, tc := range []struct {
		name string
		err  error
		want Kind
	}{
		{name: "failed precondition", err: status.Error(codes.FailedPrecondition, "bad state"), want: KindConflict},
		{name: "already exists", err: status.Error(codes.AlreadyExists, "exists"), want: KindConflict},
		{name: "aborted", err: status.Error(codes.Aborted, "aborted"), want: KindConflict},
		{name: "unauthenticated", err: status.Error(codes.Unauthenticated, "auth"), want: KindUnauthorized},
		{name: "not found", err: status.Error(codes.NotFound, "missing"), want: KindNotFound},
		{name: "deadline exceeded", err: status.Error(codes.DeadlineExceeded, "slow"), want: KindUnavailable},
		{name: "resource exhausted", err: status.Error(codes.ResourceExhausted, "busy"), want: KindUnavailable},
		{name: "canceled", err: status.Error(codes.Canceled, "cancel"), want: KindUnavailable},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mapped := MapGRPCTransportError(tc.err, GRPCStatusMapping{
				FallbackKind:    KindUnknown,
				FallbackMessage: "fallback",
			})
			var appErr Error
			if !errors.As(mapped, &appErr) {
				t.Fatalf("MapGRPCTransportError(%s) type = %T, want Error", tc.name, mapped)
			}
			if appErr.Kind != tc.want {
				t.Fatalf("MapGRPCTransportError(%s) kind = %q, want %q", tc.name, appErr.Kind, tc.want)
			}
		})
	}
}

func TestHTTPStatusBranches(t *testing.T) {
	t.Parallel()

	if got := HTTPStatus(nil); got != http.StatusOK {
		t.Fatalf("HTTPStatus(nil) = %d, want %d", got, http.StatusOK)
	}
	if got := HTTPStatus(E(KindInvalidInput, "invalid")); got != http.StatusBadRequest {
		t.Fatalf("HTTPStatus(invalid) = %d, want %d", got, http.StatusBadRequest)
	}
	if got := HTTPStatus(E(KindForbidden, "forbidden")); got != http.StatusForbidden {
		t.Fatalf("HTTPStatus(forbidden) = %d, want %d", got, http.StatusForbidden)
	}
	if got := HTTPStatus(E(KindUnavailable, "down")); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(unavailable) = %d, want %d", got, http.StatusServiceUnavailable)
	}
	if got := HTTPStatus(E(KindNotFound, "missing")); got != http.StatusNotFound {
		t.Fatalf("HTTPStatus(not found) = %d, want %d", got, http.StatusNotFound)
	}
	if got := HTTPStatus(E(KindUnknown, "boom")); got != http.StatusInternalServerError {
		t.Fatalf("HTTPStatus(unknown) = %d, want %d", got, http.StatusInternalServerError)
	}

	if got := GRPCErrorHTTPStatus(nil, http.StatusTeapot); got != http.StatusOK {
		t.Fatalf("GRPCErrorHTTPStatus(nil) = %d, want %d", got, http.StatusOK)
	}
	if got := GRPCErrorHTTPStatus(status.Error(codes.InvalidArgument, "invalid"), http.StatusTeapot); got != http.StatusBadRequest {
		t.Fatalf("GRPCErrorHTTPStatus(invalid) = %d, want %d", got, http.StatusBadRequest)
	}
	if got := GRPCErrorHTTPStatus(status.Error(codes.PermissionDenied, "forbidden"), http.StatusTeapot); got != http.StatusForbidden {
		t.Fatalf("GRPCErrorHTTPStatus(permission denied) = %d, want %d", got, http.StatusForbidden)
	}
	if got := GRPCErrorHTTPStatus(status.Error(codes.Unknown, "unknown"), http.StatusTeapot); got != http.StatusTeapot {
		t.Fatalf("GRPCErrorHTTPStatus(unknown) = %d, want fallback %d", got, http.StatusTeapot)
	}
}
