package httperrors

import (
	"errors"
	"net/http"
	"testing"

	platformerrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
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

func TestMapGRPCTransportErrorPreservesRichDetailsAndDropsGenericFallbackKey(t *testing.T) {
	st := status.New(codes.AlreadyExists, "internal invite detail")
	rich, err := st.WithDetails(
		&errdetails.ErrorInfo{
			Reason: string(platformerrors.CodeInviteRecipientAlreadyInvited),
			Domain: platformerrors.Domain,
			Metadata: map[string]string{
				"CampaignID": "campaign-1",
			},
		},
		&errdetails.LocalizedMessage{
			Locale:  "pt-BR",
			Message: "Mensagem em portugues",
		},
	)
	if err != nil {
		t.Fatalf("WithDetails() error = %v", err)
	}

	mapped := MapGRPCTransportError(rich.Err(), GRPCStatusMapping{
		FallbackKind:    KindUnknown,
		FallbackKey:     "error.web.message.failed_to_create_invite",
		FallbackMessage: "failed to create invite",
	})
	var appErr Error
	if !errors.As(mapped, &appErr) {
		t.Fatalf("expected typed app error, got %T", mapped)
	}
	if appErr.Kind != KindConflict {
		t.Fatalf("Kind = %q, want %q", appErr.Kind, KindConflict)
	}
	if appErr.Key != "" {
		t.Fatalf("Key = %q, want empty when rich details exist", appErr.Key)
	}
	if got := appErr.PublicMessage; got != "Mensagem em portugues" {
		t.Fatalf("PublicMessage = %q, want %q", got, "Mensagem em portugues")
	}
	if got := appErr.PublicMessageLocale; got != "pt-BR" {
		t.Fatalf("PublicMessageLocale = %q, want %q", got, "pt-BR")
	}
	if got := ResolveRichMessage(mapped, "en-US"); got != "User already has a pending invite in this campaign" {
		t.Fatalf("ResolveRichMessage(en-US) = %q", got)
	}
}

func TestMapGRPCTransportErrorKeepsFallbackKeyForUnrenderableErrorInfo(t *testing.T) {
	st := status.New(codes.AlreadyExists, "internal invite detail")
	rich, err := st.WithDetails(&errdetails.ErrorInfo{
		Reason: "OTHER_DOMAIN_REASON",
		Domain: "other.example",
	})
	if err != nil {
		t.Fatalf("WithDetails() error = %v", err)
	}

	mapped := MapGRPCTransportError(rich.Err(), GRPCStatusMapping{
		FallbackKind:    KindUnknown,
		FallbackKey:     "error.web.message.failed_to_create_invite",
		FallbackMessage: "failed to create invite",
	})
	var appErr Error
	if !errors.As(mapped, &appErr) {
		t.Fatalf("expected typed app error, got %T", mapped)
	}
	if appErr.Key != "error.web.message.failed_to_create_invite" {
		t.Fatalf("Key = %q, want fallback key preserved", appErr.Key)
	}
}

func TestPublicMessageReturnsOnlyExplicitSafeMessage(t *testing.T) {
	if got := PublicMessage(E(KindInvalidInput, "unsafe local detail")); got != "" {
		t.Fatalf("PublicMessage(local error) = %q, want empty", got)
	}

	err := Error{Kind: KindConflict, PublicMessage: "safe transport copy"}
	if got := PublicMessage(err); got != "safe transport copy" {
		t.Fatalf("PublicMessage(rich error) = %q, want %q", got, "safe transport copy")
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
