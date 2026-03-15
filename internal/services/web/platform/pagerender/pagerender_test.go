package pagerender

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	flashnotice "github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
	"golang.org/x/text/message"
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

func TestWriteModulePageDefaultsStatusAndNilFragment(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/app/dashboard", nil)
	rr := httptest.NewRecorder()

	err := WriteModulePage(rr, req, nil, ModulePage{
		Title: "Dashboard",
	})
	if err != nil {
		t.Fatalf("WriteModulePage() error = %v", err)
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), `id="main"`) {
		t.Fatalf("expected app shell main container in response body")
	}
}

func TestWriteModulePageInvokesResolverHooks(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/app/dashboard", nil)
	rr := httptest.NewRecorder()
	resolver := &countingResolver{}

	err := WriteModulePage(rr, req, resolver, ModulePage{
		Title:    "Dashboard",
		Fragment: textComponent(`<section id="fragment-root">ok</section>`),
	})
	if err != nil {
		t.Fatalf("WriteModulePage() error = %v", err)
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := resolver.viewerCalls.Load(); got != 1 {
		t.Fatalf("ResolveRequestViewer call count = %d, want 1", got)
	}
	if got := resolver.languageCalls.Load(); got != 1 {
		t.Fatalf("ResolveRequestLanguage call count = %d, want 1", got)
	}
}

func TestWriteModulePageAllowsNilRequest(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	err := WriteModulePage(rr, nil, nil, ModulePage{
		Title:    "Dashboard",
		Fragment: textComponent(`<section id="fragment-root">ok</section>`),
	})
	if err != nil {
		t.Fatalf("WriteModulePage() error = %v", err)
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestWriteModulePageAllowsNilWriter(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/app/dashboard", nil)
	if err := WriteModulePage(nil, req, nil, ModulePage{Title: "Dashboard"}); err != nil {
		t.Fatalf("WriteModulePage() error = %v, want nil", err)
	}
}

func TestWriteModulePageReturnsRenderError(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/app/dashboard", nil)
	rr := httptest.NewRecorder()

	err := WriteModulePage(rr, req, nil, ModulePage{
		Title:    "Dashboard",
		Fragment: brokenComponent{},
	})
	if err == nil {
		t.Fatal("WriteModulePage() error = nil, want non-nil")
	}
}

func TestWritePublicPageHandlesNilRequest(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	WritePublicPage(rr, nil, "Sign In", "desc", "en", 0, textComponent(`<section id="public-fragment">ok</section>`))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="public-fragment"`) {
		t.Fatalf("expected rendered fragment content in public page body")
	}
}

func TestWritePublicPageDefaultsStatusAndNilBody(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/discover/campaigns", nil)
	rr := httptest.NewRecorder()
	WritePublicPage(rr, req, "Discover", "desc", "en", 0, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if body := rr.Body.String(); !strings.Contains(body, `id="auth-shell"`) {
		t.Fatalf("body missing auth shell marker: %q", body)
	}
}

func TestWritePublicPageRendersToastFromFlashNotice(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	setFlashCookie(t, req, flashnotice.Notice{Kind: flashnotice.KindError, Message: "Signup expired. Please try again."})
	rr := httptest.NewRecorder()

	WritePublicPage(rr, req, "Sign In", "desc", "en", http.StatusAccepted, textComponent(`<section id="public-fragment">ok</section>`))

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusAccepted)
	}
	body := rr.Body.String()
	for _, marker := range []string{`id="app-toast"`, `Signup expired. Please try again.`, `id="public-fragment"`} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing marker %q: %q", marker, body)
		}
	}
	if !responseHasCookieName(rr, flashnotice.CookieName) {
		t.Fatalf("response missing %q clear cookie", flashnotice.CookieName)
	}
}

func TestWritePublicPageFallsBackToInternalServerErrorOnRenderFailure(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/discover/campaigns", nil)
	rr := httptest.NewRecorder()
	WritePublicPage(rr, req, "Discover", "desc", "en", http.StatusCreated, brokenComponent{})
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
	if body := rr.Body.String(); !strings.Contains(body, http.StatusText(http.StatusInternalServerError)) {
		t.Fatalf("body missing generic internal-server-error message: %q", body)
	}
}

func TestWritePublicPageAllowsNilWriter(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/discover/campaigns", nil)
	WritePublicPage(nil, req, "Discover", "desc", "en", http.StatusCreated, textComponent(`<section>ok</section>`))
}

func TestResolveFlashToastReturnsNilWhenNoNoticeExists(t *testing.T) {
	t.Parallel()

	if toast := resolveFlashToast(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil), blankLocalizer{}, "en-US"); toast != nil {
		t.Fatalf("resolveFlashToast() = %+v, want nil", toast)
	}
}

func TestResolveFlashToastFallsBackToNoticeKeyWhenLocalizationMissing(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	setFlashCookie(t, req, flashnotice.NoticeSuccess("web.notice.missing_translation"))
	rr := httptest.NewRecorder()
	toast := resolveFlashToast(rr, req, blankLocalizer{}, "en-US")
	if toast == nil {
		t.Fatalf("resolveFlashToast() = nil, want toast")
	}
	if toast.Message != "web.notice.missing_translation" {
		t.Fatalf("toast.Message = %q, want notice key fallback", toast.Message)
	}
}

func TestResolveFlashToastUsesLiteralMessageWhenKeyMissing(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/c1/invites", nil)
	setFlashCookie(t, req, flashnotice.Notice{Kind: flashnotice.KindError, Message: "User already has a pending invite in this campaign"})
	rr := httptest.NewRecorder()
	toast := resolveFlashToast(rr, req, blankLocalizer{}, "en-US")
	if toast == nil {
		t.Fatalf("resolveFlashToast() = nil, want toast")
	}
	if toast.Message != "User already has a pending invite in this campaign" {
		t.Fatalf("toast.Message = %q", toast.Message)
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

type brokenComponent struct{}

func (brokenComponent) Render(context.Context, io.Writer) error {
	return errors.New("render boom")
}

type countingResolver struct {
	viewerCalls   atomic.Int64
	languageCalls atomic.Int64
}

func (r *countingResolver) ResolveRequestViewer(_ *http.Request) module.Viewer {
	r.viewerCalls.Add(1)
	return module.Viewer{DisplayName: "Ada"}
}

func (r *countingResolver) ResolveRequestLanguage(_ *http.Request) string {
	r.languageCalls.Add(1)
	return "en"
}

type blankLocalizer struct{}

func (blankLocalizer) Sprintf(message.Reference, ...any) string { return "   " }
