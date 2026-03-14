package modulehandler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestresolver"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"golang.org/x/text/language"
)

func testPrincipalResolver(
	resolveUserID module.ResolveUserID,
	resolveLanguage module.ResolveLanguage,
	resolveViewer module.ResolveViewer,
) requestresolver.Principal {
	return requestresolver.NewPrincipal(nil, nil, resolveUserID, resolveLanguage, resolveViewer)
}

func TestNewBaseExtractsResolvers(t *testing.T) {
	t.Parallel()

	resolveUserID := func(*http.Request) string { return "user-1" }
	resolveLanguage := func(*http.Request) string { return "en" }
	resolveViewer := func(*http.Request) module.Viewer { return module.Viewer{DisplayName: "Test"} }

	base := NewBase(resolveUserID, resolveLanguage, resolveViewer)
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	if got := base.RequestUserID(r); got != "user-1" {
		t.Fatalf("RequestUserID() = %q, want %q", got, "user-1")
	}
	if got := base.ResolveRequestLanguage(r); got != "en" {
		t.Fatalf("ResolveRequestLanguage() = %q, want %q", got, "en")
	}
	if got := base.ResolveRequestViewer(r); got.DisplayName != "Test" {
		t.Fatalf("ResolveRequestViewer() = %+v, want DisplayName=Test", got)
	}
}

func TestNewBaseReturnsZeroValuesWhenResolversUnset(t *testing.T) {
	t.Parallel()

	base := NewBase(nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	if got := base.RequestUserID(req); got != "" {
		t.Fatalf("RequestUserID() = %q, want empty", got)
	}
	if got := base.ResolveRequestLanguage(req); got != "" {
		t.Fatalf("ResolveRequestLanguage() = %q, want empty", got)
	}
	if got := base.ResolveRequestViewer(req); got != (module.Viewer{}) {
		t.Fatalf("ResolveRequestViewer() = %+v, want zero Viewer", got)
	}
}

func TestNewBaseFromPrincipalExtractsResolvers(t *testing.T) {
	t.Parallel()

	base := NewBaseFromPrincipal(testPrincipalResolver(
		func(*http.Request) string { return "user-2" },
		func(*http.Request) string { return "fr" },
		func(*http.Request) module.Viewer { return module.Viewer{DisplayName: "Grace"} },
	))
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	if got := base.RequestUserID(r); got != "user-2" {
		t.Fatalf("RequestUserID() = %q, want %q", got, "user-2")
	}
	if got := base.ResolveRequestLanguage(r); got != "fr" {
		t.Fatalf("ResolveRequestLanguage() = %q, want %q", got, "fr")
	}
	if got := base.ResolveRequestViewer(r).DisplayName; got != "Grace" {
		t.Fatalf("ResolveRequestViewer().DisplayName = %q, want %q", got, "Grace")
	}
}

func TestNewBaseFromPrincipalReturnsZeroValuesWhenNil(t *testing.T) {
	t.Parallel()

	base := NewBaseFromPrincipal(nil)
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	if got := base.RequestUserID(r); got != "" {
		t.Fatalf("RequestUserID() = %q, want empty", got)
	}
	if got := base.ResolveRequestLanguage(r); got != "" {
		t.Fatalf("ResolveRequestLanguage() = %q, want empty", got)
	}
	if got := base.ResolveRequestViewer(r); got != (module.Viewer{}) {
		t.Fatalf("ResolveRequestViewer() = %+v, want zero Viewer", got)
	}
}

func TestResolveRequestViewerDelegatesToResolver(t *testing.T) {
	t.Parallel()

	want := module.Viewer{DisplayName: "Test"}
	base := NewBaseFromPrincipal(testPrincipalResolver(nil, nil, func(*http.Request) module.Viewer { return want }))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := base.ResolveRequestViewer(r); got != want {
		t.Fatalf("ResolveRequestViewer() = %+v, want %+v", got, want)
	}
}

func TestResolveRequestViewerReturnsZeroWhenNil(t *testing.T) {
	t.Parallel()

	base := NewBaseFromPrincipal(testPrincipalResolver(nil, nil, nil))
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := base.ResolveRequestViewer(r); got != (module.Viewer{}) {
		t.Fatalf("ResolveRequestViewer() = %+v, want zero Viewer", got)
	}
}

func TestResolveRequestLanguageDelegatesToResolver(t *testing.T) {
	t.Parallel()

	base := NewBaseFromPrincipal(testPrincipalResolver(nil, func(*http.Request) string { return "en" }, nil))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := base.ResolveRequestLanguage(r); got != "en" {
		t.Fatalf("ResolveRequestLanguage() = %q, want %q", got, "en")
	}
}

func TestResolveRequestLanguageReturnsEmptyWhenNil(t *testing.T) {
	t.Parallel()

	base := NewBaseFromPrincipal(testPrincipalResolver(nil, nil, nil))
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := base.ResolveRequestLanguage(r); got != "" {
		t.Fatalf("ResolveRequestLanguage() = %q, want empty", got)
	}
}

func TestRequestUserIDReturnsEmpty(t *testing.T) {
	t.Parallel()

	t.Run("nil request", func(t *testing.T) {
		base := NewBaseFromPrincipal(testPrincipalResolver(func(*http.Request) string { return "user-1" }, nil, nil))
		if got := base.RequestUserID(nil); got != "" {
			t.Fatalf("expected empty, got %q", got)
		}
	})

	t.Run("nil resolver", func(t *testing.T) {
		base := NewBaseFromPrincipal(testPrincipalResolver(nil, nil, nil))
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		if got := base.RequestUserID(r); got != "" {
			t.Fatalf("expected empty, got %q", got)
		}
	})
}

func TestRequestUserIDTrimsWhitespace(t *testing.T) {
	t.Parallel()

	base := NewBaseFromPrincipal(testPrincipalResolver(func(*http.Request) string { return "  user-1  " }, nil, nil))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := base.RequestUserID(r); got != "user-1" {
		t.Fatalf("expected %q, got %q", "user-1", got)
	}
}

func TestRequestContextAndUserIDReturnsContext(t *testing.T) {
	t.Parallel()

	base := NewBaseFromPrincipal(testPrincipalResolver(func(*http.Request) string { return "user-1" }, nil, nil))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx, userID := base.RequestContextAndUserID(r)
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
	if userID != "user-1" {
		t.Fatalf("expected %q, got %q", "user-1", userID)
	}
}

func TestNewTestBaseResolversReturnZeroValues(t *testing.T) {
	t.Parallel()

	base := NewTestBase()
	req := httptest.NewRequest(http.MethodGet, "/app/dashboard", nil)

	if got := base.RequestUserID(req); got != "" {
		t.Fatalf("RequestUserID() = %q, want empty", got)
	}
	if got := base.ResolveRequestLanguage(req); got != "" {
		t.Fatalf("ResolveRequestLanguage() = %q, want empty", got)
	}
	if got := base.ResolveRequestViewer(req); got != (module.Viewer{}) {
		t.Fatalf("ResolveRequestViewer() = %+v, want zero Viewer", got)
	}
}

func TestPageLocalizerUsesResolvedLanguage(t *testing.T) {
	t.Parallel()

	base := NewBaseFromPrincipal(testPrincipalResolver(nil, func(*http.Request) string { return "pt-BR" }, nil))
	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	rr := httptest.NewRecorder()

	loc, lang := base.PageLocalizer(rr, req)
	if loc == nil {
		t.Fatalf("PageLocalizer() localizer = nil, want non-nil")
	}
	if got := language.Make(lang); got != language.BrazilianPortuguese {
		t.Fatalf("PageLocalizer() lang = %q (%s), want %s", lang, got, language.BrazilianPortuguese)
	}
}

func TestRequestLocaleTagUsesResolvedLanguage(t *testing.T) {
	t.Parallel()

	base := NewBaseFromPrincipal(testPrincipalResolver(nil, func(*http.Request) string { return "pt-BR" }, nil))
	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	if got := base.RequestLocaleTag(req); got != language.BrazilianPortuguese {
		t.Fatalf("RequestLocaleTag() = %s, want %s", got, language.BrazilianPortuguese)
	}
}

func TestWriteNotFoundRendersAppErrorPage(t *testing.T) {
	t.Parallel()

	base := NewTestBase()
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/missing", nil)
	rr := httptest.NewRecorder()
	base.WriteNotFound(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	if body := rr.Body.String(); !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error marker: %q", body)
	}
}

func TestWriteErrorRendersStyledPageForBadRequest(t *testing.T) {
	t.Parallel()

	base := NewTestBase()
	req := httptest.NewRequest(http.MethodPost, "/app/settings/profile", nil)
	rr := httptest.NewRecorder()
	base.WriteError(rr, req, apperrors.E(apperrors.KindInvalidInput, "sensitive parse failure"))

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing styled error page marker: %q", body)
	}
	// Invariant: module error responses must not leak internal error strings.
	if strings.Contains(body, "sensitive parse failure") {
		t.Fatalf("body leaked internal text: %q", body)
	}
}

