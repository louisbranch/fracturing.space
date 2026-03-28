package httpx

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

func TestDecodeJSONStrictRejectsNilRequest(t *testing.T) {
	t.Parallel()

	var payload struct{ Name string }
	err := DecodeJSONStrict(nil, &payload, 32)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("err = %v, want %v", err, io.ErrUnexpectedEOF)
	}
}

func TestDecodeJSONStrictRejectsEmptyBody(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/auth/passkeys/register/start", http.NoBody)
	var payload struct{ Name string }
	err := DecodeJSONStrict(req, &payload, 32)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("err = %v, want %v", err, io.ErrUnexpectedEOF)
	}
}

func TestDecodeJSONStrictRejectsOversizeBody(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/auth/passkeys/register/start", strings.NewReader(`{"name":"toolong"}`))
	var payload struct {
		Name string `json:"name"`
	}
	err := DecodeJSONStrict(req, &payload, 8)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("err = %v, want %v", err, io.ErrUnexpectedEOF)
	}
}

func TestDecodeJSONStrictRejectsUnknownField(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/auth/passkeys/register/start", strings.NewReader(`{"name":"louis","extra":true}`))
	var payload struct {
		Name string `json:"name"`
	}
	err := DecodeJSONStrict(req, &payload, 128)
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("err = %v, want unknown field detail", err)
	}
}

func TestDecodeJSONStrictRejectsTrailingTokens(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/auth/passkeys/register/start", strings.NewReader(`{"name":"louis"}{"extra":true}`))
	var payload struct {
		Name string `json:"name"`
	}
	err := DecodeJSONStrict(req, &payload, 128)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("err = %v, want %v", err, io.ErrUnexpectedEOF)
	}
}

func TestDecodeJSONStrictDecodesSinglePayload(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/auth/passkeys/register/start", strings.NewReader(`{"name":" louis "}`))
	var payload struct {
		Name string `json:"name"`
	}
	if err := DecodeJSONStrict(req, &payload, 128); err != nil {
		t.Fatalf("DecodeJSONStrict() error = %v", err)
	}
	if payload.Name != " louis " {
		t.Fatalf("payload.Name = %q, want %q", payload.Name, " louis ")
	}
}

func TestDecodeJSONStrictInvalidInputMapsMalformedBodies(t *testing.T) {
	t.Parallel()

	var payload struct{ Name string }
	if status := apperrors.HTTPStatus(DecodeJSONStrictInvalidInput(nil, &payload, 32)); status != http.StatusBadRequest {
		t.Fatalf("nil request status = %d, want %d", status, http.StatusBadRequest)
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/passkeys/register/start", strings.NewReader(`{"name":`))
	if status := apperrors.HTTPStatus(DecodeJSONStrictInvalidInput(req, &payload, 32)); status != http.StatusBadRequest {
		t.Fatalf("invalid json status = %d, want %d", status, http.StatusBadRequest)
	}
}
