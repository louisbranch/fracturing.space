package jsoninput

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

func TestDecodeStrictRejectsNilRequest(t *testing.T) {
	t.Parallel()

	var payload struct{ Name string }
	err := DecodeStrict(nil, &payload, 32)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("err = %v, want %v", err, io.ErrUnexpectedEOF)
	}
}

func TestDecodeStrictRejectsEmptyBody(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/auth/passkeys/register/start", http.NoBody)
	var payload struct{ Name string }
	err := DecodeStrict(req, &payload, 32)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("err = %v, want %v", err, io.ErrUnexpectedEOF)
	}
}

func TestDecodeStrictRejectsOversizeBody(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/auth/passkeys/register/start", strings.NewReader(`{"name":"toolong"}`))
	var payload struct {
		Name string `json:"name"`
	}
	err := DecodeStrict(req, &payload, 8)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("err = %v, want %v", err, io.ErrUnexpectedEOF)
	}
}

func TestDecodeStrictRejectsUnknownField(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/auth/passkeys/register/start", strings.NewReader(`{"name":"louis","extra":true}`))
	var payload struct {
		Name string `json:"name"`
	}
	err := DecodeStrict(req, &payload, 128)
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("err = %v, want unknown field detail", err)
	}
}

func TestDecodeStrictRejectsTrailingTokens(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/auth/passkeys/register/start", strings.NewReader(`{"name":"louis"}{"extra":true}`))
	var payload struct {
		Name string `json:"name"`
	}
	err := DecodeStrict(req, &payload, 128)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("err = %v, want %v", err, io.ErrUnexpectedEOF)
	}
}

func TestDecodeStrictDecodesSinglePayload(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/auth/passkeys/register/start", strings.NewReader(`{"name":" louis "}`))
	var payload struct {
		Name string `json:"name"`
	}
	if err := DecodeStrict(req, &payload, 128); err != nil {
		t.Fatalf("DecodeStrict() error = %v", err)
	}
	if payload.Name != " louis " {
		t.Fatalf("payload.Name = %q, want %q", payload.Name, " louis ")
	}
}

func TestDecodeStrictInvalidInputMapsMalformedBodies(t *testing.T) {
	t.Parallel()

	var payload struct{ Name string }
	if status := apperrors.HTTPStatus(DecodeStrictInvalidInput(nil, &payload, 32)); status != http.StatusBadRequest {
		t.Fatalf("nil request status = %d, want %d", status, http.StatusBadRequest)
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/passkeys/register/start", strings.NewReader(`{"name":`))
	if status := apperrors.HTTPStatus(DecodeStrictInvalidInput(req, &payload, 32)); status != http.StatusBadRequest {
		t.Fatalf("invalid json status = %d, want %d", status, http.StatusBadRequest)
	}
}