func TestWritePageRendersHTMXFragment(t *testing.T) {
	t.Parallel()

	base := NewTestBase()
	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()

	base.WritePage(rr, req, "Settings", http.StatusAccepted, nil, webtemplates.AppMainLayoutOptions{}, staticComponent(`<section id="module-fragment">ok</section>`))
	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusAccepted)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="module-fragment"`) {
		t.Fatalf("body missing fragment marker: %q", body)
	}
	if strings.Contains(strings.ToLower(body), "<!doctype html") || strings.Contains(strings.ToLower(body), "<html") {
		t.Fatalf("expected htmx fragment without full document wrapper")
	}
}

func TestWritePageFallsBackToWriteErrorWhenRenderFails(t *testing.T) {
	t.Parallel()

	base := NewTestBase()
	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	rr := httptest.NewRecorder()

	base.WritePage(rr, req, "Settings", http.StatusOK, nil, webtemplates.AppMainLayoutOptions{}, failingComponent{})
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
	if body := rr.Body.String(); !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error marker after write failure: %q", body)
	}
}

type staticComponent string

func (c staticComponent) Render(_ context.Context, w io.Writer) error {
	_, err := io.WriteString(w, string(c))
	return err
}

type failingComponent struct{}

func (failingComponent) Render(context.Context, io.Writer) error {
	return errors.New("render failed")
}
