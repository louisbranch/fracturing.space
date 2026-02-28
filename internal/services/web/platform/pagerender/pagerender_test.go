package pagerender

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	flashnotice "github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
)

func TestWriteModulePageRendersHTMXFragmentWithStatus(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()

	err := WriteModulePage(rr, req, nil, ModulePage{
		Title:      "Settings",
		StatusCode: http.StatusCreated,
		Fragment:   textComponent(`<section id="fragment-root">ok</section>`),
	})
	if err != nil {
		t.Fatalf("WriteModulePage() error = %v", err)
	}
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusCreated)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="fragment-root"`) {
		t.Fatalf("body missing fragment marker: %q", body)
	}
	if strings.Contains(strings.ToLower(body), "<!doctype html") || strings.Contains(strings.ToLower(body), "<html") {
		t.Fatalf("expected htmx fragment without full document wrapper")
	}
}

func TestWriteModulePageRendersFullPageWithAppShell(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	rr := httptest.NewRecorder()

	err := WriteModulePage(rr, req, nil, ModulePage{
		Title:      "Settings",
		StatusCode: http.StatusAccepted,
		Fragment:   textComponent(`<section id="fragment-root">ok</section>`),
	})
	if err != nil {
		t.Fatalf("WriteModulePage() error = %v", err)
	}
	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusAccepted)
	}
	if got := rr.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("content-type = %q, want %q", got, "text/html; charset=utf-8")
	}
	body := rr.Body.String()
	for _, marker := range []string{`id="main"`, `id="fragment-root"`} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing marker %q: %q", marker, body)
		}
	}
}

func TestWriteModulePageRendersToastFromFlashNotice(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	setFlashCookie(t, req, flashnotice.NoticeSuccess("web.settings.user_profile.notice_saved"))
	rr := httptest.NewRecorder()

	err := WriteModulePage(rr, req, nil, ModulePage{
		Title:    "Settings",
		Fragment: textComponent(`<section id="fragment-root">ok</section>`),
	})
	if err != nil {
		t.Fatalf("WriteModulePage() error = %v", err)
	}
	body := rr.Body.String()
	for _, marker := range []string{`id="app-toast"`, `Profile updated.`} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing marker %q: %q", marker, body)
		}
	}
	if !responseHasCookieName(rr, flashnotice.CookieName) {
		t.Fatalf("response missing %q clear cookie", flashnotice.CookieName)
	}
}

func TestWriteModulePageHTMXDoesNotConsumeFlashNotice(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	req.Header.Set("HX-Request", "true")
	setFlashCookie(t, req, flashnotice.NoticeSuccess("web.settings.user_profile.notice_saved"))
	rr := httptest.NewRecorder()

	err := WriteModulePage(rr, req, nil, ModulePage{
		Title:    "Settings",
		Fragment: textComponent(`<section id="fragment-root">ok</section>`),
	})
	if err != nil {
		t.Fatalf("WriteModulePage() error = %v", err)
	}
	body := rr.Body.String()
	if strings.Contains(body, `id="app-toast"`) {
		t.Fatalf("htmx body unexpectedly contains toast markup: %q", body)
	}
	if responseHasCookieName(rr, flashnotice.CookieName) {
		t.Fatalf("htmx response unexpectedly set %q cookie", flashnotice.CookieName)
	}
}

func setFlashCookie(t *testing.T, req *http.Request, notice flashnotice.Notice) {
	t.Helper()
	seed := httptest.NewRecorder()
	flashnotice.Write(seed, req, notice)
	setCookieHeader := strings.TrimSpace(seed.Header().Get("Set-Cookie"))
	if setCookieHeader == "" {
		t.Fatalf("expected flash cookie header")
	}
	cookie, err := http.ParseSetCookie(setCookieHeader)
	if err != nil {
		t.Fatalf("ParseSetCookie() error = %v", err)
	}
	req.AddCookie(cookie)
}

func responseHasCookieName(rr *httptest.ResponseRecorder, name string) bool {
	if rr == nil {
		return false
	}
	for _, cookie := range rr.Result().Cookies() {
		if cookie != nil && cookie.Name == name {
			return true
		}
	}
	return false
}

type textComponent string

func (c textComponent) Render(_ context.Context, w io.Writer) error {
	_, err := io.WriteString(w, string(c))
	return err
}
