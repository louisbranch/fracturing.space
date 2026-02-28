package weberror

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

func TestWriteModuleErrorRendersAppErrorPageForNotFound(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/missing", nil)
	rr := httptest.NewRecorder()
	WriteModuleError(rr, req, apperrors.E(apperrors.KindNotFound, "missing"), nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	if body := rr.Body.String(); !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error state marker: %q", body)
	}
}

func TestWriteModuleErrorWritesPlainTextForBadRequest(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	rr := httptest.NewRecorder()
	WriteModuleError(rr, req, apperrors.E(apperrors.KindInvalidInput, "bad form"), nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	body := rr.Body.String()
	if !strings.Contains(body, http.StatusText(http.StatusBadRequest)) {
		t.Fatalf("body = %q, want generic bad-request message", body)
	}
	// Invariant: user-facing transport errors must not leak raw internal strings.
	if strings.Contains(body, "bad form") {
		t.Fatalf("body leaked internal error text: %q", body)
	}
}
