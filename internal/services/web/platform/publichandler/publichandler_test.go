package publichandler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/pagerender"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
)

func TestIsViewerSignedInUsesSignedInResolver(t *testing.T) {
	t.Parallel()

	base := NewBase(
		WithResolveViewer(func(*http.Request) module.Viewer { return module.Viewer{DisplayName: "ignored"} }),
		WithResolveViewerSignedIn(func(*http.Request) bool { return true }),
	)
	if got := base.IsViewerSignedIn(httptest.NewRequest(http.MethodGet, "/", nil)); !got {
		t.Fatalf("IsViewerSignedIn() = %v, want true", got)
	}
}

func TestIsViewerSignedInReturnsFalseWithoutSignedInResolver(t *testing.T) {
	t.Parallel()

	base := NewBase(WithResolveViewer(func(*http.Request) module.Viewer { return module.Viewer{DisplayName: "signed in"} }))
	if got := base.IsViewerSignedIn(httptest.NewRequest(http.MethodGet, "/", nil)); got {
		t.Fatalf("IsViewerSignedIn() = %v, want false", got)
	}
}

func TestNewBaseFromPrincipalUsesGroupedPrincipalResolver(t *testing.T) {
	t.Parallel()

	base := NewBaseFromPrincipal(principal.NewPrincipal(
		nil,
		func(*http.Request) bool { return true },
		func(*http.Request) string { return "user-1" },
		func(*http.Request) string { return "en-CA" },
		func(*http.Request) module.Viewer { return module.Viewer{DisplayName: "Louis"} },
	))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if !base.IsViewerSignedIn(req) {
		t.Fatalf("IsViewerSignedIn() = false, want true")
	}
	if got := base.ResolveRequestViewer(req).DisplayName; got != "Louis" {
		t.Fatalf("ResolveRequestViewer().DisplayName = %q, want %q", got, "Louis")
	}
	if got := base.ResolveRequestLanguage(req); got != "en-CA" {
		t.Fatalf("ResolveRequestLanguage() = %q, want %q", got, "en-CA")
	}
	if got := base.RequestUserID(req); got != "user-1" {
		t.Fatalf("RequestUserID() = %q, want %q", got, "user-1")
	}
}

func TestResolveRequestViewerReturnsZeroWithoutResolver(t *testing.T) {
	t.Parallel()

	base := NewBase()
	if got := base.ResolveRequestViewer(httptest.NewRequest(http.MethodGet, "/", nil)); got != (module.Viewer{}) {
		t.Fatalf("ResolveRequestViewer() = %+v, want zero Viewer", got)
	}
}

func TestIsViewerSignedInNotBackedByDisplayName(t *testing.T) {
	t.Parallel()

	base := NewBase(
		WithResolveViewer(func(*http.Request) module.Viewer { return module.Viewer{DisplayName: ""} }),
		WithResolveViewerSignedIn(func(*http.Request) bool { return true }),
	)
	if !base.IsViewerSignedIn(httptest.NewRequest(http.MethodGet, "/", nil)) {
		t.Fatalf("IsViewerSignedIn() = false, want true")
	}
	if got := base.ResolveRequestViewer(httptest.NewRequest(http.MethodGet, "/", nil)); got.DisplayName != "" {
		t.Fatalf("ResolveRequestViewer() = %+v, want DisplayName=empty", got)
	}
}

func TestWriteNotFoundRendersPublicErrorPage(t *testing.T) {
	t.Parallel()

	base := NewBase()
	req := httptest.NewRequest(http.MethodGet, "/discover", nil)
	rr := httptest.NewRecorder()
	base.WriteNotFound(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app-error marker: %q", body)
	}
}

func TestWritePublicPageRendersAuthShell(t *testing.T) {
	t.Parallel()

	base := NewBase()
	req := httptest.NewRequest(http.MethodGet, "/discover", nil)
	rr := httptest.NewRecorder()
	base.WritePublicPage(rr, req, pagerender.PublicPage{Title: "Discover", MetaDesc: "desc", Language: "en", StatusCode: http.StatusAccepted})
	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusAccepted)
	}
	if body := rr.Body.String(); !strings.Contains(body, `id="auth-shell"`) {
		t.Fatalf("body missing auth-shell marker: %q", body)
	}
}

func TestPageLocalizerResolvesLanguageThroughEmbeddedResolver(t *testing.T) {
	t.Parallel()

	base := NewBase(
		WithResolveViewer(func(*http.Request) module.Viewer { return module.Viewer{} }),
		WithResolveUserID(func(*http.Request) string { return "user-1" }),
	)
	base.Base = base.Base.WithLanguage(func(*http.Request) string { return "pt-BR" })
	req := httptest.NewRequest(http.MethodGet, "/discover", nil)

	loc, lang := base.PageLocalizer(httptest.NewRecorder(), req)
	if loc == nil {
		t.Fatal("PageLocalizer() localizer = nil, want non-nil")
	}
	if lang != "pt-BR" {
		t.Fatalf("PageLocalizer() language = %q, want %q", lang, "pt-BR")
	}
	if got := base.RequestUserID(req); got != "user-1" {
		t.Fatalf("RequestUserID() = %q, want %q", got, "user-1")
	}
}

func TestRequestUserIDFailsClosedForNilInputs(t *testing.T) {
	t.Parallel()

	base := NewBase()
	if got := base.RequestUserID(nil); got != "" {
		t.Fatalf("RequestUserID(nil) = %q, want empty", got)
	}

	base = NewBase(WithResolveUserID(func(*http.Request) string { return " user-1 " }))
	if got := base.RequestUserID(nil); got != "" {
		t.Fatalf("RequestUserID(nil request with resolver) = %q, want empty", got)
	}

	base = NewBase()
	req := httptest.NewRequest(http.MethodGet, "/discover", nil)
	if got := base.RequestUserID(req); got != "" {
		t.Fatalf("RequestUserID(no resolver) = %q, want empty", got)
	}
}

func TestWriteErrorRendersPublicErrorPageForNotFound(t *testing.T) {
	t.Parallel()

	base := NewBase()
	req := httptest.NewRequest(http.MethodGet, "/u/missing", nil)
	rr := httptest.NewRecorder()
	base.WriteError(rr, req, apperrors.E(apperrors.KindNotFound, "missing profile"))

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app-error marker: %q", body)
	}
	// Invariant: public errors must not leak raw backend messages.
	if strings.Contains(body, "missing profile") {
		t.Fatalf("body leaked internal message: %q", body)
	}
}

func TestWriteErrorRendersStyledPageForBadRequest(t *testing.T) {
	t.Parallel()

	base := NewBase()
	req := httptest.NewRequest(http.MethodPost, "/passkeys/login/start", nil)
	rr := httptest.NewRecorder()
	base.WriteError(rr, req, apperrors.E(apperrors.KindInvalidInput, "unsafe parser detail"))

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing styled error page marker: %q", body)
	}
	if strings.Contains(body, "unsafe parser detail") {
		t.Fatalf("body leaked internal error text: %q", body)
	}
}

func TestWriteErrorAllowsNilWriter(t *testing.T) {
	t.Parallel()

	base := NewBase()
	req := httptest.NewRequest(http.MethodGet, "/discover", nil)
	base.WriteError(nil, req, apperrors.E(apperrors.KindInvalidInput, "invalid"))
}
