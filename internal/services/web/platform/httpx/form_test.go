package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
)

func TestParseFormInvalidInputRejectsNilRequest(t *testing.T) {
	t.Parallel()

	err := ParseFormInvalidInput(nil, "error.web.message.failed_to_parse_form", "failed to parse form")
	if status := apperrors.HTTPStatus(err); status != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", status, http.StatusBadRequest)
	}
}

func TestParseFormInvalidInputRejectsMalformedBody(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/app/settings/profile", strings.NewReader("%zz"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	err := ParseFormInvalidInput(req, "error.web.message.failed_to_parse_form", "failed to parse form")
	if status := apperrors.HTTPStatus(err); status != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", status, http.StatusBadRequest)
	}
}

func TestParseFormInvalidInputParsesValidBody(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/app/settings/profile", strings.NewReader("name=louis"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if err := ParseFormInvalidInput(req, "error.web.message.failed_to_parse_form", "failed to parse form"); err != nil {
		t.Fatalf("ParseFormInvalidInput() error = %v", err)
	}
	if got := req.PostFormValue("name"); got != "louis" {
		t.Fatalf("PostFormValue(name) = %q, want %q", got, "louis")
	}
}

func TestParseFormOrRedirectErrorNoticeRedirectsOnMalformedBody(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/app/campaigns", strings.NewReader("%zz"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	ok := ParseFormOrRedirectErrorNotice(rr, req, "error.web.message.failed_to_parse_form", "/app/campaigns/create")
	if ok {
		t.Fatal("ok = true, want false")
	}
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if location := rr.Header().Get("Location"); location != "/app/campaigns/create" {
		t.Fatalf("Location = %q, want %q", location, "/app/campaigns/create")
	}
	foundFlashCookie := false
	for _, cookie := range rr.Result().Cookies() {
		if cookie.Name == flash.CookieName {
			foundFlashCookie = true
			break
		}
	}
	if !foundFlashCookie {
		t.Fatalf("flash cookie %q missing", flash.CookieName)
	}
}

func TestParseFormOrRedirectErrorNoticeParsesValidBody(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/app/campaigns", strings.NewReader("name=louis"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	ok := ParseFormOrRedirectErrorNotice(rr, req, "error.web.message.failed_to_parse_form", "/app/campaigns/create")
	if !ok {
		t.Fatal("ok = false, want true")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := req.PostFormValue("name"); got != "louis" {
		t.Fatalf("PostFormValue(name) = %q, want %q", got, "louis")
	}
}
